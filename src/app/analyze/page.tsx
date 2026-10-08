import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Link from "next/link";
import { auth } from "@/lib/auth";
import { UploadForm } from "./upload-form";

export default async function AnalyzePage() {
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
          <span className="text-[14.5px] font-bold text-ink">계약서 분석</span>
        </div>
        <UploadForm />
      </div>
    </div>
  );
}
