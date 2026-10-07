import "server-only";
import { eq } from "drizzle-orm";
import { db } from "@/db/db";
import { contract, contractDangerClause } from "@/db/schema";
import { AiServiceError, analyzeContract } from "@/lib/ai-client";
import { toClauseRows, toResultFields } from "@/lib/contracts";

/**
 * 업로드된 계약서를 분석해 결과를 저장한다. 요청 응답을 보낸 뒤 `after()`에서 실행된다.
 * 파일은 메모리로만 들고 있으므로 서버가 중간에 내려가면 이 작업은 사라진다:
 * 그때 남은 pending 행은 조회 시점에 STALE_MS가 지나면 failed로 바꾼다(contract-queries).
 */
export async function runAnalysis(contractId: string, file: File) {
  const fail = (message: string) =>
    db
      .update(contract)
      .set({ status: "failed", errorMessage: message, progress: null })
      .where(eq(contract.id, contractId));

  try {
    const analysis = await analyzeContract(file, async (progress) => {
      await db.update(contract).set({ progress }).where(eq(contract.id, contractId));
    });

    await db.transaction(async (tx) => {
      const updated = await tx
        .update(contract)
        .set({ ...toResultFields(analysis), progress: null, errorMessage: null })
        .where(eq(contract.id, contractId))
        .returning({ id: contract.id });
      // 분석하는 동안 사용자가 지운 경우: 결과를 버린다
      if (updated.length && analysis.clauses.length) {
        await tx.insert(contractDangerClause).values(toClauseRows(contractId, analysis));
      }
    });
  } catch (e) {
    if (e instanceof AiServiceError) {
      await fail(e.message);
    } else {
      console.error("[analysis-job]", contractId, e);
      await fail("분석을 저장하는 중 문제가 생겼어요");
    }
  }
}
