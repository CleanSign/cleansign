"""data/labeling/ 의 CSV를 스키마대로 검증하면서 읽는다. 표준 라이브러리만 쓴다.
검증: python3 scripts/data/labeling.py   (문제를 전부 출력하고 하나라도 있으면 종료 코드 1, CI에서도 실행)
사용: from labeling import load; rows = load("toxic_clauses_100.csv")
      빈 숫자·불리언 칸은 None, 텍스트 칸은 문자열 그대로 돌려준다. 문제가 있으면 LabelingError.
스키마 근거: data/labeling/README.md 2절. CSV의 열을 바꾸면 여기 SCHEMAS도 같이 고친다."""
import csv
import datetime
import io
import pathlib
import re
import sys
from collections.abc import Callable
from dataclasses import dataclass, field

ROOT = pathlib.Path(__file__).resolve().parents[2]
DATA = ROOT / "data/labeling"

# Same as ai-service/app/llm/base.py CLAUSE_TYPES
CLAUSE_TYPES = ("선순위근저당", "보증금반환제한", "원상복구과도부담", "묵시적갱신배제", "기타불이익", "정상")
RISK_TYPES = CLAUSE_TYPES[:-1]
L_MAX_FORMULAS = ("SENIOR_LIEN", "DEPOSIT_DELAY", "RESTORATION", "RENEWAL_BLOCK", "OTHER_CAP")
LABEL_STATUS = ("unreviewed", "needs_review", "reviewed")


class LabelingError(Exception):
    def __init__(self, problems: list[str]):
        self.problems = problems
        super().__init__("\n".join(problems))


# A parser turns the raw cell into a value, or raises ValueError with the reason.
Parser = Callable[[str], object]


def text(required: bool = True) -> Parser:
    def parse(s: str) -> str:
        if required and not s.strip():
            raise ValueError("비어 있으면 안 됨")
        return s
    return parse


def integer(required: bool = False, lo: int | None = None, hi: int | None = None) -> Parser:
    def parse(s: str) -> int | None:
        if not s.strip():
            if required:
                raise ValueError("비어 있으면 안 됨")
            return None
        if not re.fullmatch(r"-?\d+", s.strip()):
            raise ValueError("정수가 아님")
        v = int(s)
        if (lo is not None and v < lo) or (hi is not None and v > hi):
            raise ValueError(f"범위 밖 ({lo}~{hi})")
        return v
    return parse


def number(lo: float | None = None, hi: float | None = None) -> Parser:
    def parse(s: str) -> float | None:
        if not s.strip():
            return None
        try:
            v = float(s)
        except ValueError:
            raise ValueError("숫자가 아님") from None
        if (lo is not None and v < lo) or (hi is not None and v > hi):
            raise ValueError(f"범위 밖 ({lo}~{hi})")
        return v
    return parse


def boolean(s: str) -> bool | None:
    if not s.strip():
        return None
    if s not in ("true", "false"):
        raise ValueError("true 또는 false가 아님")
    return s == "true"


def choice(*values: str, required: bool = True, note_suffix: bool = False) -> Parser:
    """note_suffix: allows a value followed by a "(...)" note, e.g. 임차인일부승(사실상 거의 전부승)."""
    def parse(s: str) -> str:
        if not s:
            if required:
                raise ValueError("비어 있으면 안 됨")
            return s
        base = re.sub(r"\(.*\)$", "", s).strip() if note_suffix else s
        if base not in values:
            raise ValueError(f"허용값이 아님 ({' / '.join(values)})")
        return s
    return parse


def risk_type_or_na(s: str) -> str:
    if s not in RISK_TYPES and not s.startswith("N/A_"):
        raise ValueError(f"위험 유형이나 N/A_* 가 아님 ({' / '.join(RISK_TYPES)})")
    return s


def iso_date(s: str) -> datetime.date:
    try:
        return datetime.date.fromisoformat(s)
    except ValueError:
        raise ValueError("YYYY-MM-DD 날짜가 아님") from None


def int_list(s: str) -> list[int]:
    if not s.strip():
        return []
    if not re.fullmatch(r"\d+(;\d+)*", s):
        raise ValueError("세미콜론으로 구분한 정수 목록이 아님 (예: 21;22)")
    return [int(x) for x in s.split(";")]


def source_ref(s: str) -> str:
    """An http(s) URL, or a repo path optionally followed by a locator, e.g. 'data/labeling/x.txt (사례16)'."""
    if s.startswith(("http://", "https://")):
        return s
    if s and (ROOT / s.split(" ", 1)[0]).is_file():
        return s
    raise ValueError("http(s) URL도, 저장소에 있는 파일 경로도 아님")


