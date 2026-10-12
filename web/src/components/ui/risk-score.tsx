import { GRADE_CHIP } from "@/lib/format";
import { cn } from "./cn";

// Outlined and squarer than Tag on purpose: the grade is contract risk, not the per-clause loss-size chip.
const TONES = {
  danger: "border-danger/45 text-danger",
  warn: "border-warn/45 text-warn",
  neutral: "border-white/20 text-ink-2",
} as const;

export function GradeChip({ grade }: { grade: string }) {
  const chip = GRADE_CHIP[grade];
  if (!chip) return null;
  return (
    <span
      className={cn(
        "inline-flex items-center whitespace-nowrap rounded-md border px-2 py-0.5 text-[11.5px] font-bold",
        TONES[chip.tone],
      )}
    >
      {chip.label}
    </span>
  );
}

/** Overall contract risk score (0-100) and grade. Each part is hidden when its value is null. */
export function RiskScore({ score, grade }: { score: number | null; grade: string | null }) {
  if (score === null && grade === null) return null;
  const shown = score === null ? null : Math.min(100, Math.max(0, Math.round(score)));

  return (
    <div className="flex flex-col gap-2 rounded-[18px] bg-surface px-4.5 py-5">
      <span className="text-[12.5px] font-semibold text-ink-2">계약 위험도</span>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        {shown !== null && (
          <span className="flex items-baseline gap-1" aria-label={`100점 중 ${shown}점`}>
            <span className="text-[30px] font-extrabold leading-none tracking-tight text-ink">{shown}</span>
            <span className="text-[13px] font-semibold text-ink-3">/ 100</span>
          </span>
        )}
        {grade !== null && <GradeChip grade={grade} />}
      </div>
      <span className="text-[11.5px] leading-relaxed text-ink-3">대략적인 추정이며 법률 자문이 아니에요.</span>
    </div>
  );
}
