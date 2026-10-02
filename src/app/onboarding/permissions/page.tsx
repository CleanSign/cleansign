"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { BrandHeader } from "@/components/ui/brand";
import { Button } from "@/components/ui/button";

function PermCard({
  icon,
  title,
  tag,
  tagColor,
  desc,
}: {
  icon: React.ReactNode;
  title: string;
  tag: string;
  tagColor: string;
  desc: string;
}) {
  return (
    <div className="flex items-start gap-3.5 rounded-2xl bg-surface px-4.5 py-4">
      <div
        className={`flex h-9.5 w-9.5 shrink-0 items-center justify-center rounded-xl ${
          tagColor === "text-danger" ? "bg-accent/14 text-accent-2" : "bg-white/6 text-ink-2"
        }`}
      >
        {icon}
      </div>
      <div className="flex flex-col gap-0.5">
        <span className="text-[13.5px] font-bold text-ink">
          {title} <span className={`font-bold ${tagColor}`}>· {tag}</span>
        </span>
        <span className="text-xs leading-snug text-ink-2">{desc}</span>
      </div>
    </div>
  );
}

export default function PermissionsPage() {
  const router = useRouter();

  useEffect(() => {
    if (typeof window !== "undefined" && "Notification" in window) {
      Notification.requestPermission();
    }
  }, []);

  return (
    <div className="flex min-h-screen flex-col bg-bg md:items-center md:justify-center">
      <div className="flex w-full grow flex-col px-5 pt-6 md:w-110 md:grow-0 md:px-0 md:pt-0">
        <BrandHeader className="pb-9" />

        <div className="pb-7 text-center text-[21px] font-extrabold leading-snug tracking-tight text-ink md:text-2xl">
          원활한 이용을 위해
          <br className="md:hidden" /> 권한이 필요해요
        </div>

        <div className="flex flex-col gap-3">
          <div className="md:hidden">
            <PermCard
              icon={
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M4 8.5A1.5 1.5 0 015.5 7h2l1-2h7l1 2h2A1.5 1.5 0 0120 8.5v10A1.5 1.5 0 0118.5 20h-13A1.5 1.5 0 014 18.5z" />
                  <circle cx="12" cy="13" r="3.5" />
                </svg>
              }
              title="카메라"
              tag="필수"
              tagColor="text-danger"
              desc="계약서를 촬영해 분석하기 위해 필요해요"
            />
          </div>
          <div className="md:hidden">
            <PermCard
              icon={
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3" y="4" width="18" height="16" rx="2" />
                  <circle cx="8.5" cy="9.5" r="1.5" />
                  <path d="M21 15l-5-5-9 9" />
                </svg>
              }
              title="사진 보관함"
              tag="선택"
              tagColor="text-ink-3"
              desc="저장된 계약서 사진을 불러오기 위해 필요해요"
            />
          </div>
          <div className="hidden md:block">
            <PermCard
              icon={
                <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 16V6" />
                  <path d="M8 10l4-4 4 4" />
                  <path d="M5 16.5V18a2 2 0 002 2h10a2 2 0 002-2v-1.5" />
                </svg>
              }
              title="파일 업로드"
              tag="필수"
              tagColor="text-danger"
              desc="계약서 파일(PDF · 이미지)을 업로드해 분석하기 위해 필요해요"
            />
          </div>
          <PermCard
            icon={
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <path d="M6 10a6 6 0 0112 0c0 4 1.5 5.5 1.5 5.5H4.5S6 14 6 10z" />
                <path d="M9.5 18a2.5 2.5 0 005 0" />
              </svg>
            }
            title="알림"
            tag="선택"
            tagColor="text-ink-3"
            desc="분석 완료와 상담 알림을 보내드리기 위해 필요해요"
          />
        </div>

        <div className="mt-auto pt-6 md:mt-7">
          <Button onClick={() => router.push("/onboarding/signup")}>확인하고 시작하기</Button>
        </div>
        <div className="pb-10 pt-3.5 md:pb-0">
          <p className="text-center text-[11px] leading-relaxed text-ink-4">
            권한은 기기 설정에서 언제든 변경할 수 있어요
          </p>
        </div>
      </div>
    </div>
  );
}
