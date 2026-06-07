#!/usr/bin/env bash
# Collect Codex comments for the BUMP-GOJA-ROLLOUT PR batch.
# This does not trigger Codex; it only reads comments for triage.

set -euo pipefail

WORKSPACE=${WORKSPACE:-/home/manuel/workspaces/2026-06-06/bump-goja}
TICKET_DIR=${TICKET_DIR:-$WORKSPACE/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout}
PRS=${PRS:-$TICKET_DIR/scripts/prs.yaml}
OUT=${OUT:-$TICKET_DIR/scripts/codex-comments-2026-06-07T0926.md}

# Prefer grouped markdown-ish text for human triage. If the installed ggg changes
# output behavior, stdout still contains the raw result.
ggg batch codex-comments "$PRS" --group-by-message | tee "$OUT"
