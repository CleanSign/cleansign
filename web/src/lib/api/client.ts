/**
 * Calls the server API from the browser. If the access token has expired and a 401 comes back, refresh it and retry once.
 * (The browser receives and stores the Set-Cookie from the refresh response directly.)
 * A stream body cannot be resent, so use this only with FormData or string bodies.
 */
export async function apiFetch(input: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(input, init);
  if (res.status !== 401) return res;

  const refreshed = await fetch("/api/v1/auth/refresh", { method: "POST" });
  if (!refreshed.ok) return res;
  return fetch(input, init);
}
