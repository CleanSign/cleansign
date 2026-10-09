import "server-only";
import { cache } from "react";
import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import type { ClauseDetail, Contract, ContractPage, Me } from "./types";

const serverUrl = () => process.env.SERVER_URL ?? "http://localhost:4000";

/** 서버 컴포넌트에서 Go 서버를 부른다. 요청의 쿠키(로그인 토큰)를 그대로 전달한다. */
export async function serverFetch(path: string, init?: RequestInit): Promise<Response> {
  const cookie = (await cookies()).toString();
  return fetch(`${serverUrl()}${path}`, {
    ...init,
    headers: { ...(cookie ? { cookie } : {}), ...init?.headers },
    cache: "no-store",
  });
}

/** 로그인한 사용자. 로그인하지 않았으면 null. 같은 요청 안에서는 한 번만 부른다. */
export const getSession = cache(async (): Promise<Me | null> => {
  const res = await serverFetch("/api/v1/users/me");
  if (res.status === 401) return null;
  if (!res.ok) throw new Error(`users/me ${res.status}`);
  return res.json();
});

/** 로그인이 필요한 화면에서 쓴다. 로그인하지 않았으면 `/`로 보낸다. */
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

/** 계약서 분석 상태와 결과. 남의 계약서나 없는 계약서는 404 화면이 된다. */
export async function getContract(id: string): Promise<Contract> {
  return json(await serverFetch(`/api/v1/contracts/${encodeURIComponent(id)}`));
}

export async function getClause(id: string, clauseId: string): Promise<ClauseDetail> {
  return json(await serverFetch(`/api/v1/contracts/${encodeURIComponent(id)}/clauses/${encodeURIComponent(clauseId)}`));
}

/** 히스토리 한 페이지(최신순, 실패한 분석 포함). */
export async function getContractPage(page: number): Promise<ContractPage> {
  return json(await serverFetch(`/api/v1/contracts?page=${page}`));
}

/** 홈 화면의 최근 분석: 완료·진행 중만(실패 제외). */
export async function getRecentContracts(limit: number): Promise<ContractPage["items"]> {
  const page = await json<ContractPage>(await serverFetch(`/api/v1/contracts?status=active&limit=${limit}`));
  return page.items;
}
