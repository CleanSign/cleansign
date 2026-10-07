import re

# 외부로 나가는 텍스트에서 개인정보를 가린다. 규칙 기반이라 완전하지 않다(이름·상세주소는 가리지 못함).
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
