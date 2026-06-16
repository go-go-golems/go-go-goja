# Changelog

## 2026-06-16

- Investigated and mapped the go-go-goja auth examples (19 keycloak host, 21 generated-runtime seam) and the planned-route/auth-enforcement model; selected example 19 + 21 hybrid as the implementation to promote.
- Mapped the cross-repo release chain via `infra-tooling/docs/platform/source-repo-to-gitops-pr.md` and `2026-03-27--hetzner-k3s/docs/app-runtime-secrets-and-identity-provisioning-playbook.md`; identified `gitops/kustomize/go-go-host/` as the gold-standard copy-template.
- Mapped the Keycloak terraform pattern (`terraform/keycloak/apps/go-go-host/envs/k3s-beta/`) and the Vault/VSO secret model (runtime + image-pull + shared `infra/postgres/cluster`).
- Wrote the primary intern-facing design/implementation guide: `design/01-deploy-xgoja-keycloak-auth-host-to-yolo.md` (system map, current-state evidence, gap analysis, proposed architecture, decision records D1–D3, runtime flows, phased plan, out-of-repo specs §10–§12, test strategy, risks, references).
- Created investigation diary: `reference/01-investigation-diary.md`.
- Scope guardrail applied: only `./go-go-goja/` is modified; cluster/terraform/vault changes are specified verbatim and marked approval-gated.
- Related files: `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go` (host to promote), `examples/xgoja/21-generated-host-auth/xgoja.yaml` (generated seam), `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`, `pkg/xgoja/hostauth/config.go`, `deploy/gitops-targets.json`, `.github/workflows/publish-image.yaml`.

## 2026-06-16

Created intern-facing design/implementation guide and investigation diary for deploying an xgoja-generated Keycloak auth host to yolo.scapegoat.dev; mapped examples 19+21, the cross-repo release chain, the go-go-host reference package, and the Keycloak terraform pattern. Out-of-repo changes specified and approval-gated.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go — Host implementation to promote (commit: design only)


## 2026-06-16

Added research logbook (reference/02-...) cataloguing every resource read (in-repo examples/build/CI/source, sibling tickets, cluster repo, terraform, infra-tooling) with usefulness, staleness, and update needs; included a Part H README/navigation improvement plan.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/deploy/gitops-targets.json — Flagged stale goja-essay target referencing a non-existent GitOps package


## 2026-06-16

Added documentation improvement plan (design/02-...): maps the two-tree Glazed help system (pkg/doc 37 pages vs cmd/xgoja/doc 10 pages), refines the gap analysis (Go route API already documented in cmd/xgoja/doc/18; real gaps are host composition, hostauth, stores, serve, deployment, OIDC status, cross-tree navigation), and specifies 5 new pages + README/cross-link navigation fixes.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/doc.go — Tree 1 loader referenced by the doc plan


## 2026-06-16

Implemented source-repo auth-host increment: example 19 is now a Glazed serve command with public-base-url/redirect-url handling, auth-host Dockerfile/workflow/target were added, local smoke was updated for the demo Keycloak client, and example HTTP servers now handle SIGINT/SIGTERM with graceful Shutdown. Local Keycloak/Postgres smoke passes.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/.github/workflows/publish-auth-host-image.yaml — Auth-host image publishing workflow
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/Dockerfile.auth-host — Temporary auth-host image
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go — Glazed serve command


## 2026-06-16

Deployed goja-auth-host-demo to yolo: pushed GHCR image sha-ba77afc, added K3s GitOps/Vault/Postgres/Argo resources on task/clubmed-prod-gitops, provisioned a dedicated Keycloak realm/client/user, synced Argo to Healthy, issued TLS for goja-auth.yolo.scapegoat.dev, and passed the public Keycloak smoke test.

### Related Files

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/goja-auth-host-demo.yaml — Argo Application for the demo
- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-auth-host-demo/deployment.yaml — Live auth-host Deployment
- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/scripts/bootstrap-goja-auth-host-demo-runtime-secrets.sh — Vault runtime secret bootstrap helper


## 2026-06-16

Updated the documentation improvement plan with live production rollout lessons: Glazed auth-host config, public-base-url invariant, signal-aware shutdown, GHCR/Dockerfile ENTRYPOINT contract, K3s GitOps/Vault/Postgres/Keycloak/Argo workflow, live yolo failure modes, and new permanent doc/runbook targets.

### Related Files

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-auth-host-demo/deployment.yaml — Live deployment lessons feeding the doc plan
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-AUTH-DEPLOY--deploy-an-xgoja-generated-keycloak-auth-host-to-yolo-scapegoat-dev/design/02-documentation-improvement-plan-for-go-go-goja.md — Post-deployment documentation gap analysis and implementation plan


## 2026-06-16

Implemented permanent auth-host docs: README help-tree map, cross-links between JS and Go auth pages, xgoja host integration/config/stores/serve/production runbook pages, goja-repl Kubernetes deployment tutorial, and xgoja example annotations.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/README.md — Documentation map for both Glazed help trees
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/doc/23-auth-host-production-runbook.md — xgoja-side production runbook
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/32-deploying-an-express-auth-host.md — goja-repl deployment tutorial
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-AUTH-DEPLOY--deploy-an-xgoja-generated-keycloak-auth-host-to-yolo-scapegoat-dev/reference/01-investigation-diary.md — Diary Step 9 documentation implementation record

