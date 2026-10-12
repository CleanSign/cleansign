import Link from "next/link";
import { cn } from "./cn";

const TONES = {
  neutral: "bg-white/6 text-ink-2",
  accent: "bg-accent/14 text-accent-2",
} as const;

export function IconTile({
  tone = "neutral",
  children,
}: {
  tone?: keyof typeof TONES;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px]", TONES[tone])}>
      {children}
    </div>
  );
}

export function ListRow({
  href,
  icon,
  tone,
  title,
  badge,
  sub,
  trailing,
}: {
  href: string;
  icon: React.ReactNode;
  tone?: keyof typeof TONES;
  title: string;
  /** Small tag shown next to the title, such as "for reference only" */
  badge?: React.ReactNode;
  sub?: string;
  trailing?: React.ReactNode;
}) {
  return (
    <Link href={href} className="flex items-center gap-3 rounded-2xl bg-surface px-4 py-3.5">
      <IconTile tone={tone}>{icon}</IconTile>
      <div className="flex min-w-0 grow flex-col gap-0.5">
        <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
          <span className="text-[13.5px] font-bold text-ink">{title}</span>
          {badge && <span>{badge}</span>}
        </div>
        {sub && <span className="break-words text-[11.5px] text-ink-3">{sub}</span>}
      </div>
      {trailing ?? (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#52565E" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M9 5l7 7-7 7" />
        </svg>
      )}
    </Link>
  );
}
