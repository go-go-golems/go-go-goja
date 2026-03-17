#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${ROOT}/../../../../../.." && pwd)"

go run "${REPO_ROOT}/cmd/jsverbs-example" --dir "${ROOT}/exp02-cross-file-sections" list
