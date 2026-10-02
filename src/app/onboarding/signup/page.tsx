"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { authClient } from "@/lib/auth-client";
import { Form } from "@base-ui/react/form";
import { BrandHeader } from "@/components/ui/brand";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/text-field";
import { OtpInput } from "@/components/ui/otp-input";
import { getPendingConsent, clearPendingConsent } from "@/lib/onboarding";

export default function SignupPage() {
  const router = useRouter();
  const [step, setStep] = useState<"email" | "otp">("email");
  const [email, setEmail] = useState("");
  const [otp, setOtp] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSendOtp() {
    setError(null);
    setLoading(true);
    const { error } = await authClient.emailOtp.sendVerificationOtp({
      email,
      type: "sign-in",
    });
    setLoading(false);
    if (error) {
      setError(error.message ?? "인증코드 발송에 실패했습니다");
      return;
    }
    setStep("otp");
  }

  async function handleVerifyOtp() {
    setError(null);
    setLoading(true);
    const { error } = await authClient.signIn.emailOtp({ email, otp });
    if (error) {
      setLoading(false);
      setError(error.message ?? "인증에 실패했습니다");
      return;
    }

    const pending = getPendingConsent();
    if (pending) {
      await fetch("/api/v1/users/me/consents", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(pending),
      });
      clearPendingConsent();
    }

    setLoading(false);
    router.push("/home");
  }

  return (
    <div className="flex min-h-screen flex-col bg-bg md:items-center md:justify-center">
      <div className="flex items-center px-4 pt-4.5 pb-1.5 md:hidden">
        <Link
          href="/onboarding/permissions"
          aria-label="이전으로"
          className="flex h-9 w-9 items-center justify-center text-ink"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M15 5l-7 7 7 7" />
          </svg>
        </Link>
      </div>

      <div className="flex w-full flex-grow flex-col justify-center px-6 md:w-[420px] md:flex-grow-0 md:px-0">
        <BrandHeader className="hidden pb-9 md:flex" />

        {step === "email" && (
          <Form onFormSubmit={handleSendOtp} className="flex flex-col gap-5.5">
            <div className="flex flex-col gap-2 md:items-center">
              <span className="text-[22px] font-extrabold tracking-tight text-ink md:text-2xl">
                이메일로 시작하기
              </span>
              <span className="text-[13.5px] font-medium leading-relaxed text-ink-2 md:text-center md:text-sm">
                이메일 주소를 입력하면 인증코드를 보내드려요
              </span>
            </div>
            <TextField
              label="이메일"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <Button type="submit" disabled={!email || loading}>
              {loading ? "발송 중..." : "인증코드 받기"}
            </Button>
            <span className="text-center text-[11.5px] leading-relaxed text-ink-4">
              가입 시 이용약관 및 개인정보처리방침에 동의하게 돼요
            </span>
          </Form>
        )}

        {step === "otp" && (
          <Form onFormSubmit={handleVerifyOtp} className="flex flex-col gap-5.5">
            <div className="flex flex-col gap-2 md:items-center">
              <span className="text-[22px] font-extrabold tracking-tight text-ink md:text-2xl">
                인증코드를 입력하세요
              </span>
              <span className="text-[13.5px] font-medium leading-relaxed text-ink-2 md:text-center md:text-sm">
                {email || "you@example.com"}로 6자리 코드를 보냈어요
              </span>
            </div>
            <OtpInput label="인증코드" value={otp} onValueChange={setOtp} />
            <Button type="submit" disabled={otp.length < 6 || loading}>
              {loading ? "확인 중..." : "인증하고 시작하기"}
            </Button>
            <Button variant="text" onClick={() => setStep("email")}>
              이메일 다시 입력하기
            </Button>
          </Form>
        )}

        {error && <p className="pt-4 text-center text-sm text-danger">{error}</p>}
      </div>
    </div>
  );
}
