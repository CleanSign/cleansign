# Note: LAW_REFERENCE is a fixed per-type reference. Do not treat article numbers as confirmed facts until verified live via the law MCP.
"""Quantification policy (v0). All numbers are PLACEHOLDERs: calibrate once an evaluation set exists.
Basis: data/labeling/README.md (section 1.2 awarded-to-claimed ratio, rule 3 L_max formula), docs/06-ai-analysis-plan.md 6-5."""
POLICY_VERSION = "v0-placeholder"

# r_T: per-type median of the 'awarded-to-claimed ratio'. Restoration 0.30 and overall 0.33 are the README section 1.2 aggregates.
# The remaining types have too few samples (0-2), so they are pulled toward the overall median (the simplest form of shrinkage).
OVERALL_RATIO = 0.33
RATIO_BY_TYPE: dict[str, float] = {
    "원상복구과도부담": 0.30,
    "묵시적갱신배제": OVERALL_RATIO,
    "보증금반환제한": OVERALL_RATIO,
    "선순위근저당": OVERALL_RATIO,
    "기타불이익": OVERALL_RATIO,
}
# The fewer the samples, the wider the range (loss_range = expected × [lo, hi])
RANGE_BY_TYPE: dict[str, tuple[float, float]] = {
    "원상복구과도부담": (0.5, 1.5),
}
DEFAULT_RANGE = (0.3, 2.0)

# Assumed value (KRW) used when the deposit could not be read. The fact-extraction failure is recorded in warnings.
FALLBACK_DEPOSIT = 0
# RENEWAL_BLOCK component PLACEHOLDERs: moving cost + brokerage fee + months of rent for the vacancy period
RENEWAL_MOVING_COST = 1_000_000
RENEWAL_BROKERAGE = 500_000
RENEWAL_VACANCY_MONTHS = 1
# Jeonse has no monthly rent, so convert the deposit to monthly rent using a conversion rate (PLACEHOLDER)
JEONSE_TO_MONTHLY_RATE = 0.05 / 12
# RESTORATION fallback: with no quote amount, cap at a fixed fraction of the deposit (to be finalized during labeling)
RESTORATION_CAP_RATIO = 0.10
# Delay damages: the delay period is unknown, so assume 3 months at 5% per year (PLACEHOLDER)
DEPOSIT_DELAY_DAYS = 90
DEPOSIT_DELAY_RATE = 0.05

# Grade bands: expected_loss / deposit
GRADE_HIGH = 0.05
GRADE_MEDIUM = 0.015
SCORE_GRADE_HIGH = 60
SCORE_GRADE_MEDIUM = 30

LAW_REFERENCE: dict[str, list[str]] = {
    "선순위근저당": ["주택임대차보호법 제3조의2"],
    "보증금반환제한": ["주택임대차보호법 제10조"],
    "원상복구과도부담": ["주택임대차보호법 제10조", "LH 임대주택 수선비 부담 및 원상복구 기준"],
    "묵시적갱신배제": ["주택임대차보호법 제6조의3", "주택임대차보호법 제10조"],
    "기타불이익": ["주택임대차보호법 제10조"],
}
# Fixed text until the LLM explanation step is added. It describes the type, not an explanation written from the clause content.
REASON_TEMPLATE: dict[str, str] = {
    "선순위근저당": "선순위 담보가 있으면 경매 시 보증금을 돌려받지 못할 수 있는 유형의 조항입니다.",
    "보증금반환제한": "조건을 달아 보증금 반환을 미루거나 공제하는 유형의 조항입니다. 임차인에게 불리한 특약은 무효일 수 있습니다.",
    "원상복구과도부담": "통상적인 사용에 따른 손모까지 임차인에게 부담시킬 수 있는 유형의 조항입니다.",
    "묵시적갱신배제": "임차인의 계약갱신요구권 등 갱신 관련 권리를 제한하는 유형의 조항입니다.",
    "기타불이익": "위약금, 비용 전가 등 임차인에게 불리할 수 있는 유형의 조항입니다.",
}
SAFE_TEXT_TEMPLATE: dict[str, str] = {
    "선순위근저당": "임대인은 잔금 지급일까지 선순위 근저당 등 담보를 설정하지 않으며, 위반 시 임차인은 계약을 해제할 수 있다.",
    "보증금반환제한": "임대인은 임대차 종료 시 임차인의 목적물 반환과 동시에 보증금을 반환한다.",
    "원상복구과도부담": "통상적인 사용에 따른 손모와 노후는 임대인이 부담하고, 임차인의 고의·과실로 인한 훼손만 복구한다.",
    "묵시적갱신배제": "임차인은 주택임대차보호법에 따른 계약갱신요구권을 가진다.",
    "기타불이익": "임차인에게 법령보다 불리한 비용·위약금 부담을 정하지 않는다.",
}
