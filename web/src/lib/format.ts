/** 원 단위 금액을 읽기 쉽게: 3,612,000 → "약 361만 원", 150,000,000 → "약 1.5억 원" */
export function formatWon(n: number): string {
  if (n <= 0) return "0원";
  if (n >= 100_000_000) {
    const eok = Math.round(n / 10_000_000) / 10;
    return `약 ${eok}억 원`;
  }
  if (n >= 10_000) return `약 ${Math.round(n / 10_000).toLocaleString("ko-KR")}만 원`;
  return `약 ${n.toLocaleString("ko-KR")}원`;
}

export function formatDate(d: Date): string {
  const parts = new Intl.DateTimeFormat("ko-KR", {
    timeZone: "Asia/Seoul",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(d);
  const get = (t: string) => parts.find((p) => p.type === t)?.value ?? "";
  return `${get("year")}.${get("month")}.${get("day")}`;
}

export const stripPdf = (name: string) => name.replace(/\.pdf$/i, "");

export const CLAUSE_LABEL: Record<string, string> = {
  선순위근저당: "선순위 근저당",
  보증금반환제한: "보증금 반환 제한",
  원상복구과도부담: "원상복구 과도 부담",
  묵시적갱신배제: "갱신 권리 제한",
  기타불이익: "기타 불리한 조항",
};
export const clauseLabel = (t: string | null) => (t && CLAUSE_LABEL[t]) || "위험 조항";

/**
 * 조항 칩. severity는 '법적으로 얼마나 나쁜가'가 아니라 예상 손실이 보증금에 비해 얼마나 큰가(손실 규모)이다.
 * 그래서 "위험 높음/낮음"이 아니라 "손실 큼/보통/작음"으로 쓰고, 작은 쪽을 초록(안전)으로 칠하지 않는다.
 */
export const SEVERITY_CHIP: Record<string, { label: string; tone: "danger" | "warn" | "neutral" }> = {
  high: { label: "손실 큼", tone: "danger" },
  medium: { label: "손실 보통", tone: "warn" },
  low: { label: "손실 작음", tone: "neutral" },
};
