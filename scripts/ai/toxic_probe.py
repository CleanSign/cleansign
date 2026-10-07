"""독소 조항 100개(toxic_clauses_100.csv)를 분류기에 넣어 탐지율(정상이 아니라고 본 비율)을 잰다.
효력이 유효/유효 가능/기각/권고적인 경계 사례는 제외한다. 유형 일치는 자동 매핑(검수 전)이라 참고용.
사용: FAST=1 python3 toxic_probe.py [모델명]"""
import csv, os, sys, pathlib, collections
os.environ.setdefault("FAST", "1")
sys.argv = [sys.argv[0], *(sys.argv[1:2] or ["gemma4:12b"]), "v1"]
import classify_probe as cp

SKIP = {"유효", "유효 가능", "기각", "권고적"}
ROOT = pathlib.Path(__file__).resolve().parents[2]
rows = [r for r in csv.DictReader(open(ROOT / "data/labeling/toxic_clauses_100.csv", encoding="utf-8")) if r["legal_effect_report"] not in SKIP]
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
