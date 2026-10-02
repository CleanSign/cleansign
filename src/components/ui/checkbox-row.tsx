"use client";

import { Checkbox } from "@base-ui/react/checkbox";

export function CheckboxRow({
  children,
  ...props
}: Omit<Checkbox.Root.Props, "className" | "children"> & { children: React.ReactNode }) {
  return (
    <label className="flex cursor-pointer items-center gap-3">
      <Checkbox.Root
        className="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-md border-2 border-white/20 bg-transparent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-2 data-[checked]:border-accent data-[checked]:bg-accent data-[indeterminate]:border-accent data-[indeterminate]:bg-accent"
        {...props}
      >
        <Checkbox.Indicator className="group flex text-white">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
            <path className="group-data-[indeterminate]:hidden" d="M5 12l4 4 10-10" />
            <path className="hidden group-data-[indeterminate]:block" d="M6 12h12" />
          </svg>
        </Checkbox.Indicator>
      </Checkbox.Root>
      {children}
    </label>
  );
}
