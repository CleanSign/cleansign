import Link from "next/link";

const SECTIONS = [
  {
    title: "1. 수집하는 개인정보 항목",
    body: "이메일 주소, 업로드한 계약서 이미지 및 텍스트, 서비스 이용 기록을 수집해요.",
  },
  {
    title: "2. 개인정보의 수집 및 이용 목적",
    body: "계약서 분석 서비스 제공, 부정 이용 방지, 서비스 개선을 위한 통계 분석에 이용해요.",
  },
  {
    title: "3. 보유 및 이용 기간",
    body: "회원 탈퇴 시 즉시 파기하며, 관계 법령에 따라 보관이 필요한 정보는 해당 기간 동안 보관해요.",
  },
  {
    title: "4. 개인정보의 제3자 제공",
    body: "전문가 상담을 신청한 경우, 예약 정보(이름·연락처·상담 내용)가 담당 전문가에게 제공돼요. 그 외에는 이용자의 동의 없이 제3자에게 제공하지 않아요.",
  },
  {
    title: "5. 이용자의 권리",
    body: "언제든지 본인의 개인정보 열람, 정정, 삭제를 요청할 수 있어요. 요청은 프로필 > 고객센터를 통해 접수할 수 있어요.",
  },
  {
    title: "6. 개인정보 보호책임자 및 문의처",
    body: "개인정보 관련 문의는 privacy@cleansign.kr로 연락해 주세요.",
  },
];

export default function PrivacyPage() {
  return (
    <div className="flex min-h-screen flex-col items-center bg-bg">
      <div className="flex w-full flex-col px-5 md:w-[620px] md:px-0">
        <div className="flex items-center gap-2 pb-1.5 pt-4.5 md:pt-10">
          <Link href="/profile" aria-label="뒤로가기" className="flex h-9 w-9 items-center justify-center text-ink">
            <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M15 5l-7 7 7 7" />
            </svg>
          </Link>
          <span className="text-[14.5px] font-bold text-ink">개인정보처리방침</span>
        </div>

        <div className="pt-4">
          <span className="text-xs text-ink-3">최종 수정일 2026.09.01</span>
        </div>

        <div className="flex flex-col gap-5 pt-6 md:gap-6">
          {SECTIONS.map((s) => (
            <div key={s.title} className="flex flex-col gap-2">
              <span className="text-[14.5px] font-bold text-ink md:text-[15.5px]">{s.title}</span>
              <span className="text-[13.5px] leading-[1.75] text-ink-2 md:text-sm md:leading-[1.8]">{s.body}</span>
            </div>
          ))}
        </div>

        <div className="pb-9 pt-6 md:pb-11 md:pt-7">
          <p className="text-[11px] leading-relaxed text-ink-4 md:text-xs">
            본 방침은 예시 콘텐츠이며 실제 법률 검토가 필요해요.
          </p>
        </div>
      </div>
    </div>
  );
}
