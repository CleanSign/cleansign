"use client";

import { Button as BaseButton } from "@base-ui/react/button";
import { cn } from "./cn";

type Variant = "primary" | "text";

const VARIANTS: Record<Variant, string> = {
  primary:
    "w-full rounded-2xl bg-accent py-4 text-[15px] font-bold text-white data-[disabled]:opacity-30",
  text: "self-center bg-transparent text-[12.5px] font-semibold text-ink-2 data-[disabled]:opacity-30",
};

export function Button({
  variant = "primary",
  className,
  ...props
}: BaseButton.Props & { variant?: Variant }) {
  return (
    <BaseButton
      type="button"
      className={cn(
        "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-2",
        VARIANTS[variant],
        typeof className === "string" && className,
      )}
      {...props}
    />
  );
}
