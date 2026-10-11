"""'임대차 계약 독소조항 100개 조사.pdf'의 표(번호/조항/효력/근거)를 CSV로 옮긴다.
실행: cd ai-service && uv run python ../scripts/data/parse_toxic100.py
출력: data/labeling/toxic_clauses_100.csv (유형 매핑은 규칙 기반 초안이며 검수 전)"""
import csv, re, pathlib
from pypdf import PdfReader

ROOT = pathlib.Path(__file__).resolve().parents[2]
SRC = ROOT / "data/labeling/source_docs/임대차 계약 독소조항 100개 조사.pdf"
OUT = ROOT / "data/labeling/toxic_clauses_100.csv"

EFFECTS = ["일부 무효", "무효 가능성 높음", "무효 소지", "조건부 유효", "유효 가능", "무효", "불공정", "부당", "주의", "유효", "위법"]
SECTION_BY_NO = [(1, "금전·보증금"), (21, "유지보수·원상복구"), (41, "계약해지·갱신"), (61, "사생활·생활간섭"), (81, "기타 절차·권리제한")]


def reflow(pdf: PdfReader) -> str:
    # 이 PDF는 토큰마다 줄바꿈이 있어 한 줄로 이어 붙이고 문장부호 앞뒤 공백을 정리한다
    parts = []
    for p in pdf.pages:
        s = " ".join(l.strip() for l in (p.extract_text() or "").split("\n") if l.strip())
        s = re.sub(r"\s+([.,:;)\]·])", r"\1", s)
        s = re.sub(r"([(\[·])\s+", r"\1", s)
        parts.append(s)
    return re.sub(r"\s+", " ", " ".join(parts))


def split_rows(text: str) -> dict[int, str]:
    """1번부터 100번까지, 앞 행이 끝난 위치 뒤에서 '<번호> "' 를 순서대로 찾는다."""
    start = text.index("번호 독소조항 내용")
    pos, marks = start, {}
    for n in range(1, 101):
        m = re.compile(rf'(?<![\d])\b{n} "').search(text, pos)
        if not m:
            raise SystemExit(f"{n}번 행을 찾지 못했습니다")
        marks[n] = m
        pos = m.end()
    rows = {}
    for n in range(1, 101):
        end = marks[n + 1].start() if n < 100 else len(text)
        rows[n] = text[marks[n].end():end].strip()
    return rows


def parse_row(raw: str):
    """'조항 원문" [(보충)] 효력 근거...' 로 나눈다. 효력 키워드를 못 찾으면 effect를 비운다."""
    best = None
    for eff in EFFECTS:
        for m in re.finditer(rf'" ?(?:\([^)]{{0,40}}\) )?{re.escape(eff)}(?= )', raw):
            if best is None or m.start() < best[0].start():
                best = (m, eff)
    if not best:
        return raw.strip(' "'), "", "", raw
    m, eff = best
    clause = raw[:m.start()].strip(' "')
    note = re.search(r'\(([^)]{0,40})\)', raw[m.start():m.end()])
    basis = raw[m.end():].strip()
    return clause, eff, (note.group(1) if note else ""), basis


