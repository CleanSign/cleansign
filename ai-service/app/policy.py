# 주의: LAW_REFERENCE는 유형별 고정 참조다. 법령 MCP로 실시간 검증하기 전까지 조문 번호를 확정 사실로 취급하지 않는다.
"""정량화 정책 (v0). 모든 숫자는 PLACEHOLDER다: 평가 세트가 생기면 보정한다.
근거: data/labeling/README.md (1.2절 청구액 대비 인정비율, 3번 규칙 L_max 공식), docs/06-ai-analysis-plan.md 6-5."""
POLICY_VERSION = "v0-placeholder"

# r_T: 유형별 '청구액 대비 인정비율' 중앙값. 원상복구 0.30, 전체 0.33은 README 1.2절 집계값.
# 나머지 유형은 표본 부족(0~2건)이라 전체 중앙값으로 당겨 쓴다(shrinkage의 가장 단순한 형태).
OVERALL_RATIO = 0.33
RATIO_BY_TYPE: dict[str, float] = {
    "원상복구과도부담": 0.30,
    "묵시적갱신배제": OVERALL_RATIO,
    "보증금반환제한": OVERALL_RATIO,
    "선순위근저당": OVERALL_RATIO,
    "기타불이익": OVERALL_RATIO,
}
# 표본이 적을수록 범위를 넓힌다 (loss_range = expected × [lo, hi])
RANGE_BY_TYPE: dict[str, tuple[float, float]] = {
    "원상복구과도부담": (0.5, 1.5),
}
DEFAULT_RANGE = (0.3, 2.0)

# 보증금 정보를 못 읽었을 때 쓰는 가정값(원). 사실 추출에 실패했음을 warnings에 남긴다.
FALLBACK_DEPOSIT = 0
# RENEWAL_BLOCK 구성 요소 PLACEHOLDER: 이사비용 + 중개수수료 + 공백기간 월세 개월 수
RENEWAL_MOVING_COST = 1_000_000
RENEWAL_BROKERAGE = 500_000
RENEWAL_VACANCY_MONTHS = 1
# 전세는 월세가 없으므로 보증금에 환산율을 곱해 월차임으로 환산(PLACEHOLDER)
JEONSE_TO_MONTHLY_RATE = 0.05 / 12
# RESTORATION fallback: 견적액이 없으면 보증금의 일정 비율을 상한으로 둔다(라벨링 중 확정 예정)
RESTORATION_CAP_RATIO = 0.10
# 지연배상: 지연 일수를 알 수 없으므로 3개월, 연 5%를 가정(PLACEHOLDER)
DEPOSIT_DELAY_DAYS = 90
DEPOSIT_DELAY_RATE = 0.05

# 등급 구간: expected_loss / deposit
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
# LLM 설명 단계가 붙기 전까지의 고정 문구. 조항 내용을 읽고 쓴 설명이 아니라 유형 설명이다.
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
