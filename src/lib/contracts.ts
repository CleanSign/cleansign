import type { AiAnalysis } from "@/lib/ai-client";
import type { contract, contractDangerClause } from "@/db/schema";

const INT_MAX = 2_147_483_647; // integer 컬럼 상한

export const clampInt = (n: number | null | undefined) =>
  n == null ? null : Math.max(0, Math.min(INT_MAX, Math.round(n)));

export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

/** 분석 결과로 채우는 contract 컬럼들 (파일 이름 등 업로드 시점에 정해진 값은 포함하지 않는다) */
export function toResultFields(a: AiAnalysis): Partial<typeof contract.$inferInsert> {
  return {
    status: "completed",
    lossAmount: clampInt(a.lossAmount) ?? 0,
    riskScore: a.riskScore,
    grade: a.grade,
    lossMin: clampInt(a.lossRange[0]),
    lossMax: clampInt(a.lossRange[1]),
    depositAmount: clampInt(a.facts.deposit),
    modelVersion: `${a.modelVersion} / ${a.policyVersion}`,
    warnings: a.warnings,
    riskFlow: a.flow ?? null,
  };
}

export function toClauseRows(contractId: string, a: AiAnalysis): (typeof contractDangerClause.$inferInsert)[] {
  return a.clauses.map((c, i) => ({
    contractId,
    original: c.original,
    clauseType: c.clauseType,
    confidence: c.confidence,
    lMax: clampInt(c.lMax),
    lossAmount: clampInt(c.expectedLoss) ?? 0,
    severity: c.severity,
    reason: c.reason,
    lawReference: c.lawReference.join(", ") || null,
    safeText: c.safeText,
    orderIndex: i,
  }));
}

type ContractRow = typeof contract.$inferSelect;
type ClauseRow = typeof contractDangerClause.$inferSelect;

/** API 응답 모양. docs/02-api-ui-design.md의 { id, riskItems, lossAmount, flow, ... } */
export function toResponse(c: ContractRow, clauses: ClauseRow[]) {
  return {
    id: c.id,
    status: c.status,
    progress: c.status === "pending" ? c.progress : null,
    error: c.status === "failed" ? c.errorMessage : null,
    fileName: c.fileName,
    createdAt: c.createdAt,
    riskScore: c.riskScore,
    grade: c.grade,
    lossAmount: c.lossAmount,
    lossRange: c.lossMin != null && c.lossMax != null ? [c.lossMin, c.lossMax] : null,
    depositAmount: c.depositAmount,
    flow: c.riskFlow,
    warnings: c.warnings ?? [],
    riskItems: clauses
      .sort((a, b) => a.orderIndex - b.orderIndex)
      .map((r) => ({
        id: r.id,
        original: r.original,
        clauseType: r.clauseType,
        confidence: r.confidence,
        severity: r.severity,
        lossAmount: r.lossAmount,
        reason: r.reason,
        lawReference: r.lawReference,
        safeText: r.safeText,
      })),
  };
}
