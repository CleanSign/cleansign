import "server-only";
import { cache } from "react";
import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import type { ClauseDetail, Contract, ContractPage, Me } from "./types";

const serverUrl = () => process.env.SERVER_URL ?? "http://localhost:4000";

/** Calls the Go server from a server component. Forwards the request's cookies (login tokens) as is. */
export async function serverFetch(path: string, init?: RequestInit): Promise<Response> {
  const cookie = (await cookies()).toString();
  return fetch(`${serverUrl()}${path}`, {
    ...init,
    headers: { ...(cookie ? { cookie } : {}), ...init?.headers },
    cache: "no-store",
  });
}

/** The logged-in user, or null if not logged in. Called only once per request. */
export const getSession = cache(async (): Promise<Me | null> => {
  const res = await serverFetch("/api/v1/users/me");
  if (res.status === 401) return null;
  if (!res.ok) throw new Error(`users/me ${res.status}`);
  return res.json();
});

/** Use on screens that require login. Redirects to `/` if not logged in. */
export async function requireUser(): Promise<Me> {
  const user = await getSession();
  if (!user) redirect("/");
  return user;
}

async function json<T>(res: Response): Promise<T> {
  if (res.status === 401) redirect("/");
  if (res.status === 404) notFound();
  if (!res.ok) throw new Error(`server ${res.status}`);
  return res.json();
}

/** Contract analysis status and result. Someone else's contract, or one that does not exist, results in a 404 screen. */
export async function getContract(id: string): Promise<Contract> {
  return json(await serverFetch(`/api/v1/contracts/${encodeURIComponent(id)}`));
}

export async function getClause(id: string, clauseId: string): Promise<ClauseDetail> {
  return json(await serverFetch(`/api/v1/contracts/${encodeURIComponent(id)}/clauses/${encodeURIComponent(clauseId)}`));
}

/** One page of history (newest first, including failed analyses). */
export async function getContractPage(page: number): Promise<ContractPage> {
  return json(await serverFetch(`/api/v1/contracts?page=${page}`));
}

/** Recent analyses on the home screen: completed or in progress only (failed excluded). */
export async function getRecentContracts(limit: number): Promise<ContractPage["items"]> {
  const page = await json<ContractPage>(await serverFetch(`/api/v1/contracts?status=active&limit=${limit}`));
  return page.items;
}
