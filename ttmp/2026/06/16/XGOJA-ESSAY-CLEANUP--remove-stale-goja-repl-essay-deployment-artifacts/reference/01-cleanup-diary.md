---
Title: Cleanup diary
Ticket: XGOJA-ESSAY-CLEANUP
Status: active
Topics:
    - goja
    - deployment
    - gitops
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: Dockerfile
      Note: |-
        Root image still defaults to running the essay subcommand.
        Requires decision before auth-host deployment
    - Path: deploy/gitops-targets.json
      Note: |-
        Stale source-repo deploy target that still points at removed goja-essay GitOps package.
        Primary cleanup target
ExternalSources: []
Summary: Chronological diary for the goja-repl essay deployment cleanup ticket.
LastUpdated: 2026-06-16T18:10:00-04:00
WhatFor: Continue the stale goja-essay deployment cleanup without redoing the initial investigation.
WhenToUse: Before editing deploy/gitops-targets.json or deciding the root Dockerfile fate.
---


# Diary

## Goal

Capture the initial evidence and cleanup plan for removing stale `goja-repl essay` deployment artifacts before adding the new auth-host deployment.

## Step 1: Create cleanup ticket and verify stale deployment references

The initial pass confirmed the user's suspicion: the live cluster GitOps tree no longer contains the `goja-essay` Kustomize package or Argo Application, while the `go-go-goja` source repo still has a deploy target that points at the deleted manifest path. I created a dedicated ticket and wrote a concise implementation guide so the cleanup can happen before auth-host deployment work starts.

This step intentionally scoped the cleanup to deployment artifacts first. The essay source code (`cmd/goja-repl/essay.go`, `pkg/replessay/`, and `web/`) still exists and may remain useful locally or historically; deleting that code should be a separate explicit product decision.

### Prompt Context

**User prompt (verbatim):** "I think before we are doing the auth deploy, we should remove the go-go-goja essay which has outlived its usefulness and has been removed from yolo I think anyway. Create a ticket to do this cleanup, and a concise implementation / analysis guide"

**Assistant interpretation:** Create a focused docmgr ticket and concise guide for cleaning up stale goja essay deployment artifacts before implementing the auth deployment.

**Inferred user intent:** Avoid carrying stale `goja-essay` deployment automation into the upcoming auth-host deployment and document the cleanup so implementation is straightforward.

**Commit (code):** N/A — documentation/ticket setup only.

### What I did
- Ran `docmgr status --summary-only` in `go-go-goja`.
- Searched in-repo references with `rg -n "goja-essay|essay" . -S --glob '!ttmp/**' ...`.
- Confirmed `deploy/gitops-targets.json` still contains `goja-essay-prod` pointing at `gitops/kustomize/goja-essay/deployment.yaml`.
- Confirmed the root `Dockerfile` still runs `goja-repl essay` by default.
- Verified live cluster GitOps paths are gone with:
  - `rg -n "goja-essay|goja.yolo.scapegoat.dev" gitops docs -S || true`
  - `test -e gitops/kustomize/goja-essay ...`
  - `test -e gitops/applications/goja-essay.yaml ...`
- Created ticket `XGOJA-ESSAY-CLEANUP`.
- Wrote `design/01-goja-essay-cleanup-implementation-guide.md`.

### Why
- The stale source-repo deployment target would cause deploy automation to patch a GitOps file that no longer exists.
- The auth-host deployment should start from a clean deploy target list instead of replacing an obsolete demo app implicitly.

### What worked
- The cluster repo verification confirmed no live `gitops/` `goja-essay` package/application remains.
- The source repo evidence is narrow and actionable: `deploy/gitops-targets.json` and the root `Dockerfile` are the deployment artifacts to review first.

### What didn't work
- N/A. No failing commands during this setup step.

### What I learned
- `goja-essay` was removed from live GitOps, but historical `ttmp/` references remain and should not be rewritten.
- The immediate cleanup is deployment plumbing, not wholesale source deletion.

### What was tricky to build
- The word "remove" is ambiguous here: it can mean removing the deployed app, removing the source deploy target, or deleting the essay source feature. I resolved this by documenting the default scope as deployment-artifact cleanup only and marking source deletion as explicitly out of scope unless requested.

### What warrants a second pair of eyes
- Whether the root `Dockerfile` should be kept as a legacy/local essay image, removed, or left untouched until `Dockerfile.auth-host` exists.
- Whether `deploy/gitops-targets.json` with an empty `targets` array is accepted by the publish/GitOps PR scripts.

