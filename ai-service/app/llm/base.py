from dataclasses import dataclass
from typing import Protocol

CLAUSE_TYPES = ["선순위근저당", "보증금반환제한", "원상복구과도부담", "묵시적갱신배제", "기타불이익", "정상"]


@dataclass
class Classification:
    clause_type: str
    confidence: float


class LLMError(Exception):
    """모델 호출/파싱 실패. 파이프라인은 해당 조항을 점수에서 제외하고 warnings에 남긴다."""


class LLMProvider(Protocol):
    name: str

    async def classify(self, clause: str) -> Classification: ...

    async def aclose(self) -> None: ...
