import { headers } from "next/headers";
import { auth } from "@/lib/auth";
import { deleteContract, getContractForUser } from "@/lib/contract-queries";
import { toResponse } from "@/lib/contracts";

export async function GET(_req: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }
  const { id } = await params;

  // 남의 계약서는 존재 여부도 알려 주지 않는다(404)
  const found = await getContractForUser(session.user.id, id);
  if (!found) {
    return Response.json({ error: "not found" }, { status: 404 });
  }
  // pending/failed도 200으로 상태를 돌려준다. 클라이언트는 status를 보고 폴링을 이어간다
  return Response.json(toResponse(found.contract, found.clauses));
}

/** 계약서와 분석 결과를 지운다. DELETE /api/v1/contracts/{id} → 204 */
export async function DELETE(_req: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }
  const { id } = await params;
  const deleted = await deleteContract(session.user.id, id);
  return deleted ? new Response(null, { status: 204 }) : Response.json({ error: "not found" }, { status: 404 });
}
