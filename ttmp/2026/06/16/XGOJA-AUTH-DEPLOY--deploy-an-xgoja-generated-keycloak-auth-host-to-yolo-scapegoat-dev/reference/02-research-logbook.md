---
Title: Research logbook
Ticket: XGOJA-AUTH-DEPLOY
Status: active
Topics:
    - goja
    - keycloak
    - oidc
    - deployment
    - kubernetes
    - gitops
    - vault
    - security
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/terraform/docs/shared-keycloak-platform-playbook.md
      Note: Most stale doc read — smailnail-centred repo map/realm claim needs refresh
    - Path: deploy/gitops-targets.json
      Note: Most important correctness finding — goja-essay target points at a non-existent package
    - Path: design/01-deploy-xgoja-keycloak-auth-host-to-yolo.md
      Note: Primary design doc this logbook supports
    - Path: reference/01-investigation-diary.md
      Note: Chronological diary; this logbook is the per-resource companion
ExternalSources:
    - /home/manuel/code/wesen/go-go-golems/infra-tooling/
    - /home/manuel/code/wesen/2026-03-27--hetzner-k3s/
    - /home/manuel/code/wesen/terraform/
Summary: 'Per-resource research logbook tracking every document and external resource read while producing the XGOJA-AUTH-DEPLOY design: why each was chosen, what was useful, what is out of date/wrong, and what needs updating.'
LastUpdated: 2026-06-16T10:00:00-04:00
WhatFor: Use to decide which docs to trust, which are stale, and which READMEs to improve so future engineers don't have to dig by hand.
WhenToUse: Before re-running this investigation, or when prioritising doc-cleanup work across the four repos.
---


# Research logbook

## Purpose

This is the per-resource companion to the investigation diary. Where the diary
is chronological ("what I did in what order"), this logbook is indexed by
**resource** and records, for every document and external tree I read:

1. what I was researching at the time,
2. what I was looking for in *that* document specifically,
3. why I chose it,
4. how I found the resource itself,
5. what I found useful,
6. what I did *not* find useful,
7. what is out of date or wrong,
8. what would need updating.

It is organised by source repository, then by resource. A summary inventory
table comes first so you can scan, then the detailed entries. The final
section (Part H) is a concrete set of README/navigation improvements.

## How to use this logbook

- Treat each entry's **Out of date / wrong** and **Needs updating** fields as
  actionable doc-cleanup tickets.
- The **How I found it** field is the most important signal for the README
  work in Part H: if a critical resource was only reachable by `find`/`grep`
  rather than by a README link, that is a navigation gap to fix.
- "Confidence" is my assessment of how much the doc can be trusted as-is for
  this deployment task: High / Medium / Low.

## Resource inventory (summary)

| ID | Resource | Repo | Confidence | Key issue |
| --- | --- | --- | --- | --- |
| A1 | `examples/xgoja/README.md` (learning path) | go-go-goja | High | Doesn't flag which examples are production templates |
| A2 | example 19 (keycloak-auth-host) README+main.go+server.js | go-go-goja | High | Dev-only cookie/probe gaps (by design) |
| A3 | example 21 (generated-host-auth) README+xgoja.yaml+main.go | go-go-goja | High | Explicitly incomplete: OIDC/Postgres = follow-up |
| A4 | `AGENT.md` | go-go-goja | High | No deployment/build context |
| B1 | `Dockerfile` + `.dockerignore` | go-go-goja | High | Builds only essay; no auth-host target |
| B2 | `deploy/gitops-targets.json` | go-go-goja | **Low** | **Points at a non-existent GitOps package** |
| B3 | `.github/workflows/publish-image.yaml` + `open_gitops_pr.py` | go-go-goja | High | Uses old repo-local GitOps helper, not shared workflow |
| B4 | `Makefile` + `.goreleaser.yaml` | go-go-goja | High | Release config is for essay binary only |
| C1 | `pkg/gojahttp/auth/keycloakauth/*` | go-go-goja | High | Accurate; Config is the wiring surface |
| C2 | `pkg/xgoja/hostauth/{config.go,doc.go}` | go-go-goja | Medium | `ModeOIDC` unfinished (see A3) |
| C3 | `pkg/gojahttp/host.go` | go-go-goja | High | Accurate; HostOptions.Auth shape |
| D1 | sibling ticket `XGOJA-AUTH-PROD-DOCS` | go-go-goja ttmp | High | Good scope split; slightly aspirational outline |
| D2 | sibling tickets `XGOJA-AUTH-KEYCLOAK-MFA`, `XGOJA-AUTH-STORES` | go-go-goja ttmp | Medium | Referenced, not deeply read |
| E1 | cluster `README.md` | hetzner-k3s | High | Good "Start Here" map |
| E2 | `gitops/kustomize/go-go-host/` (full package) | hetzner-k3s | High | Gold-standard template; image pinned to old sha |
| E3 | `gitops/applications/go-go-host.yaml` | hetzner-k3s | High | Accurate Argo Application shape |
| E4 | `docs/app-runtime-secrets-and-identity-provisioning-playbook.md` | hetzner-k3s | Medium | Stale `corporate-headquarters` link; smailnail-centric |
| F1 | `keycloak/README.md` | terraform | High | Accurate structure map |
| F2 | `docs/shared-keycloak-platform-playbook.md` | terraform | **Low-Med** | **Repo map + "shared realm: smailnail" are stale** |
| F3 | `keycloak/apps/go-go-host/envs/k3s-beta/` (full env) | terraform | High | Best concrete template |
| F4 | `keycloak/modules/browser-client/main.tf` | terraform | High | Accurate confidential-client module |
| G1 | `README.md` | infra-tooling | High | Clear "what belongs here" |
| G2 | `docs/platform/source-repo-to-gitops-pr.md` | infra-tooling | Medium | Stale link; doesn't list who actually uses the flow |
| G3 | `templates/github/publish-image-ghcr.template.yml` | infra-tooling | High | Accurate caller template |

