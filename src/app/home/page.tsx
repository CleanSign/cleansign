import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Link from "next/link";
import { auth } from "@/lib/auth";
import { ListRow } from "@/components/ui/list-row";
import { Avatar } from "@/components/ui/avatar";
import { HScroll } from "@/components/ui/h-scroll";
import { getRecentContracts } from "@/lib/contract-queries";
import { formatDate, stripPdf } from "@/lib/format";
import { Tag } from "@/components/ui/tag";
import { BrandMark } from "@/components/ui/brand";

const ARTICLES = [
  {
    tag: "법률 업데이트",
    tone: "accent" as const,
    title: "전세보증금 반환보증 한도, 이렇게 바뀌었어요",
    date: "2026.09.20",
  },
  {
    tag: "제도 안내",
    tone: "warn" as const,
    title: "임대차 신고제, 놓치면 과태료 나와요",
    date: "2026.09.10",
  },
  {
    tag: "안전 가이드",
    tone: "mint" as const,
    title: "전세사기 예방, 계약 전 체크리스트 5",
    date: "2026.08.28",
  },
];

function riskLabel(count: number) {
  if (count === 0) return { text: "위험 조항 없음", cls: "text-mint" };
  return { text: `위험 조항 ${count}건`, cls: count >= 3 ? "text-danger" : "text-warn" };
}

export default async function HomePage() {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session) {
    redirect("/");
  }
  const recent = await getRecentContracts(session.user.id, 3);

  return (
    <div className="flex min-h-screen flex-col items-center bg-bg">
      <div className="flex w-full flex-col px-5 md:w-155 md:px-0">
        <div className="flex items-center justify-between pb-2 pt-5.5 md:pt-10">
          <div className="flex items-center gap-2">
            <BrandMark size={20} />
            <span className="text-base font-extrabold tracking-tight text-ink">Clean Sign</span>
          </div>
          <div className="flex items-center gap-4">
            <a href="#" className="hidden text-[13px] font-semibold text-ink-2 md:inline">
              다른 계정으로 로그인
            </a>
            <Link href="/profile" aria-label="프로필">
              <Avatar size={34}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="8" r="3.5" />
                  <path d="M4.5 20c1.5-4 5-5.5 7.5-5.5s6 1.5 7.5 5.5" />
                </svg>
              </Avatar>
            </Link>
          </div>
        </div>

        <div className="pt-4.5 md:pt-5.5">
          <Link href="/analyze" className="flex items-center gap-3.5 rounded-[20px] bg-accent px-5 py-5 md:gap-4 md:px-6.5 md:py-6">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-white/16 text-white md:h-12.5 md:w-12.5">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 3l7 3v5c0 4.5-3 8-7 9-4-1-7-4.5-7-9V6l7-3z" />
                <path d="M9 12l2 2 4-4" />
              </svg>
            </div>
            <div className="flex grow flex-col gap-0.5">
              <span className="text-[15.5px] font-extrabold text-white md:text-[17px]">새 계약서 분석하기</span>
              <span className="text-xs font-medium text-white/75 md:text-[13px]">PDF로 올리면 위험 조항을 찾아드려요</span>
            </div>
            <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="#FFFFFF" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="shrink-0">
              <path d="M9 5l7 7-7 7" />
            </svg>
          </Link>
        </div>

        <div className="pt-6">
          <span className="text-[13px] font-bold text-ink">알아두면 좋은 소식</span>
          <HScroll className="-mx-5 px-5 pt-3 md:mx-0 md:px-0">
            {ARTICLES.map((a) => (
              <a
                key={a.title}
                href="#"
                className="flex w-57 shrink-0 flex-col gap-2.5 rounded-2xl bg-surface p-4"
              >
                <Tag tone={a.tone}>{a.tag}</Tag>
                <span className="text-[13.5px] font-bold leading-snug text-ink">{a.title}</span>
                <span className="text-[11px] text-ink-3">{a.date}</span>
              </a>
            ))}
          </HScroll>
        </div>

        <div className="pt-4">
          <a
            href="#"
            className="flex items-center gap-3.5 rounded-[18px] border border-mint/22 bg-mint/8 px-4.5 py-4.5"
          >
            <div className="flex h-10.5 w-10.5 shrink-0 items-center justify-center rounded-xl bg-mint/16 text-mint">
              <svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="9" />
                <path d="M9.5 9a2.5 2.5 0 015 0c0 1.8-2.2 2-2.5 3.5" />
                <circle cx="12" cy="16.3" r="0.5" fill="currentColor" stroke="none" />
              </svg>
            </div>
            <div className="flex grow flex-col gap-0.5">
              <span className="text-[14.5px] font-extrabold text-ink">전문가와 상담하기</span>
              <span className="text-[11.5px] font-medium text-ink-2">공인중개사·변호사 상담 1건 19,000원부터</span>
            </div>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#2DD9A8" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="shrink-0">
              <path d="M9 5l7 7-7 7" />
            </svg>
          </a>
        </div>

        <div className="flex items-center justify-between pt-5.5">
          <span className="text-[13px] font-bold text-ink">최근 분석</span>
          <Link href="/contracts" className="text-[12.5px] font-semibold text-accent-2">
            전체보기
          </Link>
        </div>

        <div className="flex flex-col gap-2.5 pt-3 pb-9">
          {recent.length === 0 && (
            <div className="rounded-2xl bg-surface px-4 py-5 text-center text-[12.5px] leading-relaxed text-ink-3">
              아직 분석한 계약서가 없어요.
              <br />
              위의 &lsquo;새 계약서 분석하기&rsquo;로 시작해 보세요
            </div>
          )}
          {recent.map((r) => {
            const risk =
              r.status === "pending" ? { text: "분석 중", cls: "text-accent-2" } : riskLabel(r.clauseCount);
            return (
              <ListRow
                key={r.id}
                href={`/contracts/${r.id}`}
                icon={
                  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M7 3h7l4 4v14H7z" />
                    <path d="M14 3v4h4" />
                    <path d="M9.5 13h5M9.5 17h5" />
                  </svg>
                }
                title={stripPdf(r.fileName)}
                sub={formatDate(r.createdAt)}
                trailing={<span className={`whitespace-nowrap text-xs font-bold ${risk.cls}`}>{risk.text}</span>}
              />
            );
          })}
        </div>
      </div>
    </div>
  );
}
