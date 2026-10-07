import { headers } from "next/headers";
import { after } from "next/server";
import { auth } from "@/db/auth";
import { db } from "@/db/db";
import { contract } from "@/db/schema";
import { runAnalysis } from "@/lib/analysis-job";
import { getContractPage } from "@/lib/contract-queries";
import { MAX_UPLOAD_BYTES } from "@/lib/contracts";

export async function POST(req: Request) {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }

  const form = await req.formData().catch(() => null);
  const file = form?.get("file");
  if (!(file instanceof File)) {
    return Response.json({ error: "file이 필요합니다" }, { status: 400 });
  }
  const isPdf = file.type === "application/pdf" || file.name.toLowerCase().endsWith(".pdf");
  if (!isPdf) {
    return Response.json({ error: "PDF 파일만 올릴 수 있어요" }, { status: 415 });
  }
  if (file.size === 0 || file.size > MAX_UPLOAD_BYTES) {
    return Response.json({ error: "파일 크기는 10MB 이하여야 해요" }, { status: 413 });
  }

  // 요청이 끝난 뒤에도 파일을 쓰므로 메모리로 확실히 복사해 둔다
  const bytes = await file.arrayBuffer();
  const upload = new File([bytes], file.name, { type: "application/pdf" });

  const [row] = await db
    .insert(contract)
    .values({
      userId: session.user.id,
      fileName: file.name,
      mimeType: "application/pdf",
      status: "pending",
      progress: { stage: "reading", done: 0, total: 0 },
    })
    .returning({ id: contract.id });

  // 응답을 먼저 보내고 분석은 이어서 실행한다. 클라이언트는 GET /contracts/{id}로 상태를 확인한다
  after(() => runAnalysis(row.id, upload));

  return Response.json({ id: row.id, status: "pending" }, { status: 202 });
}

/** 히스토리 목록. GET /api/v1/contracts?page=1&limit=20 (최신순, 실패한 분석 포함) */
export async function GET(req: Request) {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }
  const q = new URL(req.url).searchParams;
  const page = Math.max(1, Math.floor(Number(q.get("page"))) || 1);
  const limit = Math.min(50, Math.max(1, Math.floor(Number(q.get("limit"))) || 20));

  const { items, total, hasMore } = await getContractPage(session.user.id, page, limit);
  return Response.json({
    items: items.map((r) => ({
      id: r.id,
      status: r.status,
      fileName: r.fileName,
      createdAt: r.createdAt,
      lossAmount: r.lossAmount,
      riskCount: r.status === "completed" ? r.clauseCount : null,
      error: r.status === "failed" ? r.errorMessage : null,
    })),
    page,
    total,
    hasMore,
  });
}