## Part A — go-go-goja (in-repo): examples & docs

### A1. `examples/xgoja/README.md` (xgoja learning path)

- **Researching:** which example is the production-oriented Keycloak auth host and which is the generated-runtime seam.
- **Looking for:** the numbered learning-path index and a one-line description of examples 18/19/20/21.
- **Why chosen:** the xgoja examples are the only concrete runnable implementations of the auth host.
- **How found:** direct `ls examples/` then `read examples/xgoja/README.md`.
- **Useful:** crisp per-example descriptions; the JSVerb source-filters explanation; the bulk smoke loop.
- **Not useful:** nothing — it is well-scoped.
- **Out of date / wrong:** none materially. Example 17 is noted as "Reserved; old route-authoring sketch removed" which is accurate.
- **Needs updating:** add a marker distinguishing **smoke-only** examples (20 hello-world) from the **production-template** example (19 keycloak-auth-host), and link 19 to the `XGOJA-AUTH-DEPLOY` ticket as its deployment guide.

### A2. example 19 `19-express-keycloak-auth-host/` (README + cmd/host/main.go + scripts/server.js + docker-compose.yml)

- **Researching:** the exact Go host to promote to production and its wiring.
- **Looking for:** how OIDC + Postgres stores + planned routes are composed; the config surface (flags/env); the production caveats the author already knew.
- **Why chosen:** README self-describes as "production-oriented companion" with the full `keycloakauth`/`sessionauth`/`appauth`/`audit`/`capability` stack.
- **How found:** the `examples/xgoja/README.md` learning path (A1).
- **Useful:** the full `main.go` is essentially the production wiring (`keycloakauth.New` config fields, `gojahttp.NewHost` AuthOptions, per-store memory/Postgres + `ApplySchema`); README lists exact production caveats (HTTPS, secure cookies, transaction store for multi-instance); `server.js` shows the planned-route DSL.
- **Not useful:** the in-memory store branches are demo-only noise for a production reading (though clearly useful for local smoke).
- **Out of date / wrong:** nothing wrong, but it is example-shaped by design: `AllowInsecureHTTP: true`, `127.0.0.1:8790` listen, no `/readyz`, hardcoded demo tenant `o1`/`demo@example.test`, `RedirectURL` derived from listen addr. These are the promotion gaps, not bugs.
- **Needs updating:** add a "Production deployment" pointer at the top of the README to `XGOJA-AUTH-DEPLOY` so readers don't treat the inline caveats as the whole story.

### A3. example 21 `21-generated-host-auth/` (README + xgoja.yaml + cmd/host/main.go)

