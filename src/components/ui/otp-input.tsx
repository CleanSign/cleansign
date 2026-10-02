"use client";

import { OTPField } from "@base-ui/react/otp-field";

export function OtpInput({
  label,
  value,
  onValueChange,
  length = 6,
}: {
  label: string;
  value: string;
  onValueChange: (value: string) => void;
  length?: number;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[12.5px] font-semibold text-ink-2">{label}</span>
      <OTPField.Root
        length={length}
        value={value}
        onValueChange={onValueChange}
        validationType="numeric"
        inputMode="numeric"
        autoComplete="one-time-code"
        className="flex gap-2"
      >
        {Array.from({ length }, (_, i) => (
          <OTPField.Input
            key={i}
            aria-label={`${label} ${i + 1}번째 자리`}
            className="box-border h-14 min-w-0 flex-1 rounded-2xl border-[1.5px] border-line bg-surface-soft text-center text-lg font-bold text-ink focus:border-accent focus:outline-none"
          />
        ))}
      </OTPField.Root>
    </div>
  );
}
