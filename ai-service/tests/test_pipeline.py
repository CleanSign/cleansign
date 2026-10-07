from app.llm.base import LLMError
from app.llm.mock import MockProvider
from app.pipeline import quantify
from app.pipeline.facts import extract_facts
from app.pipeline.mask import mask_pii
from app.pipeline.run import analyze_text
from app.pipeline.split import split_clauses
from app.schemas import Facts

CONTRACT = """주택임대차계약서
보증금 1억 5,000만원, 월세 50만원
제1조 임대인은 임차인에게 위 주택을 인도한다.
제2조 임차인은 퇴거 시 모든 시설물을 입주 당시 상태로 원상복구하여야 한다.
제3조 임차인은 전세계약갱신청구권을 요구하지 않는다.
제4조 차임은 매월 말일에 지급한다.
임대인 연락처 010-1234-5678, 주민번호 900101-1234567
"""


def test_mask_pii():
    out = mask_pii("연락처 010-1234-5678 주민 900101-1234567 메일 a@b.com")
    assert "010-1234" not in out and "900101" not in out and "a@b.com" not in out


def test_extract_facts_numeric():
    f = extract_facts("보증금 1억 5,000만원 월세 50만원")
    assert f.deposit == 150_000_000 and f.monthly_rent == 500_000 and f.lease_type == "wolse"


def test_extract_facts_unknown():
    assert extract_facts("보증금은 별도 협의").deposit is None


def test_split_by_article():
    parts = split_clauses(CONTRACT)
    assert any("원상복구" in p for p in parts)
    assert sum(p.startswith("제") for p in parts) >= 4


def test_score_noisy_or_bounded():
    assert quantify.contract_score([]) == 0
    assert quantify.contract_score([0.5, 0.5]) == 75
    assert quantify.contract_score([1.0, 1.0, 1.0]) == 100


def test_l_max_fallback_without_deposit():
    lmax, formula = quantify.l_max("원상복구과도부담", Facts())
    assert lmax == 0 and formula == "RESTORATION"


async def test_analyze_end_to_end_with_mock():
    res = await analyze_text(CONTRACT, MockProvider())
    types = {c.clause_type for c in res.clauses}
    assert types == {"원상복구과도부담", "묵시적갱신배제"}
    assert res.facts.deposit == 150_000_000
    assert res.loss_amount == sum(c.expected_loss for c in res.clauses) > 0
    assert 0 < res.risk_score <= 100
    assert res.loss_range[0] <= res.loss_amount <= res.loss_range[1]


async def test_llm_failure_is_reported_not_raised():
    class Broken:
        name = "broken"

        async def classify(self, clause):
            raise LLMError("boom")

        async def aclose(self):
            pass

    res = await analyze_text(CONTRACT, Broken())
    assert res.clauses == [] and res.stats["failed"] > 0
    assert any("모델 호출에 실패" in w for w in res.warnings)