# PDF 표에서 칸이 줄바꿈으로 뒤섞여 자동 분리가 안 되는 행: 원문을 보고 사람이 복원한 값.
# (조항 원문, 효력, 근거) — 단어는 PDF 그대로이고 순서만 바로잡았다.
MANUAL_FIX = {
    9: ("계약 기간 중 임차인이 퇴거할 경우, 남은 기간의 월세를 위약금으로 일시불 지급한다.", "무효",
        "잔여 기간 월세 전액을 위약금으로 하는 것은 임차인에게 부당하게 과중한 손해배상 의무를 부과하는 것으로 무효 소지 큼 (약관법 제 8 조)."),
    26: ("입주 시 설치된 시설물 일체를 현 상태로 반환해야 한다. (이전 임차인 시설 포함)", "무효",
         "별도의 약정이 없는 한 현 임차인은 자신이 설치한 시설에 대해서만 원상회복 의무가 있음. 이전 임차인의 시설까지 철거할 의무 없음 (대법원 90 다카 12035)."),
    28: ("형광등, 수전 등 소모품 교체도 임대인이 한다. (임차인 주장)", "기각",
         "반대로, 통상적인 소규모 소모품 (전구, 배터리 등) 교체는 민법상 임차인의 보존 행위 (비용 부담) 로 보는 것이 판례의 태도."),
    45: ("임차인이 연락 두절되거나 연체 시 짐을 임의로 처분해도 민형사상 책임을 묻지 않는다.", "무효",
         "자력구제 금지 원칙 위반. 명도소송 및 강제집행 절차 없이 임의 처분 시 주거침입, 절도, 손괴죄 성립."),
    61: ("임대인은 마스터키를 보관하며 비상시 출입할 수 있다.", "위험",
         "' 비상시 ' 의 범위가 모호하여 악용 소지 큼. 임차인 부재 시 무단출입은 주거침입죄 성립. 명확한 동의 절차 필수."),
    66: ("밤 10 시 이후 세탁기 사용 및 샤워 금지.", "권고적",
         "공동주택 생활 수칙으로 권고할 수 있으나, 이를 위반했다고 계약 해지나 위약금을 물리는 것은 과도함."),
    77: ("임대차 기간 중 임대인이 거주하는 층의 화장실을 함께 사용한다.", "부당",
         "독립된 주거 공간 임대차의 본질에 반함. (하숙 등 특수한 경우 제외)"),
    83: ("소송 발생 시 관할 법원은 임대인 주소지로 한다.", "부당",
         "민사소송법상 관할 (피고 주소지, 의무이행지 등) 이익을 임차인에게 불리하게 변경하는 합의관할 조항은 약관일 경우 무효 가능."),
    86: ("임차인은 계약금 영수증만으로 확정일자를 받는다. (계약서 미교부 시도)", "위험",
         "계약서 원본이 있어야 정확한 권리 관계 증빙 가능. 계약서 미작성 / 미교부는 중개사법 위반."),
    89: ("임차인은 배상책임보험을 의무적으로 가입하고 수익자를 임대인으로 한다.", "부당",
         "임차인 부담으로 임대인을 수익자로 하는 보험 강요는 부당."),
    95: ("근저당권 설정을 위해 임차인은 잠시 전출 신고를 했다가 재전입한다.", "위험 / 무효",
         "임차인의 대항력 순위를 뒤로 미루는 작업으로, 사고 발생 시 보증금 전액 손실 위험. 이에 응할 의무 없음."),
    98: ("임차인이 외국인인 경우 보증금을 더 높게 책정한다.", "차별", "합리적 이유 없는 국적 차별."),
}

# 우리 6유형(data/labeling/README.md)으로의 규칙 기반 초안. 검수 전이라 clause_type_auto로만 둔다.
RULES = [
    ("묵시적갱신배제", ["갱신", "묵시", "재건축", "리모델링", "계약기간", "해지"]),
    ("원상복구과도부담", ["원상복구", "수리", "수선", "보수", "교체", "도배", "장판", "청소", "시설물", "파손", "훼손"]),
    ("보증금반환제한", ["보증금"]),
]


def map_type(section_no: int, clause: str) -> str:
    if section_no >= 61:
        return "기타불이익"
    for t, kws in RULES:
        if any(k in clause for k in kws):
            return t
    return "기타불이익"


def main():
    text = reflow(PdfReader(str(SRC)))
    rows = split_rows(text)
    out = []
    for n, raw in rows.items():
        clause, eff, note, basis = parse_row(raw)
        parse_note = ""
        if n in MANUAL_FIX:
            clause, eff, basis = MANUAL_FIX[n]
            note, parse_note = "", "수동 복원(PDF 표 칸 뒤섞임)"
        # 근거 끝의 숫자는 보고서 참고문헌 번호다
        ref = re.search(r"\s(\d{1,3})\.?$", basis)
        ref_no = ref.group(1) if ref else ""
        if ref:
            basis = basis[:ref.start()].strip()
        section = [s for a, s in SECTION_BY_NO if n >= a][-1]
        out.append({
            "no": n, "report_section": section, "clause_text_verbatim": clause,
            "legal_effect_report": eff, "legal_effect_note": note, "basis_report": basis, "report_ref_no": ref_no,
            "clause_type_auto": map_type(n, clause), "label_status": "unreviewed", "parse_note": parse_note,
            "source_file": "source_docs/임대차 계약 독소조항 100개 조사.pdf",
        })
    with open(OUT, "w", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(f, fieldnames=list(out[0]))
        w.writeheader(); w.writerows(out)
    empty = [r["no"] for r in out if not r["legal_effect_report"]]
    print(f"{len(out)}행 저장 -> {OUT.relative_to(ROOT)}; 효력 미추출: {empty}")


if __name__ == "__main__":
    main()
