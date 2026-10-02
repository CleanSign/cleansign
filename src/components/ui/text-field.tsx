"use client";

import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";

export function TextField({
  label,
  ...inputProps
}: { label: string } & Omit<Input.Props, "className">) {
  return (
    <Field.Root className="flex flex-col gap-1.5">
      <Field.Label className="text-[12.5px] font-semibold text-ink-2">{label}</Field.Label>
      <Field.Control
        className="box-border w-full rounded-2xl border-[1.5px] border-line bg-surface-soft px-4 py-3.75 text-[15px] text-ink placeholder:text-ink-4 focus:border-accent focus:outline-none"
        {...inputProps}
      />
    </Field.Root>
  );
}
