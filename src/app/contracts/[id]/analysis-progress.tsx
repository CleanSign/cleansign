"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

type Progress = { stage: string; done: number; total: number } | null;

// 서버가 알려 주는 실제 처리 단계. 진행률 막대가 아니라 지금 하는 일을 보여 준다.
const STAGES = ["reading", "ocr", "splitting", "classifying", "scoring"] as const;
const LABEL: Record<string, string> = {
  reading: "계약서를 읽는 중",
  ocr: "스캔한 계약서를 글자로 바꾸는 중",
  splitting: "조항을 나누는 중",
  classifying: "조항을 하나씩 살펴보는 중",
  scoring: "예상 손실을 계산하는 중",
};
const POLL_MS = 1500;

export function AnalysisProgress({ id, initial }: { id: string; initial: Progress }) {
  const router = useRouter();
  const [progress, setProgress] = useState<Progress>(initial);
  const [networkError, setNetworkError] = useState(false);

  useEffect(() => {
    let stopped = false;
    async function tick() {
      try {
        const res = await fetch(`/api/v1/contracts/${id}`, { cache: "no-store" });
        if (res.status === 401) return router.replace("/");
        const data = await res.json();
        if (stopped) return;
        setNetworkError(false);
        if (data.status === "pending") {
          setProgress(data.progress);
        } else {
          router.refresh(); // completed 또는 failed: 서버 컴포넌트가 결과/오류 화면을 그린다
          return;
        }
      } catch {
        if (!stopped) setNetworkError(true);
      }
      if (!stopped) timer = setTimeout(tick, POLL_MS);
    }
    let timer = setTimeout(tick, POLL_MS);
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [id, router]);

  const stage = progress?.stage ?? "reading";
  const index = Math.max(0, STAGES.indexOf(stage as (typeof STAGES)[number]));
  const hasTotal = !!progress && progress.total > 0;
  const counter = stage === "classifying" && hasTotal ? `${progress.done} / ${progress.total}개 조항을 확인했어요` : stage === "ocr" && hasTotal ? `${progress.done} / ${progress.total}쪽을 읽었어요` : null;

  return (
    <div className="flex flex-col items-center gap-6 pt-24" role="status" aria-live="polite">
      <div className="h-11 w-11 animate-spin rounded-full border-[3px] border-white/16 border-t-accent" />
      <div className="flex flex-col items-center gap-2">
        <span className="text-[17px] font-extrabold tracking-tight text-ink">{LABEL[stage] ?? LABEL.reading}</span>
        <span className="text-[12.5px] text-ink-2">
          {counter ?? "계약서 길이에 따라 시간이 걸릴 수 있어요"}
        </span>
      </div>
      <ol className="flex gap-1.5" aria-hidden>
        {STAGES.map((s, i) => (
          <li key={s} className={`h-1.5 w-1.5 rounded-full ${i <= index ? "bg-accent" : "bg-white/16"}`} />
        ))}
      </ol>
      <p className="max-w-80 text-center text-[11.5px] leading-relaxed text-ink-4">
        {networkError
          ? "연결이 잠시 끊겼어요. 다시 연결하는 중이에요"
          : "이 화면을 나가도 분석은 계속돼요. 홈의 최근 분석에서 다시 열 수 있어요"}
      </p>
    </div>
  );
}
