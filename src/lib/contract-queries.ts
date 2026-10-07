import "server-only";
import { and, desc, eq, inArray } from "drizzle-orm";
import { db } from "@/db/db";
import { contract, contractDangerClause } from "@/db/schema";

// 분석 작업은 서버 프로세스 안에서 돈다. 이 시간이 지나도 pending이면 작업이 사라진 것으로 본다.
const STALE_MS = 15 * 60 * 1000;

/** 본인 소유의 분석만 돌려준다(남의 것이나 없는 것은 null). 상태는 pending/completed/failed 모두 포함한다. */
export async function getContractForUser(userId: string, id: string) {
  let [row] = await db
    .select()
    .from(contract)
    .where(and(eq(contract.id, id), eq(contract.userId, userId)));
  if (!row) return null;

  if (row.status === "pending" && Date.now() - row.createdAt.getTime() > STALE_MS) {
    [row] = await db
      .update(contract)
      .set({ status: "failed", errorMessage: "분석이 중단됐어요. 다시 올려 주세요", progress: null })
      .where(and(eq(contract.id, id), eq(contract.status, "pending")))
      .returning();
    row ??= (await db.select().from(contract).where(eq(contract.id, id)))[0];
  }

  const clauses =
    row.status === "completed"
      ? await db.select().from(contractDangerClause).where(eq(contractDangerClause.contractId, id))
      : [];
  clauses.sort((a, b) => a.orderIndex - b.orderIndex);
  return { contract: row, clauses };
}

export async function getRecentContracts(userId: string, limit: number) {
  const rows = await db
    .select({
      id: contract.id,
      fileName: contract.fileName,
      createdAt: contract.createdAt,
      status: contract.status,
      lossAmount: contract.lossAmount,
      clauseCount: db.$count(contractDangerClause, eq(contractDangerClause.contractId, contract.id)),
    })
    .from(contract)
    .where(and(eq(contract.userId, userId), inArray(contract.status, ["completed", "pending"])))
    .orderBy(desc(contract.createdAt))
    .limit(limit);
  return rows;
}

export const HISTORY_PAGE_SIZE = 20;

/** 히스토리 목록(최신순). 실패한 분석도 포함한다: 지우거나 다시 올릴 수 있어야 하기 때문이다. */
export async function getContractPage(userId: string, page: number, size = HISTORY_PAGE_SIZE) {
  const where = eq(contract.userId, userId);
  const [items, total] = await Promise.all([
    db
      .select({
        id: contract.id,
        fileName: contract.fileName,
        createdAt: contract.createdAt,
        status: contract.status,
        errorMessage: contract.errorMessage,
        lossAmount: contract.lossAmount,
        clauseCount: db.$count(contractDangerClause, eq(contractDangerClause.contractId, contract.id)),
      })
      .from(contract)
      .where(where)
      .orderBy(desc(contract.createdAt))
      .limit(size)
      .offset((page - 1) * size),
    db.$count(contract, where),
  ]);
  return { items, total, page, size, hasMore: page * size < total };
}

/** 본인 계약서를 지운다(조항은 FK cascade). 지운 행이 있으면 true. */
export async function deleteContract(userId: string, id: string) {
  const rows = await db
    .delete(contract)
    .where(and(eq(contract.id, id), eq(contract.userId, userId)))
    .returning({ id: contract.id });
  return rows.length > 0;
}
