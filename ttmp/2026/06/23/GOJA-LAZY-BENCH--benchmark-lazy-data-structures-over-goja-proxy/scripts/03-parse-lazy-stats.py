#!/usr/bin/env python3
# 03-parse-lazy-stats.py — Stage 1 of the lazy-benchmark report pipeline.
#
# Parses Go `go test -bench` raw output (which includes a custom provider/op
# metric between ns/op and B/op) into a structured stats JSON.
#
#   raw  (various/01-lazy-bench-raw.txt)
#   -> stats (various/03-stats.json)
#
#   python3 scripts/03-parse-lazy-stats.py
import re, sys, json, statistics
from pathlib import Path

HERE = Path(__file__).resolve().parent
TICKET = HERE.parent
VAR = TICKET / "various"
RAW = VAR / "01-lazy-bench-raw.txt"
OUT = VAR / "03-stats.json"

# Lines come in two shapes:
#   name  iters  ns ns/op  B B/op  allocs allocs/op                       (no provider)
#   name  iters  ns ns/op  P provider/op  B B/op  allocs allocs/op       (with provider)
PAT = re.compile(
    r'^(Benchmark\S+)\s+\d+\s+([\d.]+)\s+ns/op'
    r'(?:\s+([\d.]+)\s+provider/op)?'
    r'\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op')

def main():
    text = RAW.read_text()
    samples = {}
    for line in text.splitlines():
        m = PAT.match(line.strip())
        if not m:
            continue
        name = m.group(1)
        ns = float(m.group(2))
        prov = float(m.group(3)) if m.group(3) is not None else 0.0
        b = int(m.group(4))
        al = int(m.group(5))
        samples.setdefault(name, []).append((ns, b, al, prov))

    out = {}
    for name, rows in samples.items():
        ns = [r[0] for r in rows]
        provs = [r[3] for r in rows]
        b = rows[0][1]
        al = rows[0][2]
        mean = statistics.mean(ns)
        std = statistics.stdev(ns) if len(ns) > 1 else 0.0
        # provider/op is a deterministic invariant: take the max across samples
        # (cached reads report ~0 after the first; the invariant is the ceiling).
        prov = max(provs) if provs else 0.0
        out[name] = {
            "name": name,
            "mean": mean, "std": std,
            "cv": (std / mean * 100) if mean else 0.0,
            "min": min(ns), "max": max(ns),
            "median": statistics.median(ns),
            "samples": len(ns), "ns": ns,
            "B": b, "allocs": al,
            "provider": round(prov, 4),
        }

    OUT.write_text(json.dumps(out, indent=1))
    print(f"{'benchmark':<58}{'mean':>10}{'B':>7}{'allocs':>7}{'prov/op':>9}")
    for name, s in out.items():
        print(f"{name:<58}{s['mean']:10.0f}{s['B']:7d}{s['allocs']:7d}{s['provider']:9.2f}")
    print(f"\nwrote {OUT.relative_to(TICKET.parent.parent.parent)}")

if __name__ == "__main__":
    sys.exit(main())
