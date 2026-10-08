import { headers } from "next/headers";
import { auth } from "@/lib/auth";
import { getContractForUser } from "@/lib/contract-queries";

/** 조항 상세. GET /api/v1/contracts/{id}/clauses/{clauseId} */
export async function GET(_req: Request, { params }: { params: Promise<{ id: string; clauseId: string }> }) {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }
  const { id, clauseId } = await params;

  const found = await getContractForUser(session.user.id, id);
  const clause = found?.contract.status === "completed" ? found.clauses.find((c) => c.id === clauseId) : undefined;
  if (!found || !clause) {
    return Response.json({ error: "not found" }, { status: 404 });
  }

  return Response.json({
    id: clause.id,
    contractId: id,
    clauseType: clause.clauseType,
    original: clause.original,
    lossAmount: clause.lossAmount,
    severity: clause.severity,
    confidence: clause.confidence,
    reason: clause.reason,
    lawReference: clause.lawReference ? clause.lawReference.split(", ") : [],
    safeText: clause.safeText,
  });
}
