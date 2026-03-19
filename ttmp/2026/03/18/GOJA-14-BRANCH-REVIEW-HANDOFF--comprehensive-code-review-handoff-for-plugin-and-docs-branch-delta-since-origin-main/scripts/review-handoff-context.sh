#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
cd "$ROOT"

echo "== branch =="
git rev-parse --abbrev-ref HEAD

echo
echo "== merge-base with origin/main =="
git merge-base origin/main HEAD

echo
echo "== diff stat origin/main...HEAD =="
git diff --stat origin/main...HEAD

echo
echo "== changed files origin/main...HEAD =="
git diff --name-only origin/main...HEAD

echo
echo "== commits origin/main..HEAD =="
git log --oneline --decorate origin/main..HEAD
