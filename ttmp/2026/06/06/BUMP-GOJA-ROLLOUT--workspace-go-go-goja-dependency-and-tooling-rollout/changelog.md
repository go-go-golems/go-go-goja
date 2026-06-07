# Changelog

## 2026-06-06

- Initial workspace created


## 2026-06-06

Created rollout implementation guide, diary, and inventory script for Glazed linting, logcopter, and go-go-goja dependency/API updates.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/design-doc/01-implementation-guide.md — Detailed implementation guide
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Chronological diary
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/01-inventory-workspace.py — Repository inventory script
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/sources/01-workspace-inventory.md — Captured inventory evidence


## 2026-06-06

Expanded tasks.md into detailed Phase 0-8 rollout checklist with per-phase and repository-specific tasks.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Diary Step 2 records task expansion
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/tasks.md — Phased execution checklist


## 2026-06-06

Executed rollout for 9 repos (go-go-os-backend, vm-system, plz-confirm, go-go-host, pinocchio, workspace-manager, goja-git, goja-github-actions, go-minitrace). 3 repos (discord-bot, css-visual-diff, loupedeck) partially migrated but need xgoja provider API fixes. 7 repos not yet started.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Diary Step 3


## 2026-06-06

Committed 12 repos total. 4 repos have non-go-go-goja build errors. 3 repos need xgoja provider API migration. 1 repo blocked by transitive dependency.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Diary Step 4


## 2026-06-07

Completed xgoja provider API migrations for discord-bot, css-visual-diff, and loupedeck. All provider-level API changes applied.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Diary Step 6


## 2026-06-07

Validated migrated repos and repaired test-only go-go-goja v0.8 API drift in workspace-manager, goja-git, go-minitrace, discord-bot, loupedeck, go-go-gepa, and css-visual-diff provider tests.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Diary Step 7 validation and test cleanup


## 2026-06-07

Published plz-confirm v0.0.6, upgraded go-go-app-inventory to that tag, and validated smailnail with -tags sqlite_fts5 after updating stale runtime factory test calls.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Diary Step 8
- /home/manuel/workspaces/2026-06-06/bump-goja/plz-confirm/go.mod — Published v0.0.6 includes go-go-goja v0.8.3 dependency
- /home/manuel/workspaces/2026-06-06/bump-goja/smailnail/pkg/js/modules/smailnail/module_test.go — Runtime factory test-call migration for smailnail


## 2026-06-07

Fixed go-go-os-chat by owning the chat/webchat runtime packages and migrated go-go-app-inventory away from removed pinocchio package paths; both repos now pass GOWORK=off tests.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-app-inventory/pkg/pinoweb/hypercard_events.go — App inventory import migration to go-go-os-chat SEM/webchat packages
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Diary Step 9
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-os-chat/pkg/webchat/sem_translator.go — Geppetto v0.13 canonical event migration


## 2026-06-07

Final-validated go-go-os-chat, go-go-app-inventory, plz-confirm, and smailnail under GOWORK=off; cleaned abandoned local pinocchio compatibility branch/tag.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Diary Step 10 final validation


## 2026-06-07

Added GGG rollout operations playbook and research logbook evaluating infra-tooling resources for rollout, Codex, CI/CD, and release train workflows.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/design-doc/02-ggg-rollout-operations-playbook.md — New operator playbook
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/02-ggg-rollout-research-logbook.md — Research logbook and documentation freshness review


## 2026-06-07

Updated GGG rollout guidance: PR opening already triggers Codex, so codex-trigger is now documented as recovery-only; recorded the doc gap in the research logbook and diary.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/design-doc/02-ggg-rollout-operations-playbook.md — Corrected Codex trigger sequencing
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Recorded Step 11 documentation correction
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/02-ggg-rollout-research-logbook.md — Recorded upstream documentation clarification needed


## 2026-06-07

Persisted the rollout PR-opening script after discovering worktree .git handling and gh pr create --json portability issues.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Recorded Step 12 operational script persistence
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/02-open-rollout-prs.sh — Idempotent PR creation script


## 2026-06-07

Opened rollout PRs, generated scripts/prs.yaml, and recorded goja-text pre-push hook recovery; no manual Codex trigger was run because PR-open automation starts Codex.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/reference/01-diary.md — Recorded Step 13 PR opening
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/03-resume-open-remaining-prs.sh — Resume script after goja-text pre-push hook failure
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/prs.yaml — Generated rollout PR list


## 2026-06-07

Captured PR readiness and grouped Codex comment artifacts for the 21 rollout PRs; batch readiness currently blocked by Codex feedback and failed checks.

### Related Files

- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/04-check-pr-readiness.sh — Readiness check script
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/05-collect-codex-comments.sh — Codex comment collection script
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/codex-comments-2026-06-07T0926.md — Grouped Codex comments
- /home/manuel/workspaces/2026-06-06/bump-goja/go-go-goja/ttmp/2026/06/06/BUMP-GOJA-ROLLOUT--workspace-go-go-goja-dependency-and-tooling-rollout/scripts/readiness-2026-06-07T0925.json — Readiness snapshot

