#!/usr/bin/env bash
# Check BUMP-GOJA-ROLLOUT PR readiness without manually triggering Codex.
#
# Normal path:
#   scripts/04-check-pr-readiness.sh
#
# Watch path:
#   scripts/04-check-pr-readiness.sh --watch --until actionable --interval-seconds 30 --timeout-seconds 1800
#
# Notes:
# - PR opening auto-triggers Codex in this setup.
# - Use ggg pr codex-trigger only later as recovery if readiness shows Codex is missing/stale/stuck.

set -euo pipefail

WORKSPACE=${WORKSPACE:-/home/manuel/workspaces/2026-06-06/bump-goja}
TICKET_DIR=${TICKET_DIR:-$WORKSPACE/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout}
PRS=${PRS:-$TICKET_DIR/scripts/prs.yaml}

exec ggg batch ready "$PRS" --output table "$@"
