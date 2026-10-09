import re

from app.schemas import Facts

_AMOUNT = r"(?:금\s*)?(\d[\d,]*(?:\.\d+)?)\s*(억|만)?\s*(?:(\d[\d,]*)\s*(만))?\s*원?"


def _won(num: str, unit: str | None, num2: str | None, unit2: str | None) -> int:
    n = float(num.replace(",", ""))
    total = n * {"억": 100_000_000, "만": 10_000}.get(unit or "", 1)
    if num2 and unit2:
        total += float(num2.replace(",", "")) * 10_000
    return int(total)


def _find(label: str, text: str) -> int | None:
    m = re.search(label + r"\s*[:：]?\s*" + _AMOUNT, text)
    if not m:
        return None
    value = _won(m.group(1), m.group(2), m.group(3), m.group(4))
    return value if value > 0 else None


def extract_facts(text: str) -> Facts:
    """Read numeric notation only (e.g. '보증금 1억 5,000만원', '월세 50만원'). Korean numerals ('일억오천만원') are not parsed and yield None."""
    deposit = _find(r"(?:임대차\s*)?보증금", text)
    rent = _find(r"(?:월\s*세|차임|월\s*차임)", text)
    lease = "wolse" if rent else ("jeonse" if deposit else "unknown")
    return Facts(deposit=deposit, monthly_rent=rent, lease_type=lease)
