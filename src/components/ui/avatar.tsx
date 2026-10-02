"use client";

import { Avatar as BaseAvatar } from "@base-ui/react/avatar";
import { cn } from "./cn";

export function Avatar({
  src,
  size = 36,
  tone = "neutral",
  className,
  children,
}: {
  src?: string;
  size?: number;
  tone?: "neutral" | "accent";
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <BaseAvatar.Root
      style={{ width: size, height: size }}
      className={cn(
        "inline-flex flex-shrink-0 items-center justify-center overflow-hidden rounded-full align-middle",
        tone === "accent" ? "bg-accent/16 text-accent-2" : "bg-surface text-ink-2",
        className,
      )}
    >
      {src && <BaseAvatar.Image src={src} alt="" className="h-full w-full object-cover" />}
      <BaseAvatar.Fallback className="flex items-center justify-center">{children}</BaseAvatar.Fallback>
    </BaseAvatar.Root>
  );
}
