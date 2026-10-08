import { headers } from "next/headers";
import { auth } from "@/lib/auth";
import { db } from "@/db/db";
import { userConsent } from "@/db/schema";

export async function POST(req: Request) {
  const session = await auth.api.getSession({ headers: await headers() });

  if (!session) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }

  const body = await req.json().catch(() => null);
  const terms = body?.terms;
  const privacy = body?.privacy;
  const marketing = body?.marketing ?? false;

  if (typeof terms !== "boolean" || typeof privacy !== "boolean") {
    return Response.json(
      { error: "terms, privacy는 boolean이어야 합니다" },
      { status: 400 },
    );
  }

  if (!terms || !privacy) {
    return Response.json(
      { error: "이용약관과 개인정보처리방침 동의는 필수입니다" },
      { status: 400 },
    );
  }

  const [consent] = await db
    .insert(userConsent)
    .values({
      userId: session.user.id,
      terms,
      privacy,
      marketing: typeof marketing === "boolean" ? marketing : false,
    })
    .returning();

  return Response.json(consent, { status: 201 });
}
