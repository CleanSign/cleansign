import { headers } from "next/headers";
import { notFound, redirect } from "next/navigation";
import Link from "next/link";
import { auth } from "@/db/auth";
import { getContractForUser } from "@/lib/contract-queries";
import { SEVERITY_CHIP, clauseLabel, formatWon } from "@/lib/format";
import { Tag } from "@/components/ui/tag";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-2 pt-5.5">
      <span className="text-[13px] font-bold text-ink">{title}</span>
      {children}
    </div>
  );
}

export default async function ClauseDetailPage({
  params,
}: {
  params: Promise<{ id: string; clauseId: string }>;
}) {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session) {
    redirect("/");
  }
  const { id, clauseId } = await params;
  const found = await getContractForUser(session.user.id, id);
  const clause = found?.clauses.find((c) => c.id === clauseId);
  if (!found || found.contract.status !== "completed" || !clause) notFound();

  return (
    <div className="flex min-h-screen flex-col items-center bg-bg">
      <div className="flex w-full flex-col px-5 pb-10 md:w-130 md:px-0">
        <div className="flex items-center gap-2 pb-1.5 pt-4.5 md:pt-10">
          <Link href={`/contracts/${id}`} aria-label="분석 결과로" className="flex h-9 w-9 items-center justify-center text-ink">
            <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M15 5l-7 7 7 7" />
            </svg>
          </Link>
          <span className="text-[14.5px] font-bold text-ink">{clauseLabel(clause.clauseType)}</span>
        </div>

        <Section title="계약서 원문">
          <blockquote className="rounded-2xl bg-surface px-4 py-4 text-[13.5px] leading-[1.7] text-ink">
            {clause.original}
          </blockquote>
        </Section>

        {clause.lossAmount > 0 && (
          <Section title="예상 손실">
            <div className="flex flex-col gap-1 rounded-2xl bg-surface px-4 py-4">
              <div className="flex items-center gap-2">
                <span className="text-[19px] font-extrabold tracking-tight text-ink">{formatWon(clause.lossAmount)}</span>
                <Tag tone={SEVERITY_CHIP[clause.severity].tone}>{SEVERITY_CHIP[clause.severity].label}</Tag>
              </div>
              <span className="text-[11.5px] leading-relaxed text-ink-3">
                분쟁이 생겼을 때 과거 조정 사례에서 인정된 비율을 반영한 참고용 추정이고, 칩은 보증금에 비한 손실 규모예요
              </span>
            </div>
          </Section>
        )}

        <Section title="왜 위험할 수 있나요">
          <p className="text-[13.5px] leading-[1.7] text-ink-2">{clause.reason}</p>
          {clause.lawReference && (
            <div className="flex flex-wrap gap-1.5 pt-1">
              {clause.lawReference.split(", ").map((l) => (
                <Tag key={l} tone="warn">
                  {l}
                </Tag>
              ))}
            </div>
          )}
        </Section>

        <Section title="이렇게 바꿔 보세요">
          <p className="rounded-2xl border border-mint/22 bg-mint/8 px-4 py-4 text-[13.5px] leading-[1.7] text-ink">
            {clause.safeText}
          </p>
        </Section>

        <p className="pt-6 text-[11px] leading-relaxed text-ink-4">
          법률 자문이 아니에요. 중요한 계약은 공인중개사나 변호사에게 확인해 보세요.
        </p>
      </div>
    </div>
  );
}
