---
Title: Investigation diary
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
    - Path: .github/workflows/publish-image.yaml
      Note: Existing GHCR + GitOps PR pipeline to extend
    - Path: deploy/gitops-targets.json
      Note: |-
        Existing target metadata (essay), to be extended
        Existing target metadata (essay)
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: Production-oriented Keycloak/OIDC host; the implementation to promote
    - Path: examples/xgoja/21-generated-host-auth/xgoja.yaml
      Note: Generated-runtime-package seam template for xgoja generate
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: keycloakauth.Config and OIDC handlers
    - Path: pkg/xgoja/hostauth/config.go
      Note: hostauth.Config host-owned auth infra knobs
ExternalSources:
    - /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/go-go-host/
    - /home/manuel/code/wesen/terraform/keycloak/apps/go-go-host/envs/k3s-beta/
    - /home/manuel/code/wesen/go-go-golems/infra-tooling/docs/platform/source-repo-to-gitops-pr.md
Summary: 'Chronological investigation and planning diary for the XGOJA-AUTH-DEPLOY ticket: mapping the existing xgoja auth examples and the cross-repo cluster/terraform/infra-tooling platform, and producing an intern-ready deployment design.'
LastUpdated: 2026-06-16T09:30:00-04:00
WhatFor: Use to understand how the deployment design was derived, which files were evidence, and what the open operator actions are.
WhenToUse: Before resuming implementation work, or when reviewing why a particular wiring decision was chosen.
---


# Diary

## Goal

Capture the step-by-step investigation and design for productionizing an
xgoja-generated Keycloak auth host into the `yolo.scapegoat.dev` K3s cluster:
what I read, what I found, what was tricky, and what the resulting deployment
design is.

## Step 1: Orient on the repo, the request, and the platform references

This first step was pure orientation: confirming the workspace layout, the
scope guardrail (only modify `./go-go-goja/`), and locating the three external
reference trees the user named (`infra-tooling`, `2026-03-27--hetzner-k3s`,
`terraform`).

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr to put a proper xgoja generated http auth server in ./go-go-goja into produciton on our yolo.scapegoat.dev cluster. Look at ~/code/wesen/go-go-golems/infra-tooling/ and ~/code/wesen/2026-03-27--hetzner-k3s/ and ~/code/wesen/terraform/, figure out how to put a xgoja http auth example into productino, using our real keycloak server. Do not modify anything outside of ./go-go-goja/ , and ask me if modifications somewhere else are necessary. Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable. Keep a diary as you work."

**Assistant interpretation:** Produce a docmgr ticket with an intern-grade design/implementation guide for deploying an xgoja auth host to the K3s cluster using the real Keycloak, plus an investigation diary, then upload to reMarkable.

**Inferred user intent:** Onboard a new engineer onto the full cross-repo release chain by producing one authoritative, evidence-backed deployment doc, without modifying files outside `./go-go-goja/`.

### What I did
- Loaded the pinned skills (`diary`, `docmgr`, `ticket-research-docmgr-remarkable`) and their `references/*.md`.
- Ran `ls` on the workspace root, `go-go-goja/`, `docmgr status --summary-only`, and the three external trees.
- Confirmed the repo is `github.com/go-go-golems/go-go-goja`, has `go.work`, a `.ttmp.yaml`, and the auth subsystem lives in `pkg/gojahttp/auth/*`.

### What worked
- `docmgr status` showed 134 tickets / 341 docs already in `go-go-goja/ttmp`, so naming/structure conventions are established.
- The three reference trees resolved cleanly; `infra-tooling` README explicitly describes the "source repo -> GitOps PR" model.

### What I learned
- The platform already has a documented, reusable release chain; the job is to map our auth host onto it, not invent one.
- `go-go-goja` already publishes one image (`goja-repl`/essay) through `publish-image.yaml`, so a working GHCR + GitOps-PR pipeline exists to extend.

### Code review instructions
- Start at `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/AGENT.md` and `deploy/gitops-targets.json`.
- Run `docmgr status --summary-only` to confirm the ticket workspace root.

## Step 2: Map the auth examples and the planned-route model

