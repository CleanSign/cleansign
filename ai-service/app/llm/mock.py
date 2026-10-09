import asyncio

from app.llm.base import Classification

# Temporary keyword-based classifier. It is not meant to be accurate; it exists to run the server/tests without a model.
RULES: list[tuple[str, tuple[str, ...]]] = [
    ("묵시적갱신배제", ("갱신요구권", "갱신청구권", "갱신을 요구하지", "갱신하지 않")),
    ("원상복구과도부담", ("원상복구", "원상회복", "모든 수리", "수리비용", "시설물")),
    ("보증금반환제한", ("보증금 반환", "보증금을 반환", "보증금 공제", "신규 임차인")),
    ("선순위근저당", ("근저당", "선순위")),
    ("기타불이익", ("위약금", "과태료", "일방적으로")),
]


class MockProvider:
    name = "mock"

    def __init__(self, delay_s: float = 0):
        self.delay_s = delay_s

    async def classify(self, clause: str) -> Classification:
        if self.delay_s:
            await asyncio.sleep(self.delay_s)
        for ctype, words in RULES:
            if any(w in clause for w in words):
                return Classification(ctype, 0.7)
        return Classification("정상", 0.7)

    async def aclose(self) -> None:
        pass
