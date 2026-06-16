---
Title: Goja essay cleanup implementation guide
Ticket: XGOJA-ESSAY-CLEANUP
Status: active
Topics:
    - goja
    - deployment
    - gitops
    - documentation
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/2026-03-27--hetzner-k3s/gitops
      Note: Live GitOps tree checked; goja-essay package/application missing
    - Path: Dockerfile
      Note: Root image builds and runs only goja-repl essay; delete or replace so the repo no longer has an essay-default production image.
    - Path: .github/workflows/publish-image.yaml
      Note: Essay-specific image publishing workflow to delete; auth-host deployment should add a fresh workflow later.
    - Path: cmd/goja-repl/essay.go
      Note: |-
        Essay subcommand entrypoint to delete; preserve other goja-repl commands that use replapi.
        Essay command entrypoint to delete
    - Path: cmd/goja-repl/root.go
      Note: Essay command registration to remove while preserving non-essay goja-repl commands
    - Path: deploy/gitops-targets.json
      Note: |-
        Source-repo deploy target still points at deleted GitOps package gitops/kustomize/goja-essay/deployment.yaml.
        Stale source-repo deploy target points at removed gitops/kustomize/goja-essay/deployment.yaml
    - Path: pkg/replapi
      Note: |-
        Explicitly preserved reusable REPL API/session kernel used by non-essay commands and TUI.
        Reusable REPL API/session kernel to preserve
    - Path: pkg/replessay
      Note: Essay HTTP app package to delete
    - Path: pkg/replessay/handler.go
      Note: Essay HTTP handler implementation to delete; do not delete pkg/replapi or other reusable REPL backends.
    - Path: web
      Note: |-
        Essay React app to delete if no non-essay product surface depends on it.
        Essay React app to delete if review confirms no non-essay use
ExternalSources: []
Summary: Remove the obsolete goja-repl essay app code and stale deployment references before adding the new auth-host deployment target, while preserving the core replapi/repldb/replhttp session backend.
LastUpdated: 2026-06-16T18:05:00-04:00
WhatFor: Use before implementing the auth-host deployment so the source repo no longer publishes GitOps updates to a removed goja-essay package.
WhenToUse: When cleaning up deploy/gitops-targets.json, root Dockerfile behavior, or any goja-repl essay deployment leftovers.
---



# Goja essay cleanup implementation guide

## Executive summary

`goja-repl essay` appears to have outlived its usefulness as a deployed app on
`yolo.scapegoat.dev`. The live cluster GitOps repo no longer contains
`gitops/kustomize/goja-essay/` or `gitops/applications/goja-essay.yaml`, but the
`go-go-goja` source repo still contains a deploy target named
`goja-essay-prod` that points at that deleted package.

Before adding the new auth-host deployment target, clean up the stale essay
**deployment artifacts** so image-publishing automation does not try to patch a
non-existent manifest. The requested scope is now explicit: delete the essay **app** as well as its stale deployment plumbing. Preserve the core REPL API/session backend (`pkg/replapi`, `pkg/repldb`, `pkg/replhttp`, `pkg/replsession`) and the non-essay `goja-repl` commands that use it. In other words, remove the teaching/demo web application, not the reusable REPL API kernel.

## Current evidence

### In `go-go-goja`

`deploy/gitops-targets.json` still contains the old production target:

```json
{
  "name": "goja-essay-prod",
  "gitops_repo": "wesen/2026-03-27--hetzner-k3s",
  "gitops_branch": "main",
  "manifest_path": "gitops/kustomize/goja-essay/deployment.yaml",
  "container_name": "goja-essay"
}
```

The root `Dockerfile` is also essay-specific:

```dockerfile
ENV GOJA_REPL_ESSAY_WEB_DIST=/app/web/dist/public
ENTRYPOINT ["/app/goja-repl"]
CMD ["essay", "--addr", ":8080", "--db-path", "/data/goja-repl.sqlite"]
```

The essay feature itself still exists in source:

- `cmd/goja-repl/essay.go` registers the `essay` subcommand.
- `pkg/replessay/handler.go` serves the essay page and APIs.
- `web/` builds the essay React UI.

### In the cluster GitOps repo

Verified by inspection of `/home/manuel/code/wesen/2026-03-27--hetzner-k3s`:

```bash
rg -n "goja-essay|goja.yolo.scapegoat.dev" gitops docs -S || true
# no live gitops/docs hits

test -e gitops/kustomize/goja-essay && echo exists || echo missing
# missing

test -e gitops/applications/goja-essay.yaml && echo exists || echo missing
# missing
```

Historical ticket/docs references remain under `ttmp/`, including evidence that
`goja-essay` was removed during the 2026-06-06 Argo CD cleanup. Those historical
references should remain untouched.

## Scope decision

### Cleanup scope for this ticket

1. Remove stale source-repo deployment target(s):
   - `deploy/gitops-targets.json` should no longer contain `goja-essay-prod`.
2. Delete the essay app code:
   - remove `cmd/goja-repl/essay.go`;
   - remove the `newEssayCommand(out, opts)` registration from `cmd/goja-repl/root.go`;
   - remove essay-specific command tests from `cmd/goja-repl/root_test.go`;
   - remove `pkg/replessay/`;
   - remove the essay React app under `web/` if it is not used by another product surface;
   - remove or rewrite the root `Dockerfile` because it currently builds and runs only the essay app;
   - remove `.github/workflows/publish-image.yaml` because it builds `web/`, uses the essay Dockerfile, smoke-tests `/api/essay/...`, and opens GitOps PRs for the stale target.
3. Preserve the reusable REPL/session backend:
   - keep `pkg/replapi/`;
   - keep `pkg/repldb/`;
   - keep `pkg/replhttp/`;
   - keep `pkg/replsession/`;
   - keep non-essay `cmd/goja-repl` API/session/snapshot/TUI commands unless a separate cleanup says otherwise.
