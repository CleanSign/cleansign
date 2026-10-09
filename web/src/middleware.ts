import { NextResponse, type NextRequest } from "next/server";

const ACCESS = "cs_access";
const REFRESH = "cs_refresh";
// Refresh the access token ahead of time if it expires within this window
const REFRESH_SKEW_MS = 60_000;

const serverUrl = () => process.env.SERVER_URL ?? "http://localhost:4000";

/** Reads exp (seconds) from the JWT payload. Does not verify the signature (only the expiry time decides whether to refresh). */
function expiresAtMs(token: string | undefined): number | null {
  if (!token) return null;
  try {
    const payload = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const exp = JSON.parse(atob(payload)).exp;
    return typeof exp === "number" ? exp * 1000 : null;
  } catch {
    return null;
  }
}

/** Replaces or removes the same-named cookie in the request cookie header (so the following server components see the new value). */
function withCookies(req: NextRequest, set: Record<string, string | null>): Headers {
  const jar = new Map<string, string>();
  for (const c of req.cookies.getAll()) jar.set(c.name, c.value);
  for (const [name, value] of Object.entries(set)) {
    if (value === null) jar.delete(name);
    else jar.set(name, value);
  }
  const headers = new Headers(req.headers);
  headers.set("cookie", [...jar].map(([k, v]) => `${k}=${v}`).join("; "));
  return headers;
}

/**
 * Refreshes the login token before rendering the page.
 * Server components cannot set cookies, so once the access token (30 minutes) expires, a new one must be obtained here.
 */
export async function middleware(req: NextRequest) {
  const refresh = req.cookies.get(REFRESH)?.value;
  if (!refresh) return NextResponse.next();

  const exp = expiresAtMs(req.cookies.get(ACCESS)?.value);
  if (exp !== null && exp - Date.now() > REFRESH_SKEW_MS) return NextResponse.next();

  let res: Response;
  try {
    res = await fetch(`${serverUrl()}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { cookie: `${REFRESH}=${refresh}` },
      cache: "no-store",
    });
  } catch {
    return NextResponse.next(); // If the server is unreachable, proceed as is (the page shows the error)
  }

  const secure = process.env.COOKIE_SECURE === "true";
  const options = { httpOnly: true, secure, sameSite: "lax" as const, path: "/" };

  if (res.status === 401) {
    // The session expired or was revoked: clear the cookies to log the user out
    const out = NextResponse.next({ request: { headers: withCookies(req, { [ACCESS]: null, [REFRESH]: null }) } });
    out.cookies.delete(ACCESS);
    out.cookies.delete(REFRESH);
    return out;
  }
  if (!res.ok) return NextResponse.next();

  const t = await res.json();
  const out = NextResponse.next({ request: { headers: withCookies(req, { [ACCESS]: t.accessToken, [REFRESH]: t.refreshToken }) } });
  out.cookies.set(ACCESS, t.accessToken, { ...options, expires: new Date(t.accessExpiresAt) });
  out.cookies.set(REFRESH, t.refreshToken, { ...options, expires: new Date(t.refreshExpiresAt) });
  return out;
}

// Runs on page routes only. API requests are refreshed directly by the browser via apiFetch, and static files are left alone.
export const config = {
  matcher: ["/((?!api/|_next/|favicon.ico|.*\\.[a-zA-Z0-9]+$).*)"],
};