The goal of this step was to find the exact in-repo implementation to promote and to understand the intent-vs-infrastructure split that makes this app "xgoja-generated." I read the example READMEs, the two key host `main.go` files, and the route script.

**User prompt (verbatim):** (see Step 1)

### What I did
- Read `examples/README.md` and `examples/xgoja/README.md` to learn the numbered learning path (examples 18/19/20/21 are the auth/Express set).
- Read `examples/xgoja/19-express-keycloak-auth-host/{README.md,cmd/host/main.go,scripts/server.js,docker-compose.yml}`.
- Read `examples/xgoja/21-generated-host-auth/{README.md,xgoja.yaml,cmd/host/main.go}`.
- Grepped `keycloakauth.Config` in `pkg/gojahttp/auth/keycloakauth/keycloakauth.go` and read `pkg/xgoja/hostauth/config.go` + `hostauth/doc.go`.

### Why
- Example 19 is "production-oriented" (full Keycloak OIDC + Postgres stores) and example 21 is the "generated-runtime-package seam" (the `xgoja generate` contract). The deployment must satisfy both: real OIDC *and* generated routes.

### What worked
- Example 19 already wires `keycloakauth.New` + `sessionauth` + `appauth` + `audit` + `capability`, each with memory and Postgres `sqlstore` + `ApplySchema`. So the hard auth code exists; what's missing is the deployment shape.
- Example 21 shows the generated-seam pattern (`xgojaruntime.NewBundle` + `hostauth.ServiceFactoryKey`), which is the answer to "what does xgoja-generated mean."

### What didn't work
- Example 21's README explicitly says: "Postgres and OIDC/Keycloak configuration remain follow-up work." So `hostauth.ModeOIDC` is not yet proven end-to-end; that work is tracked in `XGOJA-AUTH-KEYCLOAK-MFA`.

### What I learned
- The planned-route DSL (`app.get(...).auth(...).resource(...).csrf().allow(...).audit(...).handle(...)`) declares *intent*; the Go `Host` enforces it. Routes not declared are rejected (`RejectRawRoutes: true`).
- Example 19 reads config from flags + env (`KEYCLOAK_ISSUER`, `*_DB_DSN`, ...), which maps cleanly onto Kubernetes env-from-secret.
- The browser gets an opaque `sid` cookie; OIDC tokens stay server-side. This is the model to keep (Decision D1 in the design doc).

### What was tricky to build
- The tension between "use the generated seam (21)" and "OIDC is only in example 19" forced Decision D2 (hybrid): generate the route package via `xgoja generate`, but wire OIDC/Postgres using example 19's proven `keycloakauth.New`. Documenting this clearly was the trickiest part, because a naive reader would assume example 21 is production-ready.

### What warrants a second pair of eyes
- Whether the hybrid seam (generated routes + direct `keycloakauth.New`) compiles cleanly with the current `pkg/xgoja/hostauth` API, since `ModeOIDC` is unfinished. The promoted `cmd/goja-auth-host` must not depend on unfinished builder plumbing.

### What should be done in the future
- Finish `hostauth.ModeOIDC` (sibling ticket `XGOJA-AUTH-KEYCLOAK-MFA`) so the host can be fully config-driven via `hostauth.Config{Mode: oidc}` instead of direct `keycloakauth.New`. Keep the `keycloakauth.Config` shape identical to ease that migration.

### Code review instructions
- Open `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go` and trace `run()` -> `keycloakauth.New` + `gojahttp.NewHost`.
- `grep -n "ModeOIDC" pkg/xgoja/hostauth/*.go` to see the unfinished seam.

## Step 3: Map the cross-repo release chain and the gold-standard package

This step established *where* the app actually runs and *how* an image becomes a running Pod. I read the infra-tooling source-repo-to-GitOps doc, the cluster runtime-secrets playbook, and the full `go-go-host` GitOps package as the copy-template.

**User prompt (verbatim):** (see Step 1)

