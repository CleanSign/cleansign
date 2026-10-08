import { headers } from "next/headers";
import { auth } from "@/lib/auth";

export async function GET() {
  const session = await auth.api.getSession({ headers: await headers() });

  if (!session) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }

  const { id, name, email, emailVerified } = session.user;
  return Response.json({ id, name, email, emailVerified });
}
