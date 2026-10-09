"""Feed the normal clauses of the standard contract (normal_clauses_standard.csv) to the classifier and count false positives (normal -> risky).
Usage: FAST=1 python3 normal_probe.py [model name]"""
import csv, os, sys, pathlib
os.environ.setdefault("FAST", "1")
sys.argv = [sys.argv[0], *(sys.argv[1:2] or ["gemma4:12b"]), "v1"]
import classify_probe as cp

ROOT = pathlib.Path(__file__).resolve().parents[2]
rows = list(csv.DictReader(open(ROOT / "data/labeling/normal_clauses_standard.csv", encoding="utf-8")))
fp, lat = 0, []
for r in rows:
    out, s = cp.ask(r["clause_text_clean"]); lat.append(s)
    bad = out["clause_type"] != "정상"; fp += bad
    print(f"{'FP' if bad else 'ok'} {r['id']:10} pred={out['clause_type']:8} conf={out['confidence']:.2f} {s:4.1f}s | {r['clause_text_clean'][:60]}")
print(f"\n{cp.MODEL}: 오탐 {fp}/{len(rows)} ({fp/len(rows):.0%}), 평균 {sum(lat)/len(lat):.1f}s")
