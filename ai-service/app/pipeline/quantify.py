"""정량화: 순수 함수. LLM을 쓰지 않는다 (data/labeling/README.md 3번 규칙의 L_max 공식, docs 6-5)."""
import math

from app import policy
from app.schemas import Facts, Grade, LMaxFormula

FORMULA_BY_TYPE: dict[str, LMaxFormula] = {
    "선순위근저당": "SENIOR_LIEN",
    "보증금반환제한": "DEPOSIT_DELAY",
    "원상복구과도부담": "RESTORATION",
    "묵시적갱신배제": "RENEWAL_BLOCK",
    "기타불이익": "OTHER_CAP",
}


def l_max(clause_type: str, facts: Facts) -> tuple[int, LMaxFormula]:
    deposit = facts.deposit if facts.deposit is not None else policy.FALLBACK_DEPOSIT
    formula = FORMULA_BY_TYPE[clause_type]
    if formula == "RESTORATION":
        value = deposit * policy.RESTORATION_CAP_RATIO
    elif formula == "DEPOSIT_DELAY":
        value = deposit * policy.DEPOSIT_DELAY_RATE * policy.DEPOSIT_DELAY_DAYS / 365
    elif formula == "RENEWAL_BLOCK":
        monthly = facts.monthly_rent if facts.monthly_rent else deposit * policy.JEONSE_TO_MONTHLY_RATE
        value = policy.RENEWAL_MOVING_COST + policy.RENEWAL_BROKERAGE + monthly * policy.RENEWAL_VACANCY_MONTHS
    else:  # SENIOR_LIEN, OTHER_CAP: 근저당액 등 입력이 없으면 보증금 전액이 fallback (README 3번 규칙)
        value = deposit
    return int(value), formula


def expected_loss(confidence: float, lmax: int, clause_type: str) -> int:
    return int(confidence * lmax * policy.RATIO_BY_TYPE[clause_type])


def loss_range(expected: int, clause_type: str) -> tuple[int, int]:
    lo, hi = policy.RANGE_BY_TYPE.get(clause_type, policy.DEFAULT_RANGE)
    return int(expected * lo), int(expected * hi)


def clause_weight(confidence: float, lmax: int, clause_type: str, deposit: int | None) -> float:
    if not deposit:
        return 0.0
    return confidence * min(1.0, lmax / deposit) * policy.RATIO_BY_TYPE[clause_type]


def severity(expected: int, deposit: int | None) -> Grade:
    if not deposit:
        return "low"
    ratio = expected / deposit
    if ratio >= policy.GRADE_HIGH:
        return "high"
    if ratio >= policy.GRADE_MEDIUM:
        return "medium"
    return "low"


def contract_score(weights: list[float]) -> int:
    """noisy-OR: 조항이 늘면 오르지만 100을 넘지 않는다."""
    survive = math.prod(1 - min(1.0, max(0.0, w)) for w in weights)
    return round(100 * (1 - survive))


def score_grade(score: int) -> Grade:
    if score >= policy.SCORE_GRADE_HIGH:
        return "high"
    if score >= policy.SCORE_GRADE_MEDIUM:
        return "medium"
    return "low"
