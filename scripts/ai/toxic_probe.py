"""Feed 100 toxic clauses (toxic_clauses_100.csv) to the classifier and measure the detection rate (share judged not normal).
Borderline cases whose validity is valid / possibly valid / dismissed / advisory are excluded. Type match uses automatic mapping (before review), so it is for reference only.
Usage: FAST=1 python3 toxic_probe.py [model name]"""
import os, sys, collections
os.environ.setdefault("FAST", "1")
sys.argv = [sys.argv[0], *(sys.argv[1:2] or ["gemma4:12b"]), "v1"]
import classify_probe as cp
from labeling import load  # classify_probe puts scripts/data on sys.path

SKIP = {"유효", "유효 가능", "기각", "권고적"}
rows = [r for r in load("toxic_clauses_100.csv") if r["legal_effect_report"] not in SKIP]
hit, same, lat, miss = 0, 0, [], []
by = collections.defaultdict(lambda: [0, 0])
for r in rows:
    out, s = cp.ask(r["clause_text_verbatim"]); lat.append(s)
    det = out["clause_type"] != "정상"; hit += det; same += out["clause_type"] == r["clause_type_auto"]
    by[r["report_section"]][0] += det; by[r["report_section"]][1] += 1
    if not det: miss.append((r["no"], r["clause_text_verbatim"][:70]))
    print(f"{'hit ' if det else 'MISS'} #{r['no']:>3} pred={out['clause_type']:8} auto={r['clause_type_auto']:8} {s:4.1f}s | {r['clause_text_verbatim'][:50]}", flush=True)
n = len(rows)
print(f"\n{cp.MODEL}: 탐지 {hit}/{n} ({hit/n:.0%}), 자동매핑과 유형 일치 {same}/{n} (참고용), 평균 {sum(lat)/n:.1f}s")
for k, (h, t) in by.items(): print(f"  {k}: {h}/{t}")
print("놓친 조항:"); [print("  ", m) for m in miss]
