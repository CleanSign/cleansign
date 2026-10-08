import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Link from "next/link";
import { auth } from "@/lib/auth";
import { LogoutButton } from "@/components/logout-button";
import { ListRow } from "@/components/ui/list-row";
import { Avatar } from "@/components/ui/avatar";
import { Tag } from "@/components/ui/tag";

function MenuRow({
  href,
  icon,
  iconActive,
  label,
  sub,
}: {
  href: string;
  icon: React.ReactNode;
  iconActive?: boolean;
  label: string;
  sub?: string;
}) {
  return <ListRow href={href} icon={icon} tone={iconActive ? "accent" : "neutral"} title={label} sub={sub} />;
}

export default async function ProfilePage() {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session) {
    redirect("/");
  }

  return (
    <div className="flex min-h-screen flex-col items-center bg-bg">
      <div className="flex w-full flex-col px-5 md:w-130 md:px-0">
        <div className="flex items-center gap-2 pb-1.5 pt-4.5 md:pt-10">
          <Link href="/home" aria-label="홈으로" className="flex h-9 w-9 items-center justify-center text-ink">
            <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M15 5l-7 7 7 7" />
            </svg>
          </Link>
          <span className="text-[14.5px] font-bold text-ink">프로필</span>
        </div>

        <div className="pt-4">
          <div className="flex items-center gap-3.5 rounded-[18px] bg-surface px-4.5 py-4.5">
            <Avatar size={48} tone="accent">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="8" r="3.5" />
                <path d="M4.5 20c1.5-4 5-5.5 7.5-5.5s6 1.5 7.5 5.5" />
              </svg>
            </Avatar>
            <div className="flex grow flex-col gap-1">
              <span className="text-[14.5px] font-bold text-ink">{session.user.email}</span>
              <Tag>무료 플랜</Tag>
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-2.5 pt-5.5">
          <MenuRow
            href="#"
            icon={
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <rect x="3" y="6" width="18" height="13" rx="2.5" />
                <path d="M3 10h18" />
              </svg>
            }
            iconActive
            label="이용권 관리"
            sub="이번 달 1/3건 사용"
          />
          <MenuRow
            href="#"
            icon={
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <path d="M6 10a6 6 0 0112 0c0 4 1.5 5.5 1.5 5.5H4.5S6 14 6 10z" />
                <path d="M9.5 18a2.5 2.5 0 005 0" />
              </svg>
            }
            label="알림 설정"
          />
          <MenuRow
            href="/privacy"
            icon={
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <rect x="5" y="11" width="14" height="9" rx="2" />
                <path d="M8 11V7.5a4 4 0 018 0V11" />
              </svg>
            }
            label="개인정보처리방침"
          />
          <MenuRow
            href="#"
            icon={
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="9" />
                <path d="M9.5 9a2.5 2.5 0 015 0c0 1.8-2.2 2-2.5 3.5" />
                <circle cx="12" cy="16.3" r="0.5" fill="currentColor" stroke="none" />
              </svg>
            }
            label="고객센터"
          />
        </div>

        <div className="pt-5">
          <LogoutButton className="flex w-full items-center gap-3 rounded-2xl bg-danger/10 px-4 py-3.5 text-left">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-danger/14 text-danger">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <path d="M9 4H6a2 2 0 00-2 2v12a2 2 0 002 2h3" />
                <path d="M15 8l4 4-4 4" />
                <path d="M19 12H9" />
              </svg>
            </div>
            <span className="text-[13.5px] font-bold text-danger">로그아웃</span>
          </LogoutButton>
        </div>

        <div className="pb-8 pt-5 md:pb-11">
          <p className="text-center text-[11.5px] text-ink-4">Clean Sign v0.1</p>
        </div>
      </div>
    </div>
  );
}
