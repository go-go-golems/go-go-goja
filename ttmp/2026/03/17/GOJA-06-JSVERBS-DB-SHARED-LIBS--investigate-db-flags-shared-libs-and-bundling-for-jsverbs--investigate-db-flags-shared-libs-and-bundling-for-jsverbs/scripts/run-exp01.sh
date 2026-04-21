#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${ROOT}/../../../../../.." && pwd)"

go run "${REPO_ROOT}/cmd/jsverbs-example" --dir "${ROOT}/exp01-unbundled-db" db-demo list-users A --db "${ROOT}/exp01.sqlite"