### What I did
- Read `infra-tooling/docs/platform/source-repo-to-gitops-pr.md` (the cross-repo contract) and `infra-tooling/README.md`.
- Read `2026-03-27--hetzner-k3s/docs/app-runtime-secrets-and-identity-provisioning-playbook.md` (the missing-bridge playbook).
- Read the entire `gitops/kustomize/go-go-host/` package (namespace, deployment, service, ingress, certificate, configmap, vault-auth/connection, runtime-secret, image-pull-secret, postgres-admin-secret, db-bootstrap job/script/serviceaccounts, kustomization).
- Read `gitops/applications/go-go-host.yaml` (Argo Application shape, `project: prod-apps`).

### Why
- `go-go-host` is a real, running Keycloak+Vault+Postgres app in this exact cluster. Copying its manifest set is the lowest-risk way to stand up a new app with the same dependencies.

### What worked
- The `go-go-host` configmap shows the exact config contract an app must accept (`listenAddr`, `publicBaseUrl`, `oidcIssuer`, `oidcClientId`, `oidcRedirectPath`, ...). The promoted `cmd/goja-auth-host` config mirrors it.
- The sync-wave annotations (`-3` namespace → `-2` SA/VaultAuth → `-1` VaultStaticSecret → `0..1` config/bootstrap → `2` deploy/service → `3` ingress) are a reusable ordering contract.

### What didn't work
- Nothing failed, but two subtleties needed care: (1) `VaultConnection.address` must be the in-cluster `http://vault.vault.svc.cluster.local:8200`, not the public Traefik hostname; (2) `local-path` PVCs must share the Deployment's sync wave (WaitForFirstConsumer). The design doc records both.

### What I learned
- *Publishing an image is not deployment.* The deployment moment is Argo reconciling a merged GitOps change. The first rollout also needs a one-time `kubectl apply -f gitops/applications/<app>.yaml`.
- The auth host is stateless apart from Postgres, so I decided to **omit the PVC** entirely (all persistence in Postgres), which sidesteps the PVC sync-wave trap.

### What was tricky to build
- Reconciling the two GitOps-PR paths in this ecosystem: this repo uses a repo-local `scripts/open_gitops_pr.py` + `GITOPS_PR_TOKEN` secret, while `infra-tooling` offers a shared reusable workflow with Vault-backed GitHub Actions OIDC. Decision D3 keeps the repo-local path for this ticket (minimal blast radius) and defers the shared-workflow migration to a fast-follow.

### What warrants a second pair of eyes
- The claim that the auth host needs **no PVC** (stateless). Verify no planned route writes to the local filesystem; all example-19 persistence is Postgres-backed.
- The cookie-`Secure`-behind-Traefik assumption: the host must honour `X-Forwarded-Proto` or login succeeds but the cookie is dropped. This needs a staging check.

### What should be done in the future
- Migrate `publish-image.yaml` to the shared `infra-tooling` reusable workflow + Vault OIDC (removes the long-lived `GITOPS_PR_TOKEN` source-repo secret).

### Code review instructions
- Diff `gitops/kustomize/go-go-host/` against the proposed `gitops/kustomize/goja-auth-host/` deltas in design doc §10.1.
- `kubectl -n go-go-host get deployment go-go-host -o yaml` to see a live reference.

## Step 4: Map the Keycloak terraform + Vault secret model

This step pinned down the identity and secret layer: how a new realm/client is created and how runtime secrets reach the Pod via VSO.

**User prompt (verbatim):** (see Step 1)

### What I did
- Read `terraform/keycloak/README.md` and `terraform/docs/shared-keycloak-platform-playbook.md`.
- Read the full `terraform/keycloak/apps/go-go-host/envs/k3s-beta/` env (`main.tf`, `variables.tf`, `providers.tf`, `versions.tf`, `terraform.tfvars.example`, `outputs.tf`).
- Read `terraform/keycloak/modules/browser-client/main.tf` (confidential client module).
- Confirmed the shared Postgres admin secret path `infra/postgres/cluster`.

### Why
- The auth host is meaningless without a matching Keycloak client. The terraform env is the source of truth for realm, client ID, redirect URIs, and the client secret that becomes `KEYCLOAK_CLIENT_SECRET`.

### What worked
- The `make scaffold-browser-app APP=... PUBLIC_APP_URL=...` helper exists exactly for this; the design doc §11 calls it as the first operator step.
- `go-go-host/envs/k3s-beta/main.tf` is a complete template (realm + dashboard client + CLI device client + platform-admin role + optional GitHub IdP + admin user), with a remote S3 backend.