def existing_file(s: str) -> str:
    if not (DATA / s).is_file():
        raise ValueError("data/labeling/ 아래에 없는 파일")
    return s


Row = dict[str, object]


def clause_id_matches_case(r: Row) -> str | None:
    if not str(r["clause_id"]).startswith(f"{r['case_id']}-"):
        return "clause_id가 '<case_id>-'로 시작하지 않음"
    return None


def ratio_matches_amounts(r: Row) -> str | None:
    lmax, loss, ratio = r["L_max_value"], r["actual_loss_amount"], r["realized_ratio_conditional"]
    if lmax and loss is not None and ratio is not None and abs(loss / lmax - ratio) > 0.01:
        return f"realized_ratio_conditional({ratio})가 actual_loss_amount/L_max_value({loss / lmax:.3f})와 다름"
    return None


@dataclass
class Schema:
    columns: dict[str, Parser]
    unique: tuple[str, ...] = ()
    checks: list[Callable[[Row], str | None]] = field(default_factory=list)


SEVERITY_COLUMNS: dict[str, Parser] = {
    "case_id": text(),
    "title": text(),
    "dispute_type": text(),
    "clause_type_mapping": risk_type_or_na,
    "has_verbatim_clause": choice("Y", "N"),
    "deposit_amount": integer(lo=0),
    "disputed_amount": integer(lo=0),
    "settlement_amount": integer(lo=0),
    "ratio_settlement_to_disputed": number(lo=0),
}

SCHEMAS: dict[str, Schema] = {
    "dataset_template.csv": Schema(
        {
            "case_id": text(),
            "clause_id": text(),
            "source_type": choice("조정성립", "조정불성립", "판결", "표준계약서", "공식해설"),
            "source_citation": text(),
            "clause_type": choice(*CLAUSE_TYPES),
            "clause_text_verbatim": text(required=False),
            "clause_text_normalized": text(required=False),
            "deposit_amount": integer(lo=0),
            "attribution": choice("single", "mixed", "none"),
            "mechanism": choice("clause_driven", "counterparty_insolvency", "other"),
            "L_max_formula_type": choice(*L_MAX_FORMULAS),
            "L_max_value": integer(lo=0),
            "dispute_occurred": boolean,
            "actual_loss_amount": integer(lo=0),
            "realized_ratio_conditional": number(lo=0, hi=1),
            "outcome": choice("임차인전부승", "임차인일부승", "임차인패소", "조정불성립", "절충", "N/A_공식규칙", note_suffix=True),
            "annotator_id": text(),
            "annotation_date": iso_date,
            "notes": text(required=False),
        },
        unique=("clause_id",),
        checks=[clause_id_matches_case, ratio_matches_amounts],
    ),
    "toxic_clauses_100.csv": Schema(
        {
            "no": integer(required=True, lo=1, hi=100),
            "report_section": choice("금전·보증금", "유지보수·원상복구", "계약해지·갱신", "사생활·생활간섭", "기타 절차·권리제한"),
            "clause_text_verbatim": text(),
            "legal_effect_report": text(),
            "legal_effect_note": text(required=False),
            "basis_report": text(required=False),
            "report_ref_no": integer(lo=1),
            "clause_type_auto": choice(*RISK_TYPES),
            "label_status": choice(*LABEL_STATUS),
            "parse_note": text(required=False),
            "source_file": existing_file,
        },
        unique=("no",),
    ),
    "normal_clauses_standard.csv": Schema(
        {
            "id": text(),
            "article": text(),
            "clause_type": choice("정상"),
            "clause_text_clean": text(),
            "ocr_fix": text(required=False),
            "page": integer(required=True, lo=1),
            "paired_toxic_type": choice(*RISK_TYPES, required=False),
            "paired_toxic_nos": int_list,
            "label_status": choice(*LABEL_STATUS),
            "source_file": existing_file,
        },
        unique=("id",),
    ),
    "severity_dataset_adrhome.csv": Schema(
        {
            **SEVERITY_COLUMNS,
            "outcome_side": choice("절충", "임차인승", "임대인승"),
            "source_url": source_ref,
            "notes": text(required=False),
        },
        unique=("case_id",),
    ),
    "severity_dataset_hldcc.csv": Schema(
        {
            **SEVERITY_COLUMNS,
            "ratio_settlement_to_deposit": number(lo=0),
            "outcome_side": choice("절충", "임차인승", "임대인승"),
            "source_url": source_ref,
            "notes": text(required=False),
        },
        unique=("case_id",),
    ),
    "hldcc_case_index.csv": Schema(
        {
            "page": integer(required=True, lo=1),
            "bbsSn": integer(required=True, lo=1),
            "category": text(),
            "title": text(),
            "date": iso_date,
        },
        unique=("bbsSn",),
    ),
}


