/**
 * 브라우저에서 서버 API를 부른다. access 토큰이 만료돼 401을 받으면 refresh로 갱신하고 한 번 다시 시도한다.
 * (갱신 응답의 Set-Cookie는 브라우저가 직접 받아 저장한다.)
 * 본문이 스트림이면 다시 보낼 수 없으므로 FormData·문자열 본문에만 쓴다.
 */
export async function apiFetch(input: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(input, init);
  if (res.status !== 401) return res;

  const refreshed = await fetch("/api/v1/auth/refresh", { method: "POST" });
  if (!refreshed.ok) return res;
  return fetch(input, init);
}
