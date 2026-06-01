#!/usr/bin/env bash
set -euo pipefail

# Demonstrate the current xgoja local-source embedding pipeline with an
# existing smoke-tested jsverbs example. This is an investigation script, not a
# product test.

REPO_ROOT=${REPO_ROOT:-"$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"}
SPEC="$REPO_ROOT/examples/xgoja/07-embedded-jsverbs/xgoja.yaml"
WORKDIR=${WORKDIR:-"$(mktemp -d)"}

cd "$REPO_ROOT"
GOWORK=off go run ./cmd/xgoja build \
  -f "$SPEC" \
  --dry-run \
  --keep-work \
  --work-dir "$WORKDIR" \
  --xgoja-replace "$REPO_ROOT"

printf '\n--- generated files ---\n'
find "$WORKDIR" -maxdepth 4 -type f | sort

printf '\n--- generated go:embed directives ---\n'
rg -n "go:embed|embeddedJSVerbs|xgoja_embed" "$WORKDIR/main.go" "$WORKDIR/xgoja.gen.json" || true

printf '\nworkdir=%s\n' "$WORKDIR"