- **Researching:** what "xgoja-generated" means for a host and how to wire it.
- **Looking for:** the `xgoja.yaml` artifact spec and the `xgojaruntime.NewBundle(Options{ConfigureServices})` seam.
- **Why chosen:** this is the canonical generated-runtime-package example; needed to satisfy the "xgoja generated" requirement.
- **How found:** A1 learning path.
- **Useful:** `xgoja.yaml` is a clean template for emitting `internal/xgojaruntime`; `main.go` shows the `hostauth.ServiceFactoryKey` injection; the README states the store-mode contract clearly.
- **Not useful:** the SQLite store demo flags are a distraction for a Postgres production target.
- **Out of date / wrong:** the README explicitly says **"Postgres and OIDC/Keycloak configuration remain follow-up work"** — so `hostauth.ModeOIDC` is unfinished. This is the single most important caveat for the whole ticket (it forces Decision D2: hybrid generated-seam + example-19 OIDC).
- **Needs updating:** cross-link the MFA/stores sibling tickets; note that `ModeOIDC` is the migration target once complete.

### A4. `AGENT.md`

- **Researching:** the build/test/lint conventions and any deployment hints.
- **Looking for:** how to run the binary, kill a port, web vs go guidelines.
- **Why chosen:** canonical agent/engineer entry point for the repo.
- **How found:** root `ls`.
- **Useful:** build commands (`go run ./...`), the `lsof-who -p $PORT -k` port-kill convention, go/web guidelines.
- **Not useful:** no deployment, image, or GitOps information at all.
- **Out of date / wrong:** none.
- **Needs updating:** add a short "Deployment" subsection pointing at `Dockerfile`, `.github/workflows/publish-image.yaml`, `deploy/gitops-targets.json`, and the cluster reference package.

## Part B — go-go-goja (in-repo): build, release, CI

### B1. `Dockerfile` + `.dockerignore`

- **Researching:** what image is built today and how (stages, uid, runtime).
- **Looking for:** the build target, frontend coupling, runtime user, exposed port, entrypoint.
- **Why chosen:** the deployment is image-based; the Dockerfile is the contract.
- **How found:** root `ls`; `cat Dockerfile`.
- **Useful:** clean 3-stage pattern (web-builder -> go-builder -> debian:12-slim), non-root uid/gid 65532, explicit `ENTRYPOINT`/`CMD`, `EXPOSE 8080`.
- **Not useful:** the entire `node`/`pnpm` frontend stage is irrelevant to the auth host (which has no frontend).
- **Out of date / wrong:** none, but it only builds `cmd/goja-repl` — there is **no auth-host target**, which is a gap, not an error.
- **Needs updating:** either add a second final stage for the auth host, or add a `Dockerfile.auth-host`; `.dockerignore` is fine as-is.

### B2. `deploy/gitops-targets.json`  ⚠️

- **Researching:** the GitOps handoff contract: which image, which repo, which manifest path, which container.
- **Looking for:** the exact `manifest_path` and `container_name` the publish workflow patches.
- **Why chosen:** it is the source-repo side of the image-based release chain.
- **How found:** root `ls deploy/`.
- **Useful:** shows the target schema (`name`, `gitops_repo`, `gitops_branch`, `manifest_path`, `container_name`).
- **Not useful / **wrong**:** its single target `goja-essay-prod` points at `gitops/kustomize/goja-essay/deployment.yaml`, and **that package does not exist** in the cluster repo (verified by `ls gitops/kustomize/`). So a `main` push today opens a GitOps PR against a missing path. This is the most important correctness finding in the whole investigation.
- **Needs updating:** (1) add the `goja-auth-host-prod` target (design §6.5); (2) add a `deploy/README.md` stating the contract and that the referenced GitOps package must pre-exist before the first `main` run; (3) resolve the stale `goja-essay` target (create the package or remove the target).

### B3. `.github/workflows/publish-image.yaml` + `scripts/open_gitops_pr.py`

- **Researching:** the existing publish + GitOps-PR automation to extend.
- **Looking for:** how the image is tagged (`sha-<short>`), how `main` vs PR is gated, how the GitOps PR is opened, what token it uses.
- **Why chosen:** CI is the only place that publishes, so the auth host must plug in here.
- **How found:** `ls .github/workflows/`.
- **Useful:** complete working pipeline — metadata-action tagging, GHA cache, `docker/build-push-action`, PR smoke of the loaded image, and the `open_gitops_pr.py` call with `GH_TOKEN`.
- **Not useful:** the PR-stage frontend build/test steps are essay-specific.
- **Out of date / wrong:** none functionally, but it uses the **repo-local** `scripts/open_gitops_pr.py` + a long-lived `GITOPS_PR_TOKEN` secret, whereas `infra-tooling` now offers a shared reusable workflow with Vault-backed GitHub Actions OIDC (see G2/G3). The doc-cleanup note: this divergence is not documented anywhere in the repo.
- **Needs updating:** add a CI/README note describing the current release path and the planned migration to the shared workflow (Decision D3).

