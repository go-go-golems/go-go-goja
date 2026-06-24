#!/usr/bin/env python3
# 03-parse-bench-stats.py
#
# Stage 1 of the report pipeline. Parses Go's `go test -bench` raw output into a
# structured statistics JSON (mean / stddev / CV / min / max / median, plus the
# raw sample list and the deterministic B/op and allocs/op invariants).
#
#   raw  (various/01-bindings-bench-raw.txt)
#   -> stats (various/03-stats.json)
#
# Run from anywhere:
#   python3 scripts/03-parse-bench-stats.py
import re, sys, json, statistics
from pathlib import Path

HERE = Path(__file__).resolve().parent
TICKET = HERE.parent
VAR = TICKET / "various"
RAW = VAR / "01-bindings-bench-raw.txt"
OUT = VAR / "03-stats.json"

PAT = re.compile(
    r'^(Benchmark\S+)\s+\d+\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op')

def main():
    text = RAW.read_text()
    samples = {}
    for line in text.splitlines():
        m = PAT.match(line.strip())
        if not m:
            continue
        name = m.group(1)
        ns = float(m.group(2))
        b = int(m.group(3))
        al = int(m.group(4))
        samples.setdefault(name, []).append((ns, b, al))

    out = {}
    for name, rows in samples.items():
        ns = [r[0] for r in rows]
        b = rows[0][1]
        al = rows[0][2]
        mean = statistics.mean(ns)
        std = statistics.stdev(ns) if len(ns) > 1 else 0.0
        out[name] = {
            "name": name,
            "mean": mean, "std": std,
            "cv": (std / mean * 100) if mean else 0.0,
            "min": min(ns), "max": max(ns),
            "median": statistics.median(ns),
            "samples": len(ns), "ns": ns,
            "B": b, "allocs": al,  # deterministic invariants (identical across samples)
        }

    OUT.write_text(json.dumps(out, indent=1))
    # human-readable echo
    print(f"{'benchmark':<54}{'mean':>9}{'std':>8}{'cv%':>6}{'min':>8}{'max':>8}{'B':>5}{'alloc':>5}")
    for name, s in out.items():
        print(f"{name:<54}{s['mean']:9.1f}{s['std']:8.1f}{s['cv']:6.1f}"
              f"{s['min']:8.1f}{s['max']:8.1f}{s['B']:5d}{s['allocs']:5d}")
    print(f"\nwrote {OUT.relative_to(TICKET.parent.parent.parent)}")
    print("NOTE: B/op and allocs/op are identical across all samples for every "
          "benchmark — they are code-path invariants, not measurements.")

if __name__ == "__main__":
    sys.exit(main())