### What should be done in the future
- Implement the cleanup in a focused code commit.
- After cleanup, add the auth-host deployment target in a separate commit.

### Code review instructions
- Start with `deploy/gitops-targets.json` and `Dockerfile`.
- Validate cluster assumptions from `/home/manuel/code/wesen/2026-03-27--hetzner-k3s` before modifying out-of-repo files.
- Run `go test ./... -count=1` after implementation.

### Technical details
- Ticket: `XGOJA-ESSAY-CLEANUP`
- Guide: `design/01-goja-essay-cleanup-implementation-guide.md`
- Expected minimal deploy target content after cleanup:
  ```json
  { "targets": [] }
  ```


## Step 2: Expand scope to delete the essay app but preserve replapi

The user clarified that cleanup should delete the obsolete essay app code as well, not only deployment plumbing. I updated the guide and task list so implementation removes the `goja-repl essay` subcommand, `pkg/replessay`, and the essay React app while explicitly preserving the reusable REPL API/session backend.

The key boundary is that `pkg/replapi` is not the essay app. It is used by non-essay `goja-repl` commands and the TUI (`cmd_eval`, `cmd_sessions`, `cmd_snapshot`, `tui`, etc.), so the cleanup must remove the teaching/demo surface without damaging the reusable session kernel.

### Prompt Context

**User prompt (verbatim):** "we want to delete the code as well (the app, obv, not the replapi). update."

**Assistant interpretation:** Expand the cleanup plan to delete the essay app implementation, while explicitly preserving `replapi` and related reusable REPL/session infrastructure.

**Inferred user intent:** Fully retire the obsolete deployed essay application before auth deploy work, avoiding leftover app code and Docker defaults, but keep core REPL APIs used elsewhere.

**Commit (code):** N/A — ticket documentation update only.

### What I did
- Searched source boundaries with `rg -n "pkg/replapi|replapi|pkg/replessay|replessay|newEssayCommand|EssayApp|essayApi|GOJA_REPL_ESSAY_WEB_DIST" cmd pkg web Makefile .github Dockerfile deploy -S`.
- Updated the implementation guide to list deletion targets:
  - `cmd/goja-repl/essay.go`
  - `newEssayCommand(out, opts)` registration in `cmd/goja-repl/root.go`
  - essay-specific test(s) in `cmd/goja-repl/root_test.go`
  - `pkg/replessay/`
  - essay React app under `web/`
  - root `Dockerfile` essay default
- Updated the implementation guide to list preservation targets:
  - `pkg/replapi/`
  - `pkg/repldb/`
  - `pkg/replhttp/`
  - `pkg/replsession/`
  - non-essay `cmd/goja-repl` commands and TUI.
- Rewrote `tasks.md` to match the expanded scope.

### Why
- Deployment-only cleanup would leave an obsolete app surface in the repo.
- Preserving `replapi` avoids breaking reusable REPL/session functionality that is not specific to the essay.

### What worked
- Grep showed a clean boundary: `cmd/goja-repl/essay.go` imports `pkg/replessay`, while many non-essay commands import `pkg/replapi` directly.
- The guide now explicitly distinguishes app deletion from backend preservation.

### What didn't work
- The initial guide had stale frontmatter notes saying source deletion was out of scope; those notes had to be corrected after the scope change.

### What I learned
- The essay app is composed of command glue (`essay.go`), HTTP wrapper (`pkg/replessay`), and a dedicated React app (`web/`).
- `replapi` is broader infrastructure and should not be treated as part of the obsolete app.

### What was tricky to build
- The tricky part is avoiding name-based deletion. Many files mention "essay" in the UI, but many files mention `replapi` for valid non-essay flows. The implementation must delete from the app boundary outward and then let compiler/test failures identify any missed essay references.

### What warrants a second pair of eyes
- Whether `web/` is exclusively the essay app or has become a shared frontend package. Current evidence points to essay-only, but review before deletion.
- Whether the root `Dockerfile` should be deleted now or replaced later by `Dockerfile.auth-host`.

### What should be done in the future
- Implement the deletion in a focused commit before starting auth-host deployment.

### Code review instructions
- Start at `cmd/goja-repl/root.go` and `cmd/goja-repl/essay.go` to remove the command seam.
- Then delete `pkg/replessay/` and `web/` and run `go test ./... -count=1`.
- Confirm `rg -n "replessay|newEssayCommand|GOJA_REPL_ESSAY_WEB_DIST|/api/essay|/static/essay" . -S --glob '!ttmp/**'` has no remaining live-source hits.

