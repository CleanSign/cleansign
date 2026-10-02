import { cn } from "./cn";

const TONES = {
  accent: "bg-accent/14 text-accent-2",
  warn: "bg-warn/14 text-warn",
  mint: "bg-mint/14 text-mint",
} as const;

export function Tag({
  tone = "accent",
  children,
}: {
  tone?: keyof typeof TONES;
  children: React.ReactNode;
}) {
  return (
    <span className={cn("self-start rounded-full px-2.5 py-0.5 text-[10.5px] font-bold", TONES[tone])}>
      {children}
    </span>
  );
}
