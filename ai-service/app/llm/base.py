from dataclasses import dataclass
from typing import Protocol

CLAUSE_TYPES = ["선순위근저당", "보증금반환제한", "원상복구과도부담", "묵시적갱신배제", "기타불이익", "정상"]


@dataclass
class Classification:
    clause_type: str
    confidence: float


class LLMError(Exception):
    """Model call/parse failure. The pipeline excludes the clause from scoring and records it in warnings."""


class LLMProvider(Protocol):
    name: str

    async def classify(self, clause: str) -> Classification: ...

    async def aclose(self) -> None: ...
