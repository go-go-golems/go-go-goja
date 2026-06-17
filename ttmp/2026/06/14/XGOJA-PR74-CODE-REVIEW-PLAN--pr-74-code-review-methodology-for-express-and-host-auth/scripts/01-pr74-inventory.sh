#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=${REPO_ROOT:-$(git rev-parse --show-toplevel)}
cd "$REPO_ROOT"

BASE=${BASE:-origin/main}
HEAD_REF=${HEAD_REF:-HEAD}

echo "# PR 74 Inventory"
echo
printf 'repo: %s\n' "$REPO_ROOT"
printf 'base: %s (%s)\n' "$BASE" "$(git rev-parse --short "$BASE")"
printf 'head: %s (%s)\n' "$HEAD_REF" "$(git rev-parse --short "$HEAD_REF")"
printf 'branch: %s\n' "$(git branch --show-current)"
echo

echo "## PR metadata"
if command -v gh >/dev/null 2>&1; then
  gh pr view 74 --json number,title,state,url,headRefName,baseRefName,author,commits,files \
    --jq '{number,title,state,url,headRefName,baseRefName,author:.author.login,commitCount:(.commits|length),fileCount:(.files|length)}'
else
  echo "gh not found"
fi

echo
echo "## Diff stat"
git diff --stat "$BASE...$HEAD_REF"

echo
echo "## Changed files by status"
git diff --name-status "$BASE...$HEAD_REF"

echo
echo "## Changed Go packages"
git diff --name-only "$BASE...$HEAD_REF" -- '*.go' \
  | xargs -r dirname \
  | sort -u \
  | sed 's#^#./#'

echo
echo "## Largest changed files by line delta"
git diff --numstat "$BASE...$HEAD_REF" \
  | awk 'BEGIN {printf "%8s %8s %8s %s\n", "added", "deleted", "delta", "file"} {if ($1 ~ /^[0-9]+$/ && $2 ~ /^[0-9]+$/) printf "%8d %8d %8d %s\n", $1, $2, $1+$2, $3}' \
  | sort -k3,3nr \
  | head -40

echo
echo "## Test files changed"
git diff --name-only "$BASE...$HEAD_REF" -- '*_test.go' | sort

echo
echo "## Docs/examples changed"
git diff --name-only "$BASE...$HEAD_REF" \
  | grep -E '(^pkg/doc/|^cmd/.*/doc/|^examples/|README\.md$)' \
  | sort || true