### What I learned
- Browser-login apps use a **confidential** client (`access_type = CONFIDENTIAL`) carrying `client_secret`; that secret is operator-managed and seeded into Vault after `terraform apply` in the same session.
- For K3s apps the playbook prefers a dedicated `k3s-beta`/`k3s-parallel` env over mutating the older hosted env (keeps rollback simple, makes K3s callbacks explicit).
- When applying against the in-cluster Keycloak `auth.yolo.scapegoat.dev`, the operator reads admin creds from `kubectl -n keycloak get secret keycloak-bootstrap-admin`.

### What was tricky to build
- The ordering dependency: terraform apply must produce the client secret *before* the Vault runtime secret can be seeded, and the Vault secret must exist *before* the Pod can start. The design doc §11–§12 encodes this as a strict operator sequence.

### What warrants a second pair of eyes
- Redirect/post-logout URI exactness. A realm can look correct for `/auth/callback` and still break hosted logout if `valid_post_logout_redirect_uris` is wrong. The design doc §11 lists both explicitly.

### What should be done in the future
- If the auth host later exposes an MCP endpoint, add a dedicated MCP client (there is an `mcp-client` terraform module) instead of overloading the browser client.

### Code review instructions
- Run `make scaffold-browser-app APP=goja-auth-host PUBLIC_APP_URL=https://goja-auth.yolo.scapegoat.dev` in dry expectation and diff against `go-go-host/envs/k3s-beta`.
- `terraform -chdir=keycloak/apps/go-go-host/envs/k3s-beta state list` to see the live resource set to mirror.

## Step 5: Write the design doc and create the ticket

This step turned the investigation into the deliverable: a docmgr ticket with an intern-grade design/implementation guide, plus this diary.

**User prompt (verbatim):** (see Step 1)

### What I did
- Searched existing ttmp tickets with `docmgr doc search --query "keycloak production deployment"` and found two relevant siblings: `XGOJA-AUTH-PROD-DOCS` (production docs + policy-adapter plan) and `XGOJA-AUTH-KEYCLOAK-MFA` (Keycloak hardening + MFA). Read both to avoid scope overlap.
- Created ticket `XGOJA-AUTH-DEPLOY` with `docmgr ticket create-ticket` (topics: goja,keycloak,oidc,deployment,kubernetes,gitops,vault,security,backend).
- Added the design doc (`design/01-...`) and this diary (`reference/01-...`).
- Wrote the design doc following the `writing-style.md` structure: exec summary → problem/scope → system map → current-state (evidence) → gap analysis → proposed architecture → decision records (D1–D3) → runtime flows → phased plan → out-of-repo specs (cluster/terraform/vault) → test strategy → risks → references.

### Why
- The two sibling tickets cover *internal* hardening (stores, MFA, docs prose). This ticket covers the distinct *cross-repo cluster deployment* scope. Stating the non-goals explicitly prevents overlap.

### What worked
- The design doc is fully evidence-backed: every major claim references a concrete file (example 19/21, `go-go-host/`, the terraform env, the infra-tooling doc).
- Decision records D1–D3 capture the three non-obvious choices (confidential client + server sessions; generated routes + direct OIDC wiring; repo-local GitOps helper for now).

### What didn't work
- Minor: `docmgr doc add --doc-type design` auto-created an empty `design/02-...` stub in addition to the `01-...` I had already written directly. Fixed by deleting the `02-` stub. (Lesson: write content directly into the `doc add`-created path, or `doc add` then overwrite that exact file.)

### What I learned
- The scope guardrail ("do not modify outside `./go-go-goja/") shaped the whole doc: the out-of-repo cluster/terraform/vault work is specified verbatim in §10–§12 but explicitly marked **approval-gated operator actions**, not implemented here.

### What was tricky to build
- Keeping the doc intern-readable while covering four repositories. Solved with the §3 system-map diagram up front and consistent naming (`goja-auth-host` everywhere: binary, image, namespace, realm, client, hostname).

