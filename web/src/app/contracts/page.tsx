import Link from "next/link";
import { getContractPage, requireUser } from "@/lib/api/server";
import { formatDate, formatWon, stripPdf } from "@/lib/format";
import { ListRow } from "@/components/ui/list-row";

const DOC_ICON = (
  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
    <path d="M7 3h7l4 4v14H7z" />
    <path d="M14 3v4h4" />
    <path d="M9.5 13h5M9.5 17h5" />
  </svg>
);

export default async function HistoryPage({ searchParams }: { searchParams: Promise<{ page?: string }> }) {
  await requireUser();
  const page = Math.max(1, Math.floor(Number((await searchParams).page)) || 1);
  const { items, total, hasMore } = await getContractPage(page);

  return (
    <div className="flex min-h-screen flex-col items-center bg-bg">
      <div className="flex w-full flex-col px-5 md:w-155 md:px-0">
        <div className="flex items-center gap-2 pb-1.5 pt-4.5 md:pt-10">
          <Link href="/home" aria-label="홈으로" className="flex h-9 w-9 items-center justify-center text-ink">
            <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M15 5l-7 7 7 7" />
            </svg>
          </Link>
          <span className="text-[14.5px] font-bold text-ink">분석 기록</span>
          <span className="text-[12px] text-ink-3">{total}건</span>
        </div>

        <div className="flex flex-col gap-2.5 pt-3.5 pb-6">
          {items.length === 0 && (
            <div className="rounded-2xl bg-surface px-4 py-8 text-center text-[12.5px] leading-relaxed text-ink-3">
              {page > 1 ? "이 페이지에는 기록이 없어요" : "아직 분석한 계약서가 없어요"}
            </div>
          )}
          {items.map((r) => {
            const trailing =
              r.status === "pending" ? (
                <span className="whitespace-nowrap text-xs font-bold text-accent-2">분석 중</span>
              ) : r.status === "failed" ? (
                <span className="whitespace-nowrap text-xs font-bold text-danger">분석 실패</span>
              ) : (
                <div className="flex flex-col items-end gap-0.5">
                  <span className={`whitespace-nowrap text-xs font-bold ${(r.riskCount ?? 0) === 0 ? "text-mint" : "text-warn"}`}>
                    {(r.riskCount ?? 0) === 0 ? "위험 조항 없음" : `위험 조항 ${r.riskCount}건`}
                  </span>
                  {r.lossAmount > 0 && <span className="text-[11px] text-ink-3">{formatWon(r.lossAmount)}</span>}
                </div>
              );
            return (
              <ListRow
                key={r.id}
                href={`/contracts/${r.id}`}
                icon={DOC_ICON}
                title={stripPdf(r.fileName)}
                sub={r.status === "failed" && r.error ? r.error : formatDate(new Date(r.createdAt))}
                trailing={trailing}
              />
            );
          })}
        </div>

        {(page > 1 || hasMore) && (
          <nav aria-label="페이지" className="flex items-center justify-between pb-9 text-[13px] font-semibold">
            {page > 1 ? (
              <Link href={`/contracts?page=${page - 1}`} className="text-accent-2">
                이전
              </Link>
            ) : (
              <span />
            )}
            <span className="text-ink-3">{page}페이지</span>
            {hasMore ? (
              <Link href={`/contracts?page=${page + 1}`} className="text-accent-2">
                다음
              </Link>
            ) : (
              <span />
            )}
          </nav>
        )}
      </div>
    </div>
  );
}
