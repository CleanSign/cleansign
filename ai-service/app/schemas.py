"""앱(Next.js)과 주고받는 JSON 계약. docs/06-ai-analysis-plan.md 6-2와 같은 모양(camelCase)."""
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel

ClauseType = Literal["선순위근저당", "보증금반환제한", "원상복구과도부담", "묵시적갱신배제", "기타불이익", "정상"]
RiskClauseType = Literal["선순위근저당", "보증금반환제한", "원상복구과도부담", "묵시적갱신배제", "기타불이익"]
Grade = Literal["high", "medium", "low"]
LMaxFormula = Literal["SENIOR_LIEN", "DEPOSIT_DELAY", "RESTORATION", "RENEWAL_BLOCK", "OTHER_CAP"]


class CamelModel(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)


class Facts(CamelModel):
    deposit: int | None = None  # 원
    monthly_rent: int | None = None  # 원
    lease_type: Literal["jeonse", "wolse", "unknown"] = "unknown"


class ClauseResult(CamelModel):
    id: str
    original: str
    clause_type: RiskClauseType
    confidence: float = Field(ge=0, le=1)
    severity: Grade
    l_max: int
    l_max_formula: LMaxFormula
    expected_loss: int
    reason: str
    law_reference: list[str]
    precedents: list[str] = []  # RAG 연결 전까지 빈 목록
    safe_text: str


class AnalysisResult(CamelModel):
    risk_score: int = Field(ge=0, le=100)
    grade: Grade
    loss_amount: int
    loss_range: tuple[int, int]
    facts: Facts
    clauses: list[ClauseResult]
    flow: dict | None = None  # 위험 흐름도: 아직 생성하지 않음
    warnings: list[str] = []
    model_version: str
    policy_version: str
    stats: dict[str, float | int] = {}


class TextRequest(CamelModel):
    text: str = Field(min_length=1, max_length=200_000)
