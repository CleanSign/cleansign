import re

# Mask personal information in text sent outside. Rule-based, so incomplete (names and detailed addresses are not masked).
PATTERNS: list[tuple[re.Pattern[str], str]] = [
    (re.compile(r"\b\d{6}\s*-\s*[1-4]\d{6}\b"), "[주민번호]"),
    (re.compile(r"\b0\d{1,2}[-.\s]?\d{3,4}[-.\s]?\d{4}\b"), "[전화번호]"),
    (re.compile(r"\b\d{2,6}-\d{2,6}-\d{2,8}(?:-\d{1,4})?\b"), "[계좌번호]"),
    (re.compile(r"[\w.+-]+@[\w-]+\.[\w.-]+"), "[이메일]"),
]


def mask_pii(text: str) -> str:
    for pattern, repl in PATTERNS:
        text = pattern.sub(repl, text)
    return text
