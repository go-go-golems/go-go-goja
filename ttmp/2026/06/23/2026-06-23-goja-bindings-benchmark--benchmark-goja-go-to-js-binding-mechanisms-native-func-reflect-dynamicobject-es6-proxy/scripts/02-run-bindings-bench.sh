#!/usr/bin/env bash
# 02-run-bindings-bench.sh — reproduce the goja binding-mechanism benchmarks.
#
# Run from the go-go-goja module root (so ./perf/goja resolves):
#
#   cd go-go-goja
#   bash ../<ticket>/scripts/02-run-bindings-bench.sh /tmp/out
#
# Produces:
#   $1/bindings-bench-raw.txt   — raw `go test -bench` output
#   $1/bindings-benchstat.txt   — benchstat-aggregated medians
set -euo pipefail

OUT="${1:-/tmp/goja-bindings-bench}"
mkdir -p "$OUT"

echo "==> Running benchmarks (count=5, benchtime=1s). Expect ~2-3 min."
go test ./perf/goja \
  -run '^$' \
  -bench 'BindingFunctionCall|BindingPropertyGet|BindingArrayIndexGet|ProxyDispatchDetail' \
  -benchtime=1s -count=5 -benchmem \
  | tee "$OUT/bindings-bench-raw.txt"

echo "==> Aggregating with benchstat."
if ! command -v benchstat >/dev/null 2>&1; then
  go install golang.org/x/perf/cmd/benchstat@latest
fi
benchstat "$OUT/bindings-bench-raw.txt" | tee "$OUT/bindings-benchstat.txt"

echo "==> Done. Results in $OUT"
