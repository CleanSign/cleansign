"use client";

import { useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";

const MAX_BYTES = 10 * 1024 * 1024;
const mb = (n: number) => `${(n / 1024 / 1024).toFixed(1)}MB`;

export function UploadForm() {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function pick(f: File | undefined) {
    setError(null);
    if (!f) return;
    if (!(f.type === "application/pdf" || f.name.toLowerCase().endsWith(".pdf"))) {
      setFile(null);
      setError("PDF 파일만 올릴 수 있어요");
      return;
    }
    if (f.size > MAX_BYTES) {
      setFile(null);
      setError("파일 크기는 10MB 이하여야 해요");
      return;
    }
    setFile(f);
  }

  async function submit() {
    if (!file) return;
    setError(null);
    setBusy(true);
    try {
      const body = new FormData();
      body.append("file", file);
      const res = await fetch("/api/v1/contracts", { method: "POST", body });
      const data = await res.json().catch(() => null);
      if (res.status !== 202) {
        setError(data?.error ?? "분석에 실패했어요. 잠시 후 다시 시도해 주세요");
        setBusy(false);
        return;
      }
      router.push(`/contracts/${data.id}`);
    } catch {
      setError("네트워크 연결을 확인하고 다시 시도해 주세요");
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-5.5 pt-5">
      <div className="flex flex-col gap-2">
        <span className="text-[22px] font-extrabold tracking-tight text-ink">계약서를 올려 주세요</span>
        <span className="text-[13.5px] font-medium leading-relaxed text-ink-2">
          PDF로 올리면 위험한 조항을 찾아 예상 손실을 알려드려요
        </span>
      </div>

      <input
        ref={inputRef}
        id="contract-file"
        type="file"
        accept="application/pdf,.pdf"
        className="sr-only"
        onChange={(e) => pick(e.target.files?.[0])}
      />
      <label
        htmlFor="contract-file"
        className="flex cursor-pointer flex-col items-center gap-3 rounded-[18px] border-[1.5px] border-dashed border-line bg-surface-soft px-6 py-10 text-center focus-within:border-accent has-[:focus-visible]:border-accent"
      >
        <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-accent/14 text-accent-2">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 16V6" />
            <path d="M8 10l4-4 4 4" />
            <path d="M5 16.5V18a2 2 0 002 2h10a2 2 0 002-2v-1.5" />
          </svg>
        </div>
        {file ? (
          <>
            <span className="break-all text-[14.5px] font-bold text-ink">{file.name}</span>
            <span className="text-[12px] text-ink-3">{mb(file.size)} · 눌러서 다른 파일 고르기</span>
          </>
        ) : (
          <>
            <span className="text-[14.5px] font-bold text-ink">PDF 파일 고르기</span>
            <span className="text-[12px] text-ink-3">10MB 이하 · 글자가 들어 있는 PDF만 가능해요</span>
          </>
        )}
      </label>

      {error && (
        <p role="alert" className="text-center text-sm text-danger">
          {error}
        </p>
      )}

      <Button disabled={!file || busy} onClick={submit}>
        {busy ? "올리는 중..." : "분석하기"}
      </Button>
      <p className="text-center text-[11.5px] leading-relaxed text-ink-4">
        스캔한 계약서는 글자로 바꾸느라 시간이 더 걸려요
      </p>
    </div>
  );
}
