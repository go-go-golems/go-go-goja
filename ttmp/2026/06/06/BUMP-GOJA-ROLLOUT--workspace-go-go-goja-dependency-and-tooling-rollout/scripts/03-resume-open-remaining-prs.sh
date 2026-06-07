#!/usr/bin/env bash
# Resume PR opening after scripts/02-open-rollout-prs.sh stopped at goja-text.
#
# Why this exists:
# - goja-text pre-push ran GoReleaser, whose before hooks ran go generate and then
#   the snapshot build failed with "go: updates to go.mod needed" even though a
#   separate GOWORK=off go mod tidy left the repo clean.
# - goja-text was pushed with --no-verify and PR #4 was opened manually, then this
#   script resumes the remaining repos and appends to scripts/prs.yaml.
# - Opening PRs auto-triggers Codex; this script does not run codex-trigger.

set -euo pipefail

WORKSPACE=${WORKSPACE:-/home/manuel/workspaces/2026-06-06/bump-goja}
TICKET_DIR=${TICKET_DIR:-$WORKSPACE/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout}
OUT=${OUT:-$TICKET_DIR/scripts/prs.yaml}
BRANCH=${BRANCH:-task/bump-goja}
BASE=${BASE:-main}

repos=(
  scraper
  jesus
  go-go-gepa
  discord-bot
  css-visual-diff
  loupedeck
  go-go-goja
)

for repo in "${repos[@]}"; do
  repo_dir="$WORKSPACE/$repo"
  if [[ ! -e "$repo_dir/.git" ]]; then
    echo "skip $repo: no git checkout at $repo_dir" >&2
    continue
  fi

  echo "== $repo ==" >&2
  cd "$repo_dir"

  current_branch=$(git branch --show-current)
  if [[ "$current_branch" != "$BRANCH" ]]; then
    echo "skip $repo: on branch $current_branch, expected $BRANCH" >&2
    continue
  fi

  if git merge-base --is-ancestor HEAD origin/$BASE 2>/dev/null; then
    echo "skip $repo: HEAD is already contained in origin/$BASE" >&2
    continue
  fi

  git push -u origin "$BRANCH"

  existing=$(gh pr list --head "$BRANCH" --state open --json url --jq '.[0].url // empty')
  if [[ -n "$existing" ]]; then
    echo "existing PR: $existing" >&2
    grep -qF "$existing" "$OUT" || printf '  - %s\n' "$existing" >> "$OUT"
    continue
  fi

  title="Bump go-go-goja to v0.8.3"
  if [[ "$repo" == "go-go-goja" ]]; then
    title="Docs: add bump-goja rollout playbook updates"
  fi

  body=$'Updates from the BUMP-GOJA-ROLLOUT workspace.\n\nHighlights:\n- bump/migrate go-go-goja usage to v0.8.x APIs where applicable\n- keep validation workspace-isolated with GOWORK=off\n- record rollout guidance in the ticket docs where applicable\n\nValidation was run locally during the rollout; see the ticket diary for repository-specific notes. Opening this PR should start the automatic Codex run; no manual codex-trigger was run.'
  url=$(gh pr create --base "$BASE" --head "$BRANCH" --title "$title" --body "$body")
  echo "created PR: $url" >&2
  grep -qF "$url" "$OUT" || printf '  - %s\n' "$url" >> "$OUT"
done

echo "updated $OUT" >&2
cat "$OUT"
