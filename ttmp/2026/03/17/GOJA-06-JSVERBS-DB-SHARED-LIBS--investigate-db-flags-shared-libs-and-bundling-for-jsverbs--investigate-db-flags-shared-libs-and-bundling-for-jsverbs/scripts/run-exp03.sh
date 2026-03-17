#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${ROOT}/../../../../../.." && pwd)"
DIST_DIR="${ROOT}/exp03-bundled/dist"

mkdir -p "${DIST_DIR}"
bun x esbuild "${ROOT}/exp03-bundled/src/index.js" \
  --bundle \
  --platform=node \
  --format=cjs \
  --external:database \
  --outfile="${DIST_DIR}/bundle.cjs"

go run "${REPO_ROOT}/cmd/jsverbs-example" --dir "${DIST_DIR}" bundled-db count-users Ma --db "${ROOT}/exp03.sqlite"
