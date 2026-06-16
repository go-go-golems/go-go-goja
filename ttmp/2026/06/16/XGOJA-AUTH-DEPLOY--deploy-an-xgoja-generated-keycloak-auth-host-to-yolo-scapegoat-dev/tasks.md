# Tasks

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked / approval-gated

## Investigation & design (this ticket)

- [x] Map go-go-goja auth examples (19 keycloak host, 21 generated seam) and the planned-route/auth model
- [x] Map the cross-repo release chain (infra-tooling source-repo-to-gitops-pr, cluster runtime-secrets playbook)
- [x] Map the gold-standard reference package (gitops/kustomize/go-go-host)
- [x] Map the Keycloak terraform pattern (terraform/keycloak/apps/go-go-host/envs/k3s-beta)
- [x] Write intern-facing design/implementation guide (design/01-...)
- [x] Keep investigation diary (reference/01-...)

## Phase 1 — Promote the host (IN-REPO, go-go-goja)

- [ ] Create `cmd/goja-auth-host/main.go` promoted from example 19 with prod defaults (bind :8080, secure cookies, graceful shutdown, /readyz, config-from-env, derived RedirectURL, drop demo seed)
- [ ] Add `cmd/goja-auth-host/config.go` (Glazed/YAML config mirroring go-go-host config.yaml)
- [ ] `go build ./cmd/goja-auth-host && go test ./...`

## Phase 2 — Generated route seam (IN-REPO, go-go-goja)

- [ ] Add `cmd/goja-auth-host/xgoja.yaml` (copy example 21) emitting `internal/xgojaruntime` from a `routes/` JS source
- [ ] Regenerate via `xgoja generate` / `go generate`
- [ ] Mount generated express module onto the Host in main.go (replace os.ReadFile path)
- [ ] Regression: `make -C examples/xgoja/21-generated-host-auth smoke` still passes

## Phase 3 — Image + CI (IN-REPO, go-go-goja)

- [ ] Add `Dockerfile.auth-host` (no frontend; CGO_ENABLED=1 for sqlite fallback)
- [ ] Extend `.github/workflows/publish-image.yaml` with auth-host job (build, tag sha-, push GHCR, open GitOps PR)
- [ ] Append `goja-auth-host-prod` target to `deploy/gitops-targets.json`
- [ ] Validate: `python3 .../infra-tooling/scripts/gitops/validate_gitops_targets.py deploy/gitops-targets.json`

## Phase 4 — Smoke + docs (IN-REPO, go-go-goja)

- [ ] Add `auth-host-smoke` Makefile target (build image, curl /healthz /readyz, expect 401 on /me)
- [ ] Link repo README / examples/xgoja/README.md to the production guide

## Phase 5–7 — Out-of-repo (APPROVAL-GATED, operator)

- [!] Cluster repo: new `gitops/kustomize/goja-auth-host/` (copy go-go-host) + `gitops/applications/goja-auth-host.yaml` + `vault/policies,roles/` — design doc §10
- [!] Terraform repo: `keycloak/apps/goja-auth-host/envs/k3s-beta/` via `make scaffold-browser-app` — design doc §11
- [!] Vault: seed `apps/goja-auth-host/beta/{runtime,image-pull}` + bootstrap k8s auth — design doc §12
- [!] First rollout: `kubectl apply -f gitops/applications/goja-auth-host.yaml` + runtime validation

## Delivery

- [ ] `docmgr doctor --ticket XGOJA-AUTH-DEPLOY --stale-after 30` passes
- [ ] Upload design + diary bundle to reMarkable (dry-run then real)
