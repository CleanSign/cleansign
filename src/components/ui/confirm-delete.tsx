"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { AlertDialog } from "@base-ui/react/alert-dialog";
import { Button } from "./button";

/** 계약서 삭제 버튼 + 확인창(Base UI AlertDialog). 지운 뒤 `redirectTo`로 이동한다. */
export function ConfirmDelete({
  contractId,
  redirectTo = "/contracts",
  children,
}: {
  contractId: string;
  redirectTo?: string;
  children: React.ReactNode;
}) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function remove() {
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(`/api/v1/contracts/${contractId}`, { method: "DELETE" });
      if (res.status !== 204 && res.status !== 404) throw new Error(String(res.status));
      setOpen(false);
      router.replace(redirectTo);
      router.refresh();
    } catch {
      setError("지우지 못했어요. 잠시 후 다시 시도해 주세요");
      setBusy(false);
    }
  }

  return (
    <AlertDialog.Root open={open} onOpenChange={(o) => !busy && setOpen(o)}>
      <AlertDialog.Trigger className="flex w-full items-center justify-center rounded-2xl bg-danger/10 py-3.5 text-[13.5px] font-bold text-danger focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-2">
        {children}
      </AlertDialog.Trigger>
      <AlertDialog.Portal>
        <AlertDialog.Backdrop className="fixed inset-0 bg-black/60" />
        <AlertDialog.Viewport className="fixed inset-0 flex items-center justify-center px-5">
          <AlertDialog.Popup className="flex w-full max-w-90 flex-col gap-4 rounded-[20px] bg-surface p-5.5 text-ink outline-none">
            <div className="flex flex-col gap-2">
              <AlertDialog.Title className="text-[17px] font-extrabold tracking-tight">이 계약서를 지울까요?</AlertDialog.Title>
              <AlertDialog.Description className="text-[13px] leading-relaxed text-ink-2">
                분석 결과도 함께 지워지고 되돌릴 수 없어요.
              </AlertDialog.Description>
            </div>
            {error && (
              <p role="alert" className="text-[12.5px] text-danger">
                {error}
              </p>
            )}
            <div className="flex gap-2.5">
              <AlertDialog.Close
                disabled={busy}
                className="flex-1 rounded-2xl bg-white/6 py-3.5 text-[14px] font-bold text-ink-2 data-[disabled]:opacity-30"
              >
                취소
              </AlertDialog.Close>
              <Button variant="danger" onClick={remove} disabled={busy} className="flex-1">
                {busy ? "지우는 중..." : "지우기"}
              </Button>
            </div>
          </AlertDialog.Popup>
        </AlertDialog.Viewport>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
