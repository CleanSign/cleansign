"use client";

import { ScrollArea } from "@base-ui/react/scroll-area";

export function HScroll({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <ScrollArea.Root className={className}>
      <ScrollArea.Viewport className="overflow-x-auto pb-1 [scrollbar-width:none]">
        <ScrollArea.Content className="flex gap-3">{children}</ScrollArea.Content>
      </ScrollArea.Viewport>
    </ScrollArea.Root>
  );
}
