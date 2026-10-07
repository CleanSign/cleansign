"""조항 분류 프롬프트 실험 (Ollama). 사용: python3 classify_probe.py [모델명] [프롬프트버전]
라벨 데이터의 조항 + 임시 정상 조항(synthetic, 데이터셋 아님)으로 유형 분류를 시험한다."""
import csv, json, sys, time, urllib.request, pathlib

MODEL = sys.argv[1] if len(sys.argv) > 1 else "gemma4:12b"
VERSION = sys.argv[2] if len(sys.argv) > 2 else "v1"
ROOT = pathlib.Path(__file__).resolve().parents[2]

TYPES = ["선순위근저당", "보증금반환제한", "원상복구과도부담", "묵시적갱신배제", "기타불이익", "정상"]

SCHEMA = {
    "type": "object",
    "properties": {
        "clause_type": {"type": "string", "enum": TYPES},
        "is_risky": {"type": "boolean"},
        "confidence": {"type": "number"},
        "reason": {"type": "string"},
    },
    "required": ["clause_type", "is_risky", "confidence", "reason"],
}
# 속도 실험: FAST=1이면 reason 없이 분류만 받는다 (분류 단계와 설명 단계를 분리하는 구조 검증)
import os
if os.environ.get("FAST"):
    SCHEMA = {"type": "object",
              "properties": {"clause_type": SCHEMA["properties"]["clause_type"], "confidence": {"type": "number"}},
              "required": ["clause_type", "confidence"]}

SYSTEM_V1 = """당신은 한국 주택임대차 계약서의 특약 조항을 검토하는 도우미입니다.
조항 하나를 읽고 아래 유형 중 하나로 분류하세요.
- 선순위근저당: 선순위 담보·근저당 설정 관련, 임차인 보증금 회수 순위를 해치는 조항
- 보증금반환제한: 조건을 붙여 보증금 반환을 미루거나 거절·공제하는 조항
- 원상복구과도부담: 통상 손모·노후까지 포함해 과도한 수리·복구 비용을 임차인에게 지우는 조항
- 묵시적갱신배제: 임차인의 계약갱신요구권·묵시적 갱신 권리를 제한하는 조항
- 기타불이익: 위약금 과다, 비용 부당 전가 등 위 유형이 아닌 임차인 불이익 조항
- 정상: 임차인에게 불리하지 않은 통상 조항
주택임대차보호법상 임차인에게 불리한 특약은 효력이 없을 수 있지만, 여기서는 '임차인에게 불리한 문구인지'를 기준으로 분류합니다.
확실하지 않으면 confidence를 낮게 쓰세요. reason은 한 문장으로 쓰세요."""

SYSTEM_V2 = SYSTEM_V1 + """
판단 순서: (1) 이 조항이 임차인의 권리나 부담을 법정 기준보다 불리하게 바꾸는가? (2) 어떤 유형에 해당하는가?
조항에 구체 금액이나 조건이 있으면 reason에 그 부분을 인용하세요."""

SYNTHETIC_NORMAL = [
    "임대인은 임대차 기간 중 목적물의 주요 설비 노후로 인한 수선을 부담한다.",
    "임대차 기간은 2026년 3월 1일부터 2028년 2월 29일까지 2년으로 한다.",
    "임차인은 목적물을 선량한 관리자의 주의로 사용한다.",
    "차임은 매월 말일에 임대인이 지정한 계좌로 지급한다.",
]

def load():
    rows = []
    for x in csv.DictReader(open(ROOT / "data/labeling/dataset_template.csv", encoding="utf-8")):
        t = x["clause_text_verbatim"].strip()
        if t and x["clause_type"] in TYPES:
            rows.append((x["case_id"], t, x["clause_type"]))
    rows += [(f"SYN-NORMAL-{i+1}", t, "정상") for i, t in enumerate(SYNTHETIC_NORMAL)]
    return rows

def ask(text):
    body = {
        "model": MODEL, "stream": False, "think": False, "format": SCHEMA,
        "options": {"temperature": 0, "num_ctx": 4096},
        "messages": [
            {"role": "system", "content": SYSTEM_V2 if VERSION == "v2" else SYSTEM_V1},
            {"role": "user", "content": f"조항: {text}"},
        ],
    }
    req = urllib.request.Request("http://localhost:11434/api/chat", json.dumps(body).encode(), {"Content-Type": "application/json"})
    t0 = time.time()
    res = json.load(urllib.request.urlopen(req, timeout=300))
    return json.loads(res["message"]["content"]), time.time() - t0

if __name__ == "__main__":
    ok = 0; lat = []
    rows = load()
    for cid, text, gold in rows:
        try:
            out, s = ask(text)
        except Exception as e:
            print(cid, "ERROR", e); continue
        lat.append(s); hit = out["clause_type"] == gold; ok += hit
        print(f"{'O' if hit else 'X'} {cid:28} gold={gold:8} pred={out['clause_type']:8} conf={out['confidence']:.2f} {s:4.1f}s | {out.get('reason','')[:70]}")
    print(f"\n{MODEL} {VERSION}: {ok}/{len(rows)} 일치, 평균 {sum(lat)/max(len(lat),1):.1f}s, 최대 {max(lat, default=0):.1f}s")
