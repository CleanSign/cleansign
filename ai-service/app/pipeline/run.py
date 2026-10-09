import asyncio
import time
from collections.abc import Awaitable, Callable

from app import policy
from app.llm.base import LLMError, LLMProvider
from app.pipeline import quantify
from app.pipeline.facts import extract_facts
from app.pipeline.mask import mask_pii
from app.pipeline.split import split_clauses
from app.schemas import AnalysisResult, ClauseResult

MAX_CLAUSES = 60

# (stage, completed count, total count). Stages: splitting → classifying → scoring
ProgressFn = Callable[[str, int, int], Awaitable[None]]


async def analyze_text(
    raw_text: str,
    provider: LLMProvider,
    concurrency: int = 1,
    on_progress: ProgressFn | None = None,
) -> AnalysisResult:
    async def report(stage: str, done: int = 0, total: int = 0) -> None:
        if on_progress:
            await on_progress(stage, done, total)

    t0 = time.perf_counter()
    await report("splitting")
    warnings: list[str] = []

    text = mask_pii(raw_text)  # Always mask before sending to the model
    facts = extract_facts(text)
    if facts.deposit is None:
        warnings.append("보증금을 읽지 못해 금액 산출이 제한됩니다.")

    clauses = split_clauses(text)
    if len(clauses) > MAX_CLAUSES:
        warnings.append(f"조항이 {len(clauses)}개라 앞의 {MAX_CLAUSES}개만 분석했습니다.")
        clauses = clauses[:MAX_CLAUSES]
    if not clauses:
        warnings.append("분석할 조항을 찾지 못했습니다.")

    sem = asyncio.Semaphore(max(1, concurrency))
    finished = 0
    await report("classifying", 0, len(clauses))

    async def classify(c: str):
        nonlocal finished
        async with sem:
            try:
                return await provider.classify(c)
            except LLMError as e:
                return e
            finally:
                finished += 1
                await report("classifying", finished, len(clauses))

    classified = await asyncio.gather(*(classify(c) for c in clauses))

    results: list[ClauseResult] = []
    weights: list[float] = []
    failed = 0
    for i, (text_i, cls) in enumerate(zip(clauses, classified), start=1):
        if isinstance(cls, LLMError):
            failed += 1
            continue
        if cls.clause_type == "정상":
            continue
        ctype = cls.clause_type
        lmax, formula = quantify.l_max(ctype, facts)
        loss = quantify.expected_loss(cls.confidence, lmax, ctype)
        results.append(
            ClauseResult(
                id=f"c{i}",
                original=text_i,
                clause_type=ctype,
                confidence=cls.confidence,
                severity=quantify.severity(loss, facts.deposit),
                l_max=lmax,
                l_max_formula=formula,
                expected_loss=loss,
                reason=policy.REASON_TEMPLATE[ctype],
                law_reference=policy.LAW_REFERENCE[ctype],
                safe_text=policy.SAFE_TEXT_TEMPLATE[ctype],
            )
        )
        weights.append(quantify.clause_weight(cls.confidence, lmax, ctype, facts.deposit))
    if failed:
        warnings.append(f"{failed}개 조항은 모델 호출에 실패해 분석에서 제외했습니다.")
    if results and not facts.deposit:
        warnings.append("보증금이 없어 위험 점수를 계산하지 못했습니다(금액은 0으로 표시).")

    await report("scoring", len(clauses), len(clauses))
    score = quantify.contract_score(weights)
    total = sum(r.expected_loss for r in results)
    lo = hi = 0
    for r in results:
        a, b = quantify.loss_range(r.expected_loss, r.clause_type)
        lo += a
        hi += b

    return AnalysisResult(
        risk_score=score,
        grade=quantify.score_grade(score),
        loss_amount=total,
        loss_range=(lo, hi),
        facts=facts,
        clauses=results,
        warnings=warnings,
        model_version=f"{provider.name}:{getattr(provider, 'model', '-')}",
        policy_version=policy.POLICY_VERSION,
        stats={"clauses": len(clauses), "risky": len(results), "failed": failed,
               "elapsedMs": int((time.perf_counter() - t0) * 1000)},
    )
