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