### B4. `Makefile` + `.goreleaser.yaml`

- **Researching:** release/versioning tooling and whether goreleaser builds the deployable.
- **Looking for:** the `VERSION`, tag targets, what `goreleaser` builds.
- **Why chosen:** needed to know if releases are image-based or binary-based.
- **How found:** root `ls`.
- **Useful:** `goreleaser` builds `cmd/goja-repl` into deb/rpm/homebrew — clarifies that **binary releases are for the essay/repl, not for cluster deployment** (cluster uses the Docker image).
- **Not useful:** the `bump-go-go-golems`/lint targets are orthogonal.
- **Out of date / wrong:** none.
- **Needs updating:** minor — a one-line comment in the Makefile that cluster deployment uses the GHCR image from `publish-image.yaml`, not goreleaser.

## Part C — go-go-goja (in-repo): source code (auth APIs)

### C1. `pkg/gojahttp/auth/keycloakauth/` (keycloakauth.go, README)

- **Researching:** the exact OIDC config surface and handler set the host must call.
- **Looking for:** `Config` fields, `OIDCClaims`, `UserSession`, `New(...)`, and the `LoginHandler`/`CallbackHandler`/`LogoutHandler` methods.
- **Why chosen:** this is the production OIDC implementation; cannot design the host without its signature.
- **How found:** `find pkg/gojahttp/auth`; example 19's imports named the package.
- **Useful:** `Config{IssuerURL, ClientID, ClientSecret, RedirectURL, Scopes, AfterLoginURL, AfterLogoutURL, SessionManager, UserNormalizer, TransactionStore}` is exactly the wiring surface; `OIDCClaims` documents that `Subject` is the stable key (not email).
- **Not useful:** nothing.
- **Out of date / wrong:** none observed.
- **Needs updating:** when `hostauth.ModeOIDC` lands (sibling ticket), keep this `Config` shape identical so the migration is mechanical.

### C2. `pkg/xgoja/hostauth/` (config.go, doc.go)

