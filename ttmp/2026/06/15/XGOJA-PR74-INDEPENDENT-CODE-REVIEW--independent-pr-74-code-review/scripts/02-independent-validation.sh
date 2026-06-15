#!/usr/bin/env bash
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
export GOFLAGS="${GOFLAGS:--buildvcs=false}"
printf '# Independent PR 74 validation\n\n'
printf 'Date: %s\n' "$(date -Is)"
printf 'Root: %s\n' "$ROOT"
printf 'Go: %s\n' "$(go version)"
printf 'GOFLAGS: %s\n' "$GOFLAGS"
printf '\n## Targeted core/package tests\n\n'
go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http ./pkg/xgoja/hostauth -count=1
printf '\n## Auth package tests\n\n'
go test ./pkg/gojahttp/auth/... -count=1
printf '\n## Example host compile tests\n\n'
go test ./examples/xgoja/18-express-auth-host/cmd/host ./examples/xgoja/20-express-hello-world/cmd/host ./examples/xgoja/21-generated-host-auth/cmd/host -count=1
printf '\n## Express auth host smoke\n\n'
make -C examples/xgoja/18-express-auth-host smoke
printf '\n## go vet focused packages\n\n'
go vet ./pkg/gojahttp/... ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http