### What warrants a second pair of eyes
- The phased plan (design doc §9): Phases 1–4 are in-repo and within scope; Phases 5–7 are the approval-gated operator actions. Confirm the in/out boundary matches the user's intent before any code is written.
- Decision D2 (hybrid generated seam): confirm the promoted host does not depend on the unfinished `hostauth.ModeOIDC` builder.

### What should be done in the future
- After operator approval, implement Phase 1 (`cmd/goja-auth-host`) and run the example-19 Keycloak smoke against it before touching CI.
- Upload the design doc + diary bundle to reMarkable (Step 6 of this diary).

### Code review instructions
- Read `design/01-deploy-xgoja-keycloak-auth-host-to-yolo.md` end to end; cross-check every file reference resolves.
- Validate `deploy/gitops-targets.json` shape against `infra-tooling/examples/platform/image-gitops-targets.example.json`.

### Technical details
- Ticket path: `ttmp/2026/06/16/XGOJA-AUTH-DEPLOY--deploy-an-xgoja-generated-keycloak-auth-host-to-yolo-scapegoat-dev/`
- Design doc: `design/01-deploy-xgoja-keycloak-auth-host-to-yolo.md` (~53 KB, 15 sections).
- Diary: `reference/01-investigation-diary.md` (this file).

## Step 6: Validate, seed vocabulary, and upload to reMarkable

Final delivery step: pass `docmgr doctor`, resolve vocabulary warnings, and publish the bundle to reMarkable with a dry-run first.

**User prompt (verbatim):** (see Step 1)

### What I did
- Ran `docmgr doctor --ticket XGOJA-AUTH-DEPLOY --stale-after 30`; it warned about four unknown topics: `deployment`, `gitops`, `kubernetes`, `vault`.
- Added them to the vocabulary (`docmgr vocab add --category topics ...`) because they are legitimate, reusable platform topics.
- Re-ran doctor → ✅ All checks passed.
- Verified reMarkable: `remarquee status` (ok) and `remarquee cloud account --non-interactive` (user=wesen@ruinwesen.com).
- Dry-run bundle upload of design + diary → clean.
- Real bundle upload to `/ai/2026/06/16/XGOJA-AUTH-DEPLOY`, then `remarquee cloud ls ... --long` confirmed the PDF landed.

### Why
- Doctor must pass cleanly before handoff (deliverable checklist). The four new topics will recur for any future cluster-deployment ticket, so vocabulary seeding is the right fix rather than ignoring the warning.

### What worked
- Dry-run then real upload completed; the bundle (single PDF with ToC, depth 2) is verified at the remote path.

### What didn't work
- Nothing.

### What I learned
- The ticket-research-docmgr-remarkable deliverable checklist is strict but mechanical: doctor → vocab → dry-run → upload → verify. Following it in order avoids rework.

### What was tricky to build
- The out-of-repo boundary. The whole ticket had to deliver a complete deployment design *without* touching the cluster/terraform/vault repos, so §10–§12 of the design doc had to be specified verbatim enough for an operator to apply directly.

### What warrants a second pair of eyes
- Confirm the uploaded PDF renders the diagrams/code blocks legibly on reMarkable (pandoc -> xelatex can mangle wide ASCII diagrams). If the system-map or login-flow diagram is unreadable, regenerate with narrower columns.

### What should be done in the future
- After operator answers the four open questions in design doc §14 (hostname, GHCR visibility, single vs split DB, shared-workflow migration), proceed to Phase 1 implementation of `cmd/goja-auth-host`.

### Code review instructions
- `docmgr doctor --ticket XGOJA-AUTH-DEPLOY --stale-after 30` → all passed.
- `remarquee cloud ls /ai/2026/06/16/XGOJA-AUTH-DEPLOY --long --non-interactive` → PDF present.

### Technical details
- Vocabulary added: `deployment`, `gitops`, `kubernetes`, `vault`.
- reMarkable remote: `/ai/2026/06/16/XGOJA-AUTH-DEPLOY/XGOJA-AUTH-DEPLOY — Deploy xgoja Keycloak auth host to yolo.pdf`.
- Bundle members: `design/01-...` (primary guide) + `reference/01-...` (this diary).