4. Validate that source builds/tests still pass.
5. Do **not** touch the cluster repo unless a verification command proves live
   `gitops/` references were reintroduced.

### Explicitly out of scope unless requested

- Deleting `pkg/replapi`, `pkg/repldb`, `pkg/replhttp`, or `pkg/replsession`.
- Deleting the `goja-repl` CLI itself.
- Rewriting historical `ttmp/` docs in either repo.
- Removing GHCR image tags or registry packages.

## Recommended implementation plan

### Phase 1 — remove stale deploy target

Edit `deploy/gitops-targets.json` so it no longer references `goja-essay-prod`.
Because the file currently contains only that one target, the minimal valid
result is:

```json
{
  "targets": []
}
```

If the auth-host target is implemented in the same branch, prefer a two-step
history:

1. commit removal of `goja-essay-prod`, then
2. commit addition of `goja-auth-host-prod`.

That makes review obvious: old target removed first, new target added second.

### Phase 2 — delete the essay app code while preserving replapi

Remove the essay-specific integration points first, then the implementation directories:

1. In `cmd/goja-repl/root.go`, remove `newEssayCommand(out, opts)` from the root command registration list.
2. Delete `cmd/goja-repl/essay.go`.
3. Update `cmd/goja-repl/root_test.go` by deleting the essay-help test(s).
4. Delete `pkg/replessay/` and its tests.
5. Delete `web/` if no non-essay code imports it. Current evidence indicates it is the essay React app (`EssayApp`, `essayApi`, `/api/essay/...`, `/static/essay/`).
6. Delete the root `Dockerfile` if it existed only to package the essay, or replace it with a neutral/auth-host-specific Dockerfile in the auth deploy work. Do not leave an essay-default Dockerfile in place.
7. Delete `.github/workflows/publish-image.yaml` because it is not generic: it installs/builds `web/`, builds `./Dockerfile`, smoke-tests essay endpoints, and uses the stale GitOps target list.

Preservation rule: any code imported by non-essay commands stays. The grep boundary is:

```bash
rg -n "replapi|repldb|replhttp|replsession" cmd pkg -S
```

The expected preserved surface includes `cmd/goja-repl/cmd_*.go`, `cmd/goja-repl/tui.go`, `pkg/replapi/`, `pkg/repldb/`, `pkg/replhttp/`, and `pkg/replsession/`.

### Phase 3 — verify cluster state before auth deploy

From the cluster repo:

```bash
cd /home/manuel/code/wesen/2026-03-27--hetzner-k3s
rg -n "goja-essay|goja.yolo.scapegoat.dev" gitops docs -S || true
test ! -e gitops/kustomize/goja-essay
test ! -e gitops/applications/goja-essay.yaml
```

If kubectl access is available, also verify live state:

```bash
kubectl -n argocd get application goja-essay
kubectl get ns goja-essay
```

Expected result: both commands should report `NotFound`. If either object
exists, stop and clean up the live Argo Application/namespace before adding new
auth-host resources.

### Phase 4 — validate source repo

Run:

```bash
go test ./... -count=1
```

If Dockerfile changes are made, also run the relevant build command:

```bash
docker build -t go-go-goja:cleanup-check .
```

Only add the new auth-host deployment target after the stale essay target is
gone and validation passes.

## Pseudocode checklist

```text
function cleanupGojaEssayDeployment():
    assert cluster.gitopsPath("gitops/kustomize/goja-essay") is missing
    assert cluster.gitopsFile("gitops/applications/goja-essay.yaml") is missing

    targets = readJSON("deploy/gitops-targets.json")
    targets = targets.removeWhere(name == "goja-essay-prod")
    writeJSON("deploy/gitops-targets.json", targets)

    removeRootCommandRegistration("newEssayCommand")
    delete("cmd/goja-repl/essay.go")
    delete("pkg/replessay/")
    delete("web/")
    deleteEssayTests()

    if rootDockerfileRunsEssay():
        delete("Dockerfile") or replaceWithAuthHostDockerfileLater()

    run("go test ./... -count=1")
    commit("Remove stale goja essay deployment target")
```

## Risks and review notes

- **Do not delete replapi.** The app is obsolete, but the reusable REPL API/session backend is still used by non-essay commands and the TUI. Preserve `pkg/replapi`, `pkg/repldb`, `pkg/replhttp`, and `pkg/replsession`.
- **Empty target list behavior.** Confirm the GitOps PR script handles
  `{"targets": []}` gracefully. If it does not, update the script or add a
  no-op guard before merging.
- **Root Dockerfile ambiguity.** The root Dockerfile currently encodes the old
  essay workload. Avoid reusing it for auth-host deployment unless the command,
  probes, and runtime data assumptions are changed deliberately.

## Acceptance criteria

- `deploy/gitops-targets.json` no longer references `goja-essay-prod`,
  `goja-essay`, or `gitops/kustomize/goja-essay/deployment.yaml`.
- Live GitOps repo inspection still shows no `goja-essay` package/application.
- The essay-specific image publish workflow is gone; auth-host publishing will be introduced separately.
- `cmd/goja-repl essay` is no longer registered or documented as a command.
- `pkg/replessay/` and essay-specific `web/` code are gone.
- `pkg/replapi/`, `pkg/repldb/`, `pkg/replhttp/`, and `pkg/replsession/` remain and their tests pass.
- The root Dockerfile no longer runs the essay app by default (deleted, replaced, or deferred to an auth-host-specific Dockerfile).
- `go test ./... -count=1` passes.
- This ticket's diary and changelog record the expanded cleanup decision.
