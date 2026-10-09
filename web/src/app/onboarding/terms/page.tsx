"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { CheckboxGroup } from "@base-ui/react/checkbox-group";
import { Separator } from "@base-ui/react/separator";
import { savePendingConsent } from "@/lib/onboarding";
import { BrandHeader } from "@/components/ui/brand";
import { Button } from "@/components/ui/button";
import { CheckboxRow } from "@/components/ui/checkbox-row";

function Row({
  value,
  label,
  tag,
  tagColor,
}: {
  value: string;
  label: string;
  tag: string;
  tagColor: string;
}) {
  return (
    <CheckboxRow value={value}>
      <span className="text-[13.5px] text-ink-2/90">
        <span className={`font-bold ${tagColor}`}>[{tag}] </span>
        {label}
      </span>
    </CheckboxRow>
  );
}

const ALL_VALUES = ["terms", "privacy", "marketing"];

export default function TermsPage() {
  const router = useRouter();
  const [values, setValues] = useState<string[]>([]);

  const terms = values.includes("terms");
  const privacy = values.includes("privacy");
  const marketing = values.includes("marketing");
  const canProceed = terms && privacy;

  function handleNext() {
    savePendingConsent({ terms, privacy, marketing });
    router.push("/onboarding/permissions");
  }

  return (
    <div className="flex min-h-screen flex-col bg-bg md:items-center md:justify-center">
      <div className="flex w-full grow flex-col px-5 pt-6 md:w-110 md:grow-0 md:px-0 md:pt-0">
        <BrandHeader className="pb-9" />

        <div className="pb-7 text-center text-[21px] font-extrabold leading-snug tracking-tight text-ink md:text-2xl">
          서비스 이용을 위해
          <br className="md:hidden" /> 약관에 동의해주세요
        </div>

        <CheckboxGroup
          value={values}
          onValueChange={setValues}
          allValues={ALL_VALUES}
          className="flex flex-col gap-4 rounded-2xl bg-surface p-4.5"
        >
          <CheckboxRow parent>
            <span className="text-[15px] font-extrabold text-ink">약관 전체 동의</span>
          </CheckboxRow>
          <Separator className="h-px bg-white/8" />
          <Row value="terms" label="이용약관 동의" tag="필수" tagColor="text-danger" />
          <Row value="privacy" label="개인정보 처리방침 동의" tag="필수" tagColor="text-danger" />
          <Row value="marketing" label="마케팅 정보 수신 동의" tag="선택" tagColor="text-ink-3" />
        </CheckboxGroup>

        <div className="mt-auto pt-6 pb-10 md:mt-7 md:pb-0">
          <Button disabled={!canProceed} onClick={handleNext}>
            동의하고 계속하기
          </Button>
        </div>
      </div>
    </div>
  );
}