### Technical details
- Preserve grep boundary:
  ```bash
  rg -n "replapi|repldb|replhttp|replsession" cmd pkg -S
  ```
- Deletion grep after implementation:
  ```bash
  rg -n "replessay|newEssayCommand|GOJA_REPL_ESSAY_WEB_DIST|/api/essay|/static/essay" . -S --glob '!ttmp/**'
  ```


## Step 3: Implement essay app deletion and validate preserved REPL backend

I removed the obsolete essay application end-to-end: the `goja-repl essay` command seam, the `pkg/replessay` HTTP wrapper, the essay React `web/` app, the essay Dockerfile, the essay GitHub image-publish workflow, and the stale GitOps target. The reusable REPL API/session backend remains in place and tests still pass.

The important implementation choice was to delete the essay-specific publish workflow instead of trying to make it generic. That workflow was essay-shaped throughout: it installed `web/`, built the essay frontend, built `./Dockerfile`, smoke-tested `/api/essay/...`, and opened GitOps PRs against the stale essay target. The auth-host deployment should introduce a new workflow/image contract deliberately rather than inheriting this old one.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Perform the cleanup described by the updated ticket: delete the essay app code and deployment plumbing while preserving replapi.

**Inferred user intent:** Clear obsolete essay app code before proceeding with auth deployment work.

**Commit (code):** N/A — changes are currently uncommitted.

### What I did
- Removed `newEssayCommand(out, opts)` from `cmd/goja-repl/root.go`.
- Removed the essay help test from `cmd/goja-repl/root_test.go`.
- Deleted `cmd/goja-repl/essay.go`.
- Deleted `pkg/replessay/`.
- Deleted the essay React app under `web/`.
- Deleted root `Dockerfile` because it only built/ran `goja-repl essay`.
- Deleted `.github/workflows/publish-image.yaml` because it was essay-specific end-to-end.
- Replaced `deploy/gitops-targets.json` with an empty target list:
  ```json
  {
    "targets": []
  }
  ```
- Ran `gofmt -w cmd/goja-repl/root.go cmd/goja-repl/root_test.go`.

### Why
- The app has been retired from yolo and the GitOps package was already removed.
- Leaving the workflow/Dockerfile/target in place would make later auth-host deployment inherit obsolete essay assumptions.

### What worked
- Cleanup grep outside `ttmp/` returned no live references for:
  `replessay`, `newEssayCommand`, `GOJA_REPL_ESSAY_WEB_DIST`, `/api/essay`, `/static/essay`, `goja-essay`, `goja-repl essay`, `interactive REPL essay`, `web/pnpm`, and `Dockerfile`.
- Preserved backend directories still exist: `pkg/replapi`, `pkg/repldb`, `pkg/replhttp`, `pkg/replsession`.
- `go run ./cmd/goja-repl --help | grep -i essay || true` produced no essay command output.
- `go test ./... -count=1` passed.

### What didn't work
- N/A. The full test suite passed after cleanup.

### What I learned
- The image publish workflow was not reusable deployment plumbing; it was coupled to the retired essay frontend, Dockerfile, and smoke URLs.
- The safest path is to remove it now and let the auth-host ticket add a new image workflow with the correct command/probes/target.

### What was tricky to build
- The tricky part was recognizing that deleting `Dockerfile` also required deleting the workflow that built it. If only the Dockerfile had been deleted, CI would fail. If only the stale GitOps target had been removed, CI would still publish an obsolete essay image. The cleanup therefore had to remove all three together: Dockerfile, workflow, and target.

### What warrants a second pair of eyes
- Confirm that deleting the root `Dockerfile` is acceptable until the auth-host Dockerfile is added.
- Confirm that no external automation expects `.github/workflows/publish-image.yaml` to exist before the auth-host deployment work reintroduces image publishing.

### What should be done in the future
- In the auth-host ticket, create a fresh auth-host-specific Dockerfile and image publish workflow rather than resurrecting the essay workflow.

### Code review instructions
- Review `cmd/goja-repl/root.go` first to confirm only the essay command was removed.
- Review deleted directories/files: `cmd/goja-repl/essay.go`, `pkg/replessay/`, `web/`, `Dockerfile`, `.github/workflows/publish-image.yaml`.
- Review `deploy/gitops-targets.json` to confirm it is intentionally empty until auth-host deployment adds a new target.
- Validate with:
  ```bash
  rg -n "replessay|newEssayCommand|GOJA_REPL_ESSAY_WEB_DIST|/api/essay|/static/essay|goja-essay|goja-repl essay|interactive REPL essay|web/pnpm|Dockerfile" . -S --glob '!ttmp/**'
  go test ./... -count=1
  ```

