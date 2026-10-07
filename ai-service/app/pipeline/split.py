import re

# "제1조", "1.", "①", "특약" 같은 조항 머리말에서 끊는다. 규칙 기반 v0.
HEAD = re.compile(r"(?m)^\s*(?:제\s*\d+\s*조|\d{1,2}\s*[.)]\s|[①-⑳]|[-•·]\s)")
MIN_LEN = 8
MAX_LEN = 600


def split_clauses(text: str) -> list[str]:
    text = re.sub(r"[ \t]+", " ", text.replace("\r", ""))
    starts = [m.start() for m in HEAD.finditer(text)]
    if not starts:
        parts = [p for p in re.split(r"\n\s*\n", text)]
    else:
        bounds = ([0] if starts[0] > 0 else []) + starts + [len(text)]
        parts = [text[a:b] for a, b in zip(bounds, bounds[1:])]
    out: list[str] = []
    for p in parts:
        p = re.sub(r"\s+", " ", p).strip()
        if len(p) >= MIN_LEN:
            out.append(p[:MAX_LEN])
    return out
