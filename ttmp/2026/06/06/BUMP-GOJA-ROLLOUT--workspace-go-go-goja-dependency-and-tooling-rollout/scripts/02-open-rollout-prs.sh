#!/usr/bin/env bash
# Open BUMP-GOJA-ROLLOUT pull requests and write scripts/prs.yaml.
#
# Retrace notes:
# - Workspace repos are Git worktrees, so .git is often a file, not a directory.
#   Use [ -e "$repo/.git" ], not [ -d "$repo/.git" ].
# - This machine's gh pr create does not support --json. Create the PR, then read
#   the URL from stdout or gh pr list/view.
# - Opening the PR triggers Codex automatically in this setup. Do not run
#   ggg pr codex-trigger immediately after PR creation; use it only as recovery.
# - This script is intentionally idempotent: if an open PR already exists for
#   task/bump-goja, it records that URL instead of creating a duplicate.

set -euo pipefail

WORKSPACE=${WORKSPACE:-/home/manuel/workspaces/2026-06-06/bump-goja}
TICKET_DIR=${TICKET_DIR:-$WORKSPACE/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout}
OUT=${OUT:-$TICKET_DIR/scripts/prs.yaml}
BRANCH=${BRANCH:-task/bump-goja}
BASE=${BASE:-main}

repos=(
  plz-confirm
  go-go-os-chat
  go-go-app-inventory
  smailnail
  vm-system
  go-go-os-backend
  go-go-host
  pinocchio
  workspace-manager
  goja-git
  goja-github-actions
  go-minitrace
  js-analyzer
  goja-text
  scraper
  jesus
  go-go-gepa
  discord-bot
  css-visual-diff
  loupedeck
  go-go-goja
)

mkdir -p "$(dirname "$OUT")"
printf 'prs:\n' > "$OUT"

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
    printf '  - %s\n' "$existing" >> "$OUT"
    continue
  fi

  title="Bump go-go-goja to v0.8.3"
  if [[ "$repo" == "go-go-goja" ]]; then
    title="Docs: add bump-goja rollout playbook updates"
  fi

  body=$'Updates from the BUMP-GOJA-ROLLOUT workspace.\n\nHighlights:\n- bump/migrate go-go-goja usage to v0.8.x APIs where applicable\n- keep validation workspace-isolated with GOWORK=off\n- record rollout guidance in the ticket docs where applicable\n\nValidation was run locally during the rollout; see the ticket diary for repository-specific notes. Opening this PR should start the automatic Codex run; no manual codex-trigger was run.'

  # gh pr create prints the PR URL on success.
  url=$(gh pr create --base "$BASE" --head "$BRANCH" --title "$title" --body "$body")
  echo "created PR: $url" >&2
  printf '  - %s\n' "$url" >> "$OUT"
done

echo "wrote $OUT" >&2
cat "$OUT"
