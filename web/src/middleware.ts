import { NextResponse, type NextRequest } from "next/server";

const ACCESS = "cs_access";
const REFRESH = "cs_refresh";
// access 토큰이 이 시간 안에 만료되면 미리 갱신한다
const REFRESH_SKEW_MS = 60_000;

const serverUrl = () => process.env.SERVER_URL ?? "http://localhost:4000";

/** JWT 본문의 exp(초)를 읽는다. 서명은 검증하지 않는다(만료 시각만 보고 갱신 여부를 정한다). */
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

/** 요청 쿠키 헤더에서 이름이 같은 쿠키를 바꾸거나 지운다(뒤따르는 서버 컴포넌트가 새 값을 보게 한다). */
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
 * 화면을 그리기 전에 로그인 토큰을 갱신한다.
 * 서버 컴포넌트는 쿠키를 설정할 수 없어서, access 토큰(30분)이 만료된 뒤에는 여기서 새로 받아 두어야 한다.
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
    return NextResponse.next(); // 서버에 닿지 않으면 그대로 진행한다(화면이 오류를 보여 준다)
  }

  const secure = process.env.COOKIE_SECURE === "true";
  const options = { httpOnly: true, secure, sameSite: "lax" as const, path: "/" };

  if (res.status === 401) {
    // 세션이 만료·폐기됐다: 쿠키를 지워 로그아웃 상태로 만든다
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

// 화면 경로에서만 돈다. API 요청은 브라우저가 apiFetch로 직접 갱신하고, 정적 파일은 건드리지 않는다.
export const config = {
  matcher: ["/((?!api/|_next/|favicon.ico|.*\\.[a-zA-Z0-9]+$).*)"],
};