- **Researching:** the generated-host auth config seam and whether OIDC is config-driven yet.
- **Looking for:** `Config`/`Mode`/`SessionConfig`/`StoresConfig` fields and the `ModeOIDC` status.
- **Why chosen:** example 21 injects auth through `hostauth.ServiceFactoryKey`; needed to know if production OIDC is reachable purely via config.
- **How found:** example 21 imports; `find pkg/xgoja/hostauth`.
- **Useful:** `doc.go` clearly states the package sits outside JS modules and is host-owned; `config.go` enumerates `Mode none|dev|oidc`, store drivers `memory|sqlite|postgres`, and per-store `ApplySchema`.
- **Not useful:** nothing.
- **Out of date / wrong:** **`ModeOIDC` is declared but not fully wired** (confirmed against A3's README). Trusting it as production-ready would be wrong.
- **Needs updating:** document in `doc.go` that `ModeOIDC` is in-progress and point to `XGOJA-AUTH-KEYCLOAK-MFA`.

### C3. `pkg/gojahttp/host.go`

- **Researching:** the Host constructor and the AuthOptions shape (authenticator/CSF/resources/authorizer/audit).
- **Looking for:** `NewHost(HostOptions{Dev, RejectRawRoutes, Auth{...}})` exact fields.
- **Why chosen:** the promoted host composes everything through this constructor.
- **How found:** example 19 import + grep.
- **Useful:** confirms the five auth seams and `RejectRawRoutes` default behavior.
- **Not useful:** nothing.
- **Out of date / wrong:** none.
- **Needs updating:** none.

## Part D — go-go-goja ttmp (sibling tickets)

### D1. `XGOJA-AUTH-PROD-DOCS` (index + design doc)

- **Researching:** whether production deployment docs already existed (to avoid duplication).
- **Looking for:** scope, proposed production-guide outline, policy-adapter plan, dependencies.
- **Why chosen:** `docmgr doc search --query "keycloak production deployment"` returned it as the top hit.
- **How found:** `docmgr doc search`.
- **Useful:** clean scope split — it owns production hardening *prose* and policy-adapter evaluation; its proposed outline (architecture/topology/Keycloak/sessions/CSRF/authz/audit/stores/checklist/troubleshooting) is a good complement to this ticket's *deployment* focus.
- **Not useful:** the policy-adapter (Casbin/OpenFGA/OPA) evaluation is orthogonal to deployment.
- **Out of date / wrong:** none; its dependencies correctly point at `XGOJA-AUTH-STORES` and `XGOJA-AUTH-KEYCLOAK-MFA`.
- **Needs updating:** once `XGOJA-AUTH-DEPLOY` lands, cross-link it from the PROD-DOCS index so the deployment story and the hardening prose reference each other.

### D2. `XGOJA-AUTH-KEYCLOAK-MFA` and `XGOJA-AUTH-STORES` (referenced only)

- **Researching:** the boundary of sibling work (stores, MFA, transaction store).
- **Looking for:** confirmation that durable stores + OIDC transaction storage are owned elsewhere.
- **Why chosen:** the design doc's non-goals depend on them.
- **How found:** search hits + cross-references from D1.
- **Useful:** their existence justifies this ticket's non-goals.
- **Not useful:** I did not deep-read them (out of time budget); treated only as scope boundaries.
- **Out of date / wrong:** unknown — not deeply read.
- **Needs updating:** when resuming, deep-read both to confirm the `TransactionStore` contract for multi-instance callbacks (a Risk in the design doc).

## Part E — cluster repo (`2026-03-27--hetzner-k3s`)

### E1. cluster `README.md`

- **Researching:** what the cluster is, what runs on it, and where the "bring your repo here" guides live.
- **Looking for:** the architecture model, the list of running apps, and the entry-point doc links.
- **Why chosen:** the user explicitly pointed here; it is the cluster source of truth.
- **How found:** direct `read` of the user-named path.
- **Useful:** the operating-model diagram (Terraform -> cloud-init -> GitOps -> Argo); the "Start Here" link list (troubleshooting, app-deployment-pipeline, runtime-secrets playbook, examples, GHCR pull pattern, OIDC).
- **Not useful:** nothing significant.
- **Out of date / wrong:** the "currently runs" list reads as authoritative; spot-check any app before assuming it's live (e.g. `smailnail` was removed 2026-06-06 per E4). Minor staleness risk.
- **Needs updating:** add an explicit "reference app to copy" callout for `go-go-host` (Keycloak+Vault+Postgres) and for a no-DB app, so new apps have an obvious template.

### E2. `gitops/kustomize/go-go-host/` (full package: ~18 manifests)

- **Researching:** the concrete manifest set to copy for a Keycloak+Vault+Postgres app.
- **Looking for:** sync-wave ordering, VaultAuth/VaultStaticSecret shape, ConfigMap config contract, image-pull secret rendering, db-bootstrap job, probes, ingress/TLS.
- **Why chosen:** it is the closest running analog to the auth host (same three dependencies).
- **How found:** noticed `go-go-host` appears in both `gitops/kustomize/` and `terraform/keycloak/apps/`; recognized it as the reference.
- **Useful:** everything — this is the gold-standard template. The ConfigMap shows the exact config keys an app must accept (`listenAddr`, `publicBaseUrl`, `oidcIssuer`, `oidcClientId`, `oidcRedirectPath`, ...); the runtime-secret VSO shows env-from-secret; the db-bootstrap job shows the idempotent role+DB creation.
- **Not useful:** the CLI device client / GitHub IdP / PVC parts are app-specific and not needed for the auth host.
- **Out of date / wrong:** the Deployment image is pinned `sha-6c833cb` (normal/immutable for GitOps, not an error); `fsGroup: 10001` while the go-go-goja image uses uid/gid `65532` — a consistency note, not a bug in this repo.
- **Needs updating:** none in this repo; but the new `goja-auth-host` package should omit the PVC (stateless) and match its own image gid.

### E3. `gitops/applications/go-go-host.yaml`

- **Researching:** the Argo `Application` shape and which project/labels to use.
- **Looking for:** `spec.project`, `destination`, `source.path`, `syncPolicy`, and the `scapegoat.dev/*` labels.
- **Why chosen:** every app needs exactly one of these, bootstrapped once.
- **How found:** `ls gitops/applications/`.
- **Useful:** shows `project: prod-apps`, `CreateNamespace=true`, `ServerSideApply=true`, and the capability labels (`has-database`, `has-ingress`, etc.).
- **Not useful:** nothing.
- **Out of date / wrong:** none.
- **Needs updating:** none.

### E4. `docs/app-runtime-secrets-and-identity-provisioning-playbook.md`

- **Researching:** the cross-repo operator sequence (Keycloak -> Vault -> VSO -> Argo) and the per-app prerequisites checklist.
- **Looking for:** the ordered phases, the minimal file contract per repo, and the common failure modes.
- **Why chosen:** G2 (source-repo-to-gitops) explicitly deferred to this doc for the identity/secrets half.
- **How found:** linked from `infra-tooling/docs/platform/source-repo-to-gitops-pr.md` and the cluster README.
- **Useful:** the system-boundary diagram, the 7-point pre-sync checklist, the recommended operator sequence (Phases 1-6), the minimal per-repo file contract, and the failure-mode catalog (`ImagePullBackOff`, VSO-no-Secret, redirect mismatch, PVC sync-wave).
- **Not useful:** the concrete `smailnail` validation commands are historical-only (app removed) — but the doc does say so.
- **Out of date / wrong:** **stale link** — references `/home/manuel/code/wesen/corporate-headquarters/infra-tooling/...` but the real path is `go-go-golems/infra-tooling`. Also smailnail-centric examples (noted as removed but still pervasive).
- **Needs updating:** fix the `corporate-headquarters` -> `go-go-golems` path; refresh the historical smailnail examples with a current app (go-go-host) as the worked example.

## Part F — terraform repo (`terraform`)

### F1. `keycloak/README.md`

- **Researching:** the Keycloak terraform layout and how apps are organised.
- **Looking for:** the apps/envs structure, the credential model, and the pointer to the stable playbook.
- **Why chosen:** canonical entry point for Keycloak-as-code.
- **How found:** user-named path; `read keycloak/README.md`.
- **Useful:** structure map (`modules/`, `apps/<app>/envs/{local,hosted,k3s-parallel,k3s-beta}`), credential model (`TF_VAR_*`), the `make scaffold-browser-app` shortcut list.
- **Not useful:** nothing.
- **Out of date / wrong:** the "Structure" list only enumerates smailnail/hair-booking/coinvault/infra-access — it omits `go-go-host` (and mirotalk-sfu, draft-review) which actually exist under `apps/`.
- **Needs updating:** refresh the Structure list to match `apps/` on disk.

### F2. `docs/shared-keycloak-platform-playbook.md`  ⚠️

- **Researching:** the operator playbook for adding a new Keycloak-backed app.
- **Looking for:** real endpoints, the source-of-truth rules, env semantics (local/hosted/k3s-parallel), the add-a-new-app checklist, and common failure modes.
- **Why chosen:** F1 pointed here as the stable guide.
- **How found:** linked from `keycloak/README.md`.
- **Useful:** the three-layer system model, the "read this before making changes" rules, the K3s-parallel rollout rule, the `scaffold-browser-app` path, the redirect/post-logout failure-mode notes, the adopt-existing-client (`import_existing`) guidance.
- **Not useful:** nothing major.
- **Out of date / wrong:** **two stale claims.** (1) The "Repository map" and "System model" center on `smailnail` ("Hosted shared realm today: smailnail"), but smailnail was removed 2026-06-06. (2) The repo map lists only smailnail/hair-booking `ttmp`, omitting the other apps now present. The K3s host is `auth.yolo.scapegoat.dev`; older prose sometimes implies `auth.scapegoat.dev` — the distinction (Coolify-hosted vs in-cluster) matters and should be sharper.
- **Needs updating:** rewrite the repository map and "shared realm today" to reflect current apps (go-go-host, coinvault, etc.); add an explicit "which Keycloak host for which env" table (`auth.scapegoat.dev` Coolify vs `auth.yolo.scapegoat.dev` in-cluster).

### F3. `keycloak/apps/go-go-host/envs/k3s-beta/` (main.tf, variables.tf, providers.tf, versions.tf, terraform.tfvars.example, outputs.tf)

- **Researching:** a complete, working terraform env to copy for a new K3s Keycloak app.
- **Lookinging for:** realm/client resources, redirect-URI locals, GitHub IdP, admin role/user, backend config, provider auth, outputs.
- **Why chosen:** go-go-host is the same shape (browser app on K3s with GitHub SSO).
- **How found:** `find terraform/keycloak/apps/go-go-host`.
- **Useful:** everything — the `locals{ valid_redirect_uris, valid_post_logout_redirect_uris, web_origins }` block is the exact correctness surface; the confidential dashboard client + public CLI device client; the remote S3 backend key pattern; the `outputs` (realm, client IDs, callback URL).
- **Not useful:** the GitHub IdP + wesen-admin-user blocks are optional for the auth host.
- **Out of date / wrong:** none; this env is current.
- **Needs updating:** none; copy it for `goja-auth-host`.

### F4. `keycloak/modules/browser-client/main.tf`

- **Researching:** the reusable client module semantics.
- **Looking for:** whether the module produces a confidential client with scopes.
- **Why chosen:** needed to confirm the client type the auth host requires.
- **How found:** `find terraform/keycloak/modules`.
- **Useful:** confirms `access_type = "CONFIDENTIAL"` + `client_secret` + optional default/optional scope attachments — exactly the Decision D1 shape.
- **Not useful:** nothing.
- **Out of date / wrong:** none.
- **Needs updating:** none.

## Part G — infra-tooling (`go-go-golems/infra-tooling`)

### G1. `README.md`

- **Researching:** what infra-tooling owns and whether it has reusable release/GitOps mechanics.
- **Looking for:** the "what belongs here" rules and the reusable building blocks list.
- **Why chosen:** the user named it as a reference; it is the neutral home for shared mechanics.
- **How found:** user-named path; `read README.md`.
- **Useful:** clear ownership boundaries, the layout tree, the "Current Recommended Reuse Points" (package publishing, source-repo -> GitOps PR flow, GHCR publish workflow, open-gitops-pr action, validator script).
- **Not useful:** nothing.
- **Out of date / wrong:** none material.
- **Needs updating:** none.

### G2. `docs/platform/source-repo-to-gitops-pr.md`

- **Researching:** the canonical cross-repo release contract (image-based variant).
- **Looking for:** the release chain, ownership boundaries, the `deploy/gitops-targets.json` schema, the secret expectations, the private-GHCR boundary, the immutable-tag rule, the first-rollout reminder.
- **Why chosen:** this is the authoritative description of the flow go-go-goja participates in.
- **How found:** infra-tooling README links it; also referenced from the cluster playbook (E4).
- **Useful:** the "publishing is not deployment" framing, the ownership split, the full target-JSON schema (single/multi-container, `patch_strategy`), the Vault-backed OIDC token expectations, the `validate_gitops_targets.py` pointer.
- **Not useful:** the static-publisher-job and federated-remote variants are not relevant to the auth host.
- **Out of date / wrong:** **stale link** to `/home/manuel/code/wesen/corporate-headquarters/infra-tooling/...` (should be `go-go-golems`). Also it describes the *target* shared workflow but doesn't list **which repos currently use it vs the old repo-local helper** — go-go-goja still uses the repo-local `open_gitops_pr.py` (B3), which is an undocumented divergence.
- **Needs updating:** fix the link; add a "current adopters" table (who calls `publish-ghcr-image.yml@main` vs who still uses a local helper) so migration status is visible.

### G3. `templates/github/publish-image-ghcr.template.yml`

- **Researching:** the recommended caller-workflow shape for the shared flow.
- **Looking for:** the `uses:` line, the `with:` inputs (dockerfile, build_context, test_command, gitops_target_config, push_image, open_gitops_pr, tooling_repository/ref), required permissions.
- **Why chosen:** it is the template go-go-goja would adopt for the CI migration (Decision D3).
- **How found:** infra-tooling README; `ls templates/github`.
- **Useful:** shows the exact caller contract and the `secrets: inherit` + OIDC permission pattern.
- **Not useful:** nothing.
- **Out of date / wrong:** none.
- **Needs updating:** none.

## Part H — README / navigation improvement suggestions

These are the concrete doc changes that would have saved the most manual digging.
They are ordered by impact. None are implemented here (out of scope for this
ticket); each is a candidate for a small follow-up.

### H1. go-go-goja: add a root "Deployment" map  (highest impact)

Today there is **no single place** in go-go-goja that says how the repo ships
to the cluster. An engineer must independently discover `Dockerfile`,
`deploy/gitops-targets.json`, `.github/workflows/publish-image.yaml`, and the
cluster reference package. Add a short section to `README.md` (and mirror a
pointer in `AGENT.md`):

```markdown
## Deployment

This repo ships container images to the yolo.scapegoat.dev K3s cluster.

- Image build: `Dockerfile` (essay/repl) — add per-binary targets for new apps.
- Release pipeline: `.github/workflows/publish-image.yaml` -> GHCR `sha-<short>` tags.
- GitOps handoff: `deploy/gitops-targets.json` -> opens a PR in
  `wesen/2026-03-27--hetzner-k3s` (see `scripts/open_gitops_pr.py`).
- Cluster reference package (copy this): `gitops/kustomize/go-go-host/`.
- Cross-repo contract: `infra-tooling/docs/platform/source-repo-to-gitops-pr.md`.
- Deploying an auth host end-to-end: see ticket `XGOJA-AUTH-DEPLOY`.
```

### H2. go-go-goja: add `deploy/README.md` explaining the target contract

`deploy/gitops-targets.json` silently points at a path that does not exist
(`goja-essay`). A sibling `deploy/README.md` should state: the schema, that the
referenced `manifest_path` **must already exist** in the GitOps repo before the
first `main` push, and the immutable-tag rule (`sha-<short>` only).

### H3. go-go-goja: flag production-template vs smoke-only examples

In `examples/xgoja/README.md`, annotate the learning-path entries:
- `19-express-keycloak-auth-host` — **production template** (link to `XGOJA-AUTH-DEPLOY`).
- `21-generated-host-auth` — **generated-seam template** (note OIDC/Postgres are follow-up).
- `20-express-hello-world` — smoke-only.
This prevents an intern from assuming example 21 is production-ready.

### H4. terraform: refresh the shared-keycloak playbook  (highest impact outside go-go-goja)

`docs/shared-keycloak-platform-playbook.md` is the most stale doc I read:
- Rewrite the "Repository map" and "Hosted shared realm today" to drop
  smailnail (removed 2026-06-06) and list current apps (go-go-host, coinvault,
  hair-booking, draft-review, ...).
- Add a one-table "which Keycloak host for which env" reference
  (`auth.scapegoat.dev` Coolify-hosted vs `auth.yolo.scapegoat.dev` in-cluster).
- Refresh `keycloak/README.md` "Structure" list to match `apps/` on disk.

### H5. cluster repo + infra-tooling: fix stale `corporate-headquarters` links

Both `docs/app-runtime-secrets-and-identity-provisioning-playbook.md` (E4) and
`docs/platform/source-repo-to-gitops-pr.md` (G2) contain a link to
`/home/manuel/code/wesen/corporate-headquarters/infra-tooling/...`. The real
path is `go-go-golems/infra-tooling`. Fix both.

### H6. infra-tooling: add a "current adopters" table to the GitOps-PR doc

`docs/platform/source-repo-to-gitops-pr.md` describes the target shared flow
but not who uses it. Add a table of repos and their status (shared workflow /
local helper), so the migration off long-lived `GITOPS_PR_TOKEN` secrets is
visible. Today go-go-goja is silently on the old local helper.

### H7. cluster repo: name the reference packages explicitly

In the cluster `README.md` "Start Here" section, add two explicit copy-template
pointers so new apps don't have to be discovered by `ls`:
- Keycloak + Vault + Postgres app: `gitops/kustomize/go-go-host/`.
- no-DB / static app: `gitops/kustomize/docs-yolo/` (or similar).

## Summary of doc health

- **Trust as-is:** A1, A2, A4, B1, B3, B4, C1, C3, D1, E1, E2, E3, F1, F3, F4, G1, G3.
- **Trust with care (minor staleness):** A3 (incomplete by design), C2 (`ModeOIDC` unfinished), D2 (not deep-read), E4 (stale link + smailnail-centric), G2 (stale link + no adopters list).
- **Fix before relying on them:** **B2** (`goja-essay` target points at a non-existent package), **F2** (smailnail-centred repo map/realm claim is wrong now).
- **Highest-leverage README work:** H1 (go-go-goja deployment map) and H4 (terraform playbook refresh) — these two would have saved the majority of the manual digging.
