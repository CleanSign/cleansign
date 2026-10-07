import "server-only";

export type AiClause = {
  id: string;
  original: string;
  clauseType: string;
  confidence: number;
  severity: "high" | "medium" | "low";
  lMax: number;
  lMaxFormula: string;
  expectedLoss: number;
  reason: string;
  lawReference: string[];
  precedents: string[];
  safeText: string;
};

export type AiAnalysis = {
  riskScore: number;
  grade: "high" | "medium" | "low";
  lossAmount: number;
  lossRange: [number, number];
  facts: { deposit: number | null; monthlyRent: number | null; leaseType: string };
  clauses: AiClause[];
  flow: unknown | null;
  warnings: string[];
  modelVersion: string;
  policyVersion: string;
  stats: Record<string, number>;
};

export type AiProgress = { stage: string; done: number; total: number };

/** 사용자에게 보여 줄 수 있는 실패. status는 우리 API 기준의 분류다. */
export class AiServiceError extends Error {
  constructor(
    message: string,
    public status: 422 | 502 | 504,
  ) {
    super(message);
  }
}

const baseUrl = () => process.env.AI_SERVICE_URL ?? "http://localhost:8000";
// 분석은 백그라운드로 돌기 때문에 사용자 요청 시간 제한과 무관하다. 로컬 모델은 길게 걸릴 수 있다.
const timeoutMs = () => Number(process.env.AI_SERVICE_TIMEOUT_MS ?? 600_000);

/** AI 서버의 NDJSON 스트림을 읽으며 진행 상황을 알리고, 마지막에 결과를 돌려준다. */
export async function analyzeContract(
  file: File,
  onProgress: (p: AiProgress) => void | Promise<void>,
): Promise<AiAnalysis> {
  const form = new FormData();
  form.append("file", file, file.name);

  const headers: Record<string, string> = {};
  if (process.env.AI_SERVICE_TOKEN) headers["X-Internal-Token"] = process.env.AI_SERVICE_TOKEN;

  let res: Response;
  try {
    res = await fetch(`${baseUrl()}/v1/analyze/stream`, {
      method: "POST",
      body: form,
      headers,
      signal: AbortSignal.timeout(timeoutMs()),
    });
  } catch (e) {
    throw toConnectionError(e);
  }

  if (res.status === 422) {
    // 계약서를 읽지 못한 경우(스캔본 등). 메시지는 사용자에게 그대로 보여 줘도 된다.
    const body = await res.json().catch(() => null);
    throw new AiServiceError(body?.detail ?? "계약서를 읽을 수 없어요", 422);
  }
  if (!res.ok || !res.body) {
    console.error("[ai-service]", res.status, await res.text().catch(() => ""));
    throw new AiServiceError("분석 중 문제가 생겼어요", 502);
  }

  try {
    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    for (;;) {
      const { value, done } = await reader.read();
      buffer += decoder.decode(value, { stream: !done });
      let nl: number;
      while ((nl = buffer.indexOf("\n")) >= 0) {
        const line = buffer.slice(0, nl).trim();
        buffer = buffer.slice(nl + 1);
        if (!line) continue;
        const ev = JSON.parse(line);
        if (ev.type === "progress") {
          await onProgress({ stage: ev.stage, done: ev.done, total: ev.total });
        } else if (ev.type === "result") {
          return ev.data as AiAnalysis;
        } else if (ev.type === "error") {
          if (ev.readable) throw new AiServiceError(ev.detail, 422); // 스캔본 OCR 실패 등: 메시지를 그대로 보여 준다
          console.error("[ai-service]", ev.detail);
          throw new AiServiceError("분석 중 문제가 생겼어요", 502);
        }
      }
      if (done) break;
    }
  } catch (e) {
    if (e instanceof AiServiceError) throw e;
    throw toConnectionError(e);
  }
  throw new AiServiceError("분석 결과를 받지 못했어요", 502);
}

function toConnectionError(e: unknown) {
  if (e instanceof Error && (e.name === "TimeoutError" || e.name === "AbortError")) {
    return new AiServiceError("분석이 너무 오래 걸려 중단했어요. 잠시 후 다시 시도해 주세요", 504);
  }
  return new AiServiceError("분석 서버에 연결하지 못했어요", 502);
}