### Technical details
- Test command run:
  ```bash
  go test ./... -count=1
  ```
- Result: passed.


## Step 4: Backfill commit-prep diary, document workflow removal, and stage only cleanup files

Before committing, I revisited the documentation to make sure the ticket reflects the actual implementation, not only the original plan. The main backfill was documenting that `.github/workflows/publish-image.yaml` was deleted because it was coupled to the essay app end-to-end: Node/pnpm setup for `web/`, the root essay Dockerfile, essay smoke URLs, and the stale GitOps target list.

I also checked the working tree carefully because there are other untracked 2026-06-16 ticket documents from the auth-deploy work. The cleanup commit should include the essay cleanup code and the `XGOJA-ESSAY-CLEANUP` ticket only, not the unrelated auth-deploy ticket docs or the pre-existing `ttmp/vocabulary.yaml` modification.

### Prompt Context

**User prompt (verbatim):** "commit and keep a detailed diary of the work you did, backfilling whatever steps you think are good to document (in case we haven't already)"

**Assistant interpretation:** Commit the implemented cleanup and ensure the diary contains enough chronological detail for review/continuation, including any missing implementation details.

**Inferred user intent:** Produce a clean, reviewable commit with durable ticket documentation explaining what changed, why, and how it was validated.

**Commit (code):** Pending at time of writing this diary entry.

### What I did
- Inspected `git status --short` and found two untracked 2026-06-16 ticket trees; only this cleanup ticket should be staged for the cleanup commit.
- Updated the implementation guide to explicitly mention `.github/workflows/publish-image.yaml` as a deletion target.
- Updated `tasks.md` to add/complete the publish-workflow deletion task.
- Planned selective staging for:
  - cleanup code/deletions;
  - `deploy/gitops-targets.json`;
  - `ttmp/.../XGOJA-ESSAY-CLEANUP...`.
- Planned not to stage:
  - `ttmp/2026/06/16/XGOJA-AUTH-DEPLOY...`;
  - `ttmp/vocabulary.yaml` (pre-existing unrelated ticket vocabulary change).

### Why
- The workflow deletion is review-critical because removing `Dockerfile` alone would break CI if the workflow remained.
- Selective staging prevents unrelated doc/ticket work from being bundled into the cleanup commit.

### What worked
- The cleanup diary already had Steps 1–3 covering investigation, scope expansion, and implementation/validation.
- The final backfill clarified the workflow deletion and commit hygiene.

### What didn't work
- N/A.

### What I learned
- Large deletion commits need explicit staging discipline when ticket docs from adjacent work are untracked in the same date directory.

### What was tricky to build
- The tricky part was not the code edit; it was preserving commit boundaries. `ttmp/2026/06/16/` contains both the auth-deploy ticket and this cleanup ticket, so staging the whole date directory would accidentally commit unrelated deliverables.

### What warrants a second pair of eyes
- Confirm the cleanup commit intentionally leaves no image-publish workflow until the auth-host deployment introduces a replacement.
- Confirm `ttmp/vocabulary.yaml` is intentionally left unstaged as unrelated/pre-existing work.

### What should be done in the future
- Add a fresh auth-host-specific Dockerfile/workflow in the auth-deploy ticket.

### Code review instructions
- Review the commit as a deletion/retirement commit: command seam first, app package/UI deletion second, CI/deploy plumbing third, ticket docs last.
- Re-run:
  ```bash
  rg -n "replessay|newEssayCommand|GOJA_REPL_ESSAY_WEB_DIST|/api/essay|/static/essay|goja-essay|goja-repl essay|interactive REPL essay|web/pnpm|Dockerfile" . -S --glob '!ttmp/**'
  go test ./... -count=1
  docmgr doctor --ticket XGOJA-ESSAY-CLEANUP --stale-after 30
  ```

### Technical details
- Selective staging command pattern:
  ```bash
  git add -A -- .github/workflows/publish-image.yaml Dockerfile cmd/goja-repl/essay.go cmd/goja-repl/root.go cmd/goja-repl/root_test.go deploy/gitops-targets.json pkg/replessay web
  git add ttmp/2026/06/16/XGOJA-ESSAY-CLEANUP--remove-stale-goja-repl-essay-deployment-artifacts
  ```
