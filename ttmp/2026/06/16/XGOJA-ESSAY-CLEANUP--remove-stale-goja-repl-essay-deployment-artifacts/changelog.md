# Changelog

## 2026-06-16

- Initial workspace created


## 2026-06-16

Created cleanup ticket and concise implementation guide for removing stale goja-repl essay deployment artifacts before auth-host deployment; verified live GitOps no longer contains goja-essay package/application, while source repo still has deploy/gitops-targets.json target and essay-specific Dockerfile.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/Dockerfile — Essay-specific image behavior to decide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/deploy/gitops-targets.json — Stale target to remove


## 2026-06-16

Expanded cleanup scope per user clarification: delete the obsolete goja-repl essay app code (cmd/goja-repl/essay.go, pkg/replessay, essay web app, command registration, essay Docker default) while explicitly preserving pkg/replapi/pkg/repldb/pkg/replhttp/pkg/replsession and non-essay goja-repl commands.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/replapi — Explicitly preserved backend
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/replessay — Now in deletion scope


## 2026-06-16

Implemented essay app cleanup: removed goja-repl essay command registration and test, deleted cmd/goja-repl/essay.go, pkg/replessay, web/, root Dockerfile, essay publish workflow, and stale goja-essay GitOps target; preserved replapi/repldb/replhttp/replsession; go test ./... -count=1 passed.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/goja-repl/root.go — Removed essay command registration
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/deploy/gitops-targets.json — Now empty until auth-host deployment adds a new target
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/replapi — Preserved reusable backend


## 2026-06-16

Backfilled cleanup documentation before commit: documented deletion of the essay-specific publish-image workflow, added diary Step 4 with commit-boundary/staging notes, and confirmed auth-host publishing should be introduced separately.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/.github/workflows/publish-image.yaml — Essay-specific workflow removed; auth-host needs a new workflow later
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ESSAY-CLEANUP--remove-stale-goja-repl-essay-deployment-artifacts/reference/01-cleanup-diary.md — Backfilled commit-prep diary step


## 2026-06-18

Ticket closed

