#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=${REPO_ROOT:-$(git rev-parse --show-toplevel)}
cd "$REPO_ROOT"

export GOFLAGS=${GOFLAGS:--buildvcs=false}

echo "# PR 74 Targeted Validation"
echo
printf 'repo: %s\n' "$REPO_ROOT"
printf 'go: %s\n' "$(go version)"
printf 'GOFLAGS: %s\n' "$GOFLAGS"
echo

run() {
  echo
  echo "## $*"
  "$@"
}

run go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http ./pkg/xgoja/hostauth -count=1
run go test ./pkg/gojahttp/auth/... -count=1
run go test ./examples/xgoja/18-express-auth-host/cmd/host ./examples/xgoja/20-express-hello-world/cmd/host ./examples/xgoja/21-generated-host-auth/cmd/host -count=1

echo
if [ -f examples/xgoja/21-generated-host-auth/Makefile ]; then
  echo "## examples/xgoja/21-generated-host-auth make targets"
  grep -E '^[a-zA-Z0-9_.-]+:' examples/xgoja/21-generated-host-auth/Makefile | sed 's/:.*//'
fi
