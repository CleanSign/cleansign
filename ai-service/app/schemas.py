"""JSON contract exchanged with the server (Go). Same shape as docs/06-ai-analysis-plan.md 6-2 (camelCase)."""
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
    deposit: int | None = None  # KRW
    monthly_rent: int | None = None  # KRW
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
    precedents: list[str] = []  # Empty list until RAG is connected
    safe_text: str


class AnalysisResult(CamelModel):
    risk_score: int = Field(ge=0, le=100)
    grade: Grade
    loss_amount: int
    loss_range: tuple[int, int]
    facts: Facts
    clauses: list[ClauseResult]
    flow: dict | None = None  # Risk flow chart: not generated yet
    warnings: list[str] = []
    model_version: str
    policy_version: str
    stats: dict[str, float | int] = {}


class TextRequest(CamelModel):
    text: str = Field(min_length=1, max_length=200_000)
