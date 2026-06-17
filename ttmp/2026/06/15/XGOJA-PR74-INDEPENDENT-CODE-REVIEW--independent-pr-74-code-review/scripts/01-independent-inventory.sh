#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
BASE="${BASE_REF:-origin/main}"
HEAD_REF="${HEAD_REF:-HEAD}"

printf '# Independent PR 74 inventory\n\n'
printf 'Date: %s\n' "$(date -Is)"
printf 'Root: %s\n' "$ROOT"
printf 'Branch: %s\n' "$(git branch --show-current)"
printf 'Base ref: %s\n' "$BASE"
printf 'Base commit: %s\n' "$(git rev-parse "$BASE")"
printf 'Head ref: %s\n' "$HEAD_REF"
printf 'Head commit: %s\n' "$(git rev-parse "$HEAD_REF")"
printf '\n## Worktree status\n\n'
git status --short
printf '\n## Diff stat (%s...%s)\n\n' "$BASE" "$HEAD_REF"
git diff --stat "$BASE...$HEAD_REF"
printf '\n## Name status\n\n'
git diff --name-status "$BASE...$HEAD_REF"
printf '\n## Changed Go packages\n\n'
git diff --name-only "$BASE...$HEAD_REF" -- '*.go' \
  | xargs -r dirname \
  | sort -u
printf '\n## Largest changed files by added+deleted lines\n\n'
git diff --numstat "$BASE...$HEAD_REF" \
  | awk 'BEGIN{printf "%8s %8s %8s %s\n", "added", "deleted", "total", "path"} $1 ~ /^[0-9]+$/ && $2 ~ /^[0-9]+$/ {printf "%8d %8d %8d %s\n", $1, $2, $1+$2, $3}' \
  | sort -k3,3nr -k4,4 \
  | head -80
printf '\n## Test files changed/added\n\n'
git diff --name-only "$BASE...$HEAD_REF" -- '*_test.go' | sort
printf '\n## Documentation/examples changed\n\n'
git diff --name-only "$BASE...$HEAD_REF" -- 'pkg/doc/*' 'examples/**' 'cmd/**/doc/*' '*.md' | sort
