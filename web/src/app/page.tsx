import { redirect } from "next/navigation";
import Link from "next/link";
import { getSession } from "@/lib/api/server";

export default async function SplashPage() {
  const user = await getSession();
  if (user) {
    redirect("/home");
  }

  return (
    <Link
      href="/onboarding/terms"
      aria-label="시작하기"
      className="flex min-h-screen flex-col bg-bg"
    >
      <div className="flex grow flex-col items-center justify-center gap-5 px-8">
        <div className="flex h-20 w-20 items-center justify-center rounded-3xl bg-accent/14 text-accent-2 md:h-24 md:w-24">
          <svg
            width="40"
            height="40"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="md:h-11.5 md:w-11.5"
          >
            <path d="M12 3l7 3v5c0 4.5-3 8-7 9-4-1-7-4.5-7-9V6l7-3z" />
            <path d="M9 12l2 2 4-4" />
          </svg>
        </div>
        <div className="flex flex-col items-center gap-2.5">
          <span className="text-2xl font-extrabold tracking-tight text-ink">
            Clean Sign
          </span>
          <span className="text-center text-sm font-medium leading-relaxed text-ink-2">
            한번에 확인하는 임대차 리스크
          </span>
        </div>
      </div>

      <div className="flex items-center justify-center gap-1.5 pb-12">
        <span className="h-1.5 w-1.5 rounded-full bg-accent" />
        <span className="h-1.5 w-1.5 rounded-full bg-white/16" />
        <span className="h-1.5 w-1.5 rounded-full bg-white/16" />
      </div>
    </Link>
  );
}