def _read(name: str) -> tuple[list[Row], list[str]]:
    schema = SCHEMAS[name]
    names = list(schema.columns)
    rows: list[Row] = []
    problems: list[str] = []
    data = (DATA / name).read_bytes()
    try:
        # utf-8-sig: Excel's "CSV UTF-8" adds a BOM, which would otherwise end up in the first header name
        content = data.decode("utf-8-sig")
    except UnicodeDecodeError as e:
        line_no = data[: e.start].count(b"\n") + 1
        return [], [f"{name}:{line_no}: UTF-8로 읽을 수 없음 (한글 엑셀의 기본 CSV는 CP949). 엑셀에서는 'CSV UTF-8(쉼표로 분리)'로 다시 저장하세요"]
    with io.StringIO(content, newline="") as f:
        reader = csv.reader(f)
        header = next(reader, [])
        if header != names:
            missing = [c for c in names if c not in header]
            extra = [c for c in header if c not in names]
            detail = f"없는 열 {missing}, 모르는 열 {extra}" if missing or extra else "열 순서가 다름"
            return [], [f"{name}:1: 헤더가 스키마와 다름 ({detail})"]
        seen: dict[str, dict[object, int]] = {c: {} for c in schema.unique}
        line = reader.line_num + 1
        for raw in reader:
            where = f"{name}:{line}"
            line = reader.line_num + 1
            if len(raw) != len(names):
                key = raw[0] if raw else ""
                problems.append(f"{where}: 열이 {len(raw)}개 (헤더는 {len(names)}개), 첫 칸='{key}'. 이 행은 값이 밀렸을 수 있어 검사하지 않음")
                continue
            row: Row = {}
            for col, cell in zip(names, raw):
                try:
                    row[col] = schema.columns[col](cell)
                except ValueError as e:
                    shown = cell if len(cell) <= 40 else cell[:40] + "…"
                    problems.append(f"{where}: {col}: {e} (값='{shown}')")
            if len(row) < len(names):
                continue
            for col in schema.unique:
                if row[col] in seen[col]:
                    problems.append(f"{where}: {col}='{row[col]}' 가 {seen[col][row[col]]}행과 중복")
                seen[col][row[col]] = int(where.rsplit(":", 1)[1])
            for check in schema.checks:
                if msg := check(row):
                    problems.append(f"{where}: {msg}")
            rows.append(row)
    return rows, problems


def load(name: str) -> list[Row]:
    """Read one CSV in data/labeling/. Raises LabelingError listing every problem if any row breaks the schema."""
    if name not in SCHEMAS:
        raise LabelingError([f"{name}: 스키마가 없음. scripts/data/labeling.py의 SCHEMAS에 추가하세요"])
    rows, problems = _read(name)
    if problems:
        raise LabelingError(problems)
    return rows


def validate_all() -> list[str]:
    problems: list[str] = []
    for path in sorted(DATA.glob("*.csv")):
        if path.name not in SCHEMAS:
            problems.append(f"{path.name}: 스키마가 없음. scripts/data/labeling.py의 SCHEMAS에 추가하세요")
    loaded: dict[str, list[Row]] = {}
    for name in SCHEMAS:
        rows, p = _read(name)
        loaded[name] = rows
        problems += p
    if any(p.startswith("toxic_clauses_100.csv:") for p in problems):
        return problems  # Cross-file check against a partly read file would only add false errors
    toxic_nos = {r["no"] for r in loaded["toxic_clauses_100.csv"]}
    for r in loaded["normal_clauses_standard.csv"]:
        if unknown := [n for n in r["paired_toxic_nos"] if n not in toxic_nos]:
            problems.append(f"normal_clauses_standard.csv: {r['id']}: paired_toxic_nos {unknown} 가 toxic_clauses_100.csv에 없음")
    return problems


if __name__ == "__main__":
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    found = validate_all()
    for p in found:
        print(p)
    if found:
        print(f"\n문제 {len(found)}개. data/labeling/README.md 2절 스키마를 확인하세요")
        sys.exit(1)
    print(f"data/labeling: CSV {len(SCHEMAS)}개 모두 스키마 통과")
