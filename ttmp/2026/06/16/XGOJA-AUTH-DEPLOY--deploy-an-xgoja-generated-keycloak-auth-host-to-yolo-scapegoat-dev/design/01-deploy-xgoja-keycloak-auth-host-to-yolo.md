---
Title: Deploy an xgoja-generated Keycloak auth host to yolo.scapegoat.dev
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
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: .github/workflows/publish-image.yaml
      Note: Existing GHCR + open_gitops_pr.py pipeline to extend for the auth-host image
    - Path: Dockerfile
      Note: |-
        Current Dockerfile builds only cmd/goja-repl (essay); must add an auth-host build target
        Current Dockerfile builds only cmd/goja-repl; must add an auth-host build target
    - Path: deploy/gitops-targets.json
      Note: |-
        Current target references a non-existent goja-essay package; add a goja-auth-host target
        Current target references non-existent goja-essay package; add a goja-auth-host target
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: |-
        Production-oriented Keycloak/OIDC host; reference implementation to promote into cmd/goja-auth-host
        Production-oriented Keycloak/OIDC host to promote into cmd/goja-auth-host
    - Path: examples/xgoja/19-express-keycloak-auth-host/scripts/server.js
      Note: |-
        Planned Express route declarations (healthz, me, project update, invites) embedded by the host
        Planned Express route declarations embedded by the host
    - Path: examples/xgoja/21-generated-host-auth/cmd/host/main.go
      Note: Generated-runtime-package seam (hostauth.ServiceFactoryKey injection) used as the route-embedding model
    - Path: examples/xgoja/21-generated-host-auth/xgoja.yaml
      Note: xgoja/v2 spec that emits internal/xgojaruntime; template for the production route artifact
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: |-
        keycloakauth.Config (IssuerURL, ClientID, ClientSecret, RedirectURL, ...) and OIDC handlers
        keycloakauth.Config (IssuerURL
    - Path: pkg/gojahttp/host.go
      Note: |-
        gojahttp.NewHost + HostOptions.Auth (Authenticator, CSRF, Resources, Authorizer, Audit)
        gojahttp.NewHost + HostOptions.Auth (Authenticator
    - Path: pkg/xgoja/hostauth/config.go
      Note: |-
        hostauth.Config (Mode dev|oidc, Session, Stores) — the host-owned auth infra config knobs
        hostauth.Config (Mode dev|oidc
ExternalSources:
    - /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/go-go-host/
    - /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/go-go-host.yaml
    - /home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-runtime-secrets-and-identity-provisioning-playbook.md
    - /home/manuel/code/wesen/terraform/keycloak/apps/go-go-host/envs/k3s-beta/
    - /home/manuel/code/wesen/terraform/docs/shared-keycloak-platform-playbook.md
    - /home/manuel/code/wesen/go-go-golems/infra-tooling/docs/platform/source-repo-to-gitops-pr.md
Summary: Intern-facing analysis, design, and implementation guide for shipping an xgoja-generated Keycloak auth host from this repo onto the yolo.scapegoat.dev K3s cluster, using the real in-cluster Keycloak, Vault, PostgreSQL, and Argo CD GitOps pipeline.
LastUpdated: 2026-06-16T00:00:00-04:00
WhatFor: Use when productionizing the gojahttp Keycloak auth host example into a real cluster deployment, and when onboarding a new engineer to the cross-repo release chain.
WhenToUse: Before writing the in-repo cmd/goja-auth-host, Dockerfile target, or deploy/gitops-targets.json entry, and before requesting the out-of-repo gitops/terraform/vault changes.
---


# Deploy an xgoja-generated Keycloak auth host to yolo.scapegoat.dev

## 1. Executive summary

This document is the complete, intern-facing guide for taking an
**xgoja-generated HTTP authentication server** that already exists in this
repository (`go-go-goja`) and running it for real on our
`yolo.scapegoat.dev` Kubernetes cluster, authenticated by our real shared
Keycloak.

The repository already contains a working, production-oriented Keycloak auth
host (`examples/xgoja/19-express-keycloak-auth-host`) and a generated-runtime
seam (`examples/xgoja/21-generated-host-auth`). What is missing is the
**deployment story**: a production binary, a container image, a GitOps package,
a Keycloak client, and the secret/identity wiring that lets the cluster
actually pull, start, and authenticate the app.

The plan has two halves:

1. **In-repo (this repository).** Promote the example host into a first-class
   production command (`cmd/goja-auth-host`), give it a Dockerfile build
   target, wire it into the existing GHCR + GitOps-PR publish workflow, and
   point `deploy/gitops-targets.json` at a new cluster package.
2. **Out-of-repo (needs operator approval).** Create the matching Kustomize +
   Argo package in the cluster repo, the Keycloak realm/client in the terraform
   repo, and the Vault runtime/image-pull secrets + Kubernetes auth roles.

This guide explains every layer an intern needs to understand, with file
references, pseudocode, diagrams, and decision records, so that the work can be
executed without re-deriving the platform context.

> **Scope guardrail.** Per the task brief, this ticket only modifies files
> inside `./go-go-goja/`. Every change required in the cluster repo, the
> terraform repo, or Vault is listed in §10 and §11 as an **approval-gated
> operator action**, not implemented here.

## 2. Problem statement and scope

### 2.1 What we are deploying

A small HTTP server written in Go that:

- embeds a JavaScript (Goja) runtime and serves a set of **planned routes**
  declared in JavaScript (the "xgoja-generated" part),
- authenticates browser users against our real **Keycloak** using OIDC
  Authorization Code + PKCE,
- keeps **opaque server-side app sessions** (not browser-held tokens),
- persists sessions, audit records, app users/resources, and capability tokens
  in **PostgreSQL**,
- enforces **CSRF**, **resource authorization**, and **audit logging** on
  unsafe routes.

This is exactly the shape of `examples/xgoja/19-express-keycloak-auth-host`.

### 2.2 Why it is not done yet (gap summary)

| Area | Current state | Gap for production |
| --- | --- | --- |
| Host code | Example host exists, runs locally on `127.0.0.1:8790` | Example-shaped: hardcoded demo tenant `o1`, `AllowInsecureHTTP: true`, no `/readyz`, no graceful shutdown, config via flags not a config file |
| Container | `Dockerfile` builds **only** `cmd/goja-repl` (the essay app) | No image for the auth host |
| CI/CD | `publish-image.yaml` publishes `ghcr.io/go-go-golems/go-go-goja` (essay) and opens a GitOps PR | Auth-host image not built/published; no GitOps target for it |
| GitOps target | `deploy/gitops-targets.json` points at `gitops/kustomize/goja-essay/deployment.yaml` | That package **does not exist** in the cluster repo; and there is no auth-host target at all |
| Keycloak | Real Keycloak runs at `auth.yolo.scapegoat.dev` | No realm/client for the auth host |
| Secrets | Vault + VSO pattern is proven (`go-go-host`) | No runtime/image-pull secret paths or k8s auth roles for the auth host |

### 2.3 In scope

- Design and implement the **in-repo** production host, image, and pipeline.
- Produce the **exact** out-of-repo manifests/terraform/vault specs so an
  operator can apply them after approval.
- Document the end-to-end release and runtime flows for an intern.

### 2.4 Out of scope (handled by sibling tickets)

- Persistent-store schema/migrations design → `XGOJA-AUTH-STORES`.
- Keycloak MFA + OIDC transaction storage primitives → `XGOJA-AUTH-KEYCLOAK-MFA`.
- Production hardening docs (cookie/TLS/proxy policy prose) → `XGOJA-AUTH-PROD-DOCS`.
- Replacing `appauth.Authorizer` with a policy engine → future optional work.

## 3. System map (read this first)

If you read nothing else, read this. The deployment is a chain of **four
repositories plus the cluster**. Each box is an ownership boundary with its own
failure modes.

```text
┌─────────────────────────┐   ┌──────────────────────────┐
│  THIS REPO              │   │  infra-tooling           │
│  go-go-goja             │   │  go-go-golems/infra-tooling│
│                         │   │                          │
│  cmd/goja-auth-host     │   │  .github/workflows/      │
│  Dockerfile (+ target)  │   │    publish-ghcr-image.yml│
│  deploy/gitops-targets  │──▶│  actions/open-gitops-pr  │
│  .github/workflows/     │   │  docs/platform/          │
│    publish-image.yaml   │   │    source-repo-to-gitops │
└─────────────────────────┘   └──────────────────────────┘
            │                              │
            │  1) build image              │  3) patch manifest + open PR
            ▼                              ▼
┌─────────────────────────┐   ┌──────────────────────────┐
│  GHCR                   │   │  GITOPS REPO             │
│  ghcr.io/go-go-golems/  │   │  2026-03-27--hetzner-k3s │
│    goja-auth-host:sha-* │   │                          │
└─────────────────────────┘   │  gitops/kustomize/       │
            │                  │    goja-auth-host/       │
            │  2) pull         │  gitops/applications/    │
            │                  │    goja-auth-host.yaml   │
            │                  │  vault/{policies,roles}/ │
            │                  └──────────────────────────┘
            │                              │
            │                              │  4) Argo CD reconciles
            ▼                              ▼
┌─────────────────────────────────────────────────────────┐
│  CLUSTER  (yolo.scapegoat.dev, single-node Hetzner K3s) │
│                                                         │
│  namespace: goja-auth-host                              │
│   Pod  (pulls image, reads VSO Secret, mounts ConfigMap)│
│   Ingress  goja-auth.yolo.scapegoat.dev  (Traefik + TLS)│
│                                                         │
│  shared platform: Keycloak, Vault+VSO, PostgreSQL,      │
│                   cert-manager, Argo CD                 │
└─────────────────────────────────────────────────────────┘
            ▲
            │  identity + realm + client created by:
┌─────────────────────────┐
│  TERRAFORM REPO         │
│  terraform              │
│  keycloak/apps/         │
│    goja-auth-host/      │
│      envs/k3s-beta/     │
└─────────────────────────┘
```

**Golden rule:** *Publishing an image is not deployment.* Deployment is the
moment Argo CD reconciles a merged change in the GitOps repo. Until then,
nothing runs. (See
`infra-tooling/docs/platform/source-repo-to-gitops-pr.md`.)

## 4. Current-state architecture (evidence-based)

This section grounds every later claim in real files. An intern should open
each referenced file once before continuing.

### 4.1 What go-go-goja is

`go-go-goja` is a Go framework that embeds a JavaScript runtime (the
[`dop251/goja`](https://github.com/dop251/goja) VM) and lets Go applications
expose host capabilities to JS, declare HTTP routes in JS, and generate
CLI binaries from a declarative `xgoja.yaml` spec. Two packages matter here:

- **`pkg/gojahttp`** — the HTTP host framework. It owns the `*http.ServeMux`,
  the **planned route registry** (routes must be declared up front; raw routes
  are rejected by default), and the **auth subsystem**.
- **`pkg/xgoja`** — code generation (`xgoja generate`). It emits an importable
  Go package (`internal/xgojaruntime`) that bundles JS module sources and Cobra
  commands into a host binary. `pkg/xgoja/hostauth` defines the host-owned auth
  configuration seam.

Evidence:

- `pkg/gojahttp/host.go` — `gojahttp.NewHost(HostOptions{Dev, RejectRawRoutes, Auth{...}})`.
- `pkg/gojahttp/auth/` — subpackages `keycloakauth`, `sessionauth`, `appauth`,
  `capability`, `audit`, each with a memory and a Postgres `sqlstore`.

### 4.2 The planned-route + auth model (the heart of the app)

The design separates **intent** (declared in JS) from **infrastructure**
(owned in Go). This is the single most important concept for an intern.

```text
JavaScript (express DSL)            Go host (gojahttp + auth/*)
─────────────────────────           ────────────────────────────
app.get("/me")                      keycloakauth   -> OIDC login/callback/logout
  .auth(express.user().required())  sessionauth    -> opaque cookie + CSRF
  .allow("user.self.read")          appauth        -> users/tenants/resources
  .audit("user.self.read")          appauth        -> deny-by-default authorizer
  .handle((ctx,res)=>{...})         audit          -> JSON audit sink
                                    gojahttp.Host  -> route registry + enforcer
```

A route's chain (`.auth`, `.resource`, `.csrf`, `.allow`, `.audit`) is a
**declaration**. The Go `Host` enforces it: an undeclared route is rejected
(`RejectRawRoutes: true`), an unsafe route without `.csrf()` is rejected, and
every `.allow(action)` is checked against the `Authorizer` before the handler
runs. Handlers receive a `ctx` with `actor`, `resource(...)`, `request`, etc.

Evidence — `examples/xgoja/19-express-keycloak-auth-host/scripts/server.js`:

```js
app.patch("/orgs/:orgId/projects/:projectId")
  .auth(express.user().required())
  .resource(express.resource("project").idFromParam("projectId").tenantFromParam("orgId").mustExist())
  .csrf()
  .allow("project.update")
  .audit("project.updated")
  .handle((ctx, res) => { res.json({ updated: ctx.resource("project").id }) })
```

### 4.3 Example 19 — the production-oriented Keycloak host (reference)

File: `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go`.

This is the closest thing we have to a production binary. It already wires the
full auth stack against Postgres:

- Reads config from flags + env:
  `--listen`, `--script`, `--issuer` (`KEYCLOAK_ISSUER`),
  `--client-id` (`KEYCLOAK_CLIENT_ID`), `--client-secret` (`KEYCLOAK_CLIENT_SECRET`),
  `--session-db-dsn` (`SESSION_DB_DSN`), `--audit-db-dsn` (`AUDIT_DB_DSN`),
  `--app-db-dsn` (`APPAUTH_DB_DSN`), `--capability-db-dsn` (`CAPABILITY_DB_DSN`).
- Builds `keycloakauth.New(ctx, Config{IssuerURL, ClientID, ClientSecret, RedirectURL: ".../auth/callback", AfterLoginURL:"/", AfterLogoutURL:"/", SessionManager: sessions, UserNormalizer: ...})`.
- Builds `gojahttp.NewHost(HostOptions{Dev:true, RejectRawRoutes:true, Auth:{Authenticator:sessions, CSRF:sessions, Resources:appauth.Resolver{...}, Authorizer:appauth.Authorizer{...}, Audit:auditSink}})`.
- Mounts an Express runtime, runs the JS route script, and serves:
  `GET /auth/login`, `GET /auth/callback`, `POST /auth/logout`, `GET /auth/session`,
  plus the JS-declared `/healthz`, `/me`, `/async-*`, and the project route.
- Stores: each of session/audit/appauth/capability has a memory fallback and a
  Postgres `sqlstore` with `ApplySchema(ctx)` on startup.

Example 19's own README lists what still must change for production: HTTPS,
secure cookies (not `AllowInsecureHTTP`), keep the Postgres stores, review
realm/client settings, add a shared transaction store for multi-instance
callbacks, keep authorization in Go.

### 4.4 Example 21 — the generated-runtime-package seam (the "xgoja generated" part)

Files: `examples/xgoja/21-generated-host-auth/{xgoja.yaml,cmd/host/main.go,verbs/sites.js,internal/xgojaruntime/}`.

This is the canonical "xgoja-generated" pattern. `xgoja generate` consumes
`xgoja.yaml` and emits `internal/xgojaruntime/xgoja_runtime.gen.go`, which
exposes `xgojaruntime.NewBundle(Options{ConfigureServices})`. The host imports
that generated package and injects auth infrastructure through a typed
host-service key:

```go
bundle, _ := xgojaruntime.NewBundle(xgojaruntime.Options{
    ConfigureServices: func(s *app.HostServices) {
        _ = s.SetHostService(hostauth.ServiceFactoryKey,
            hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: defaultAuthConfig()}))
    },
})
```

`hostauth.Config` (file: `pkg/xgoja/hostauth/config.go`) is the host-owned
config surface: `Mode` (`none|dev|oidc`), `Session` (cookie, idle/absolute
timeouts), and `Stores` (per-store `Driver` `memory|sqlite|postgres`, `DSN`,
`ApplySchema`). The JS side only declares routes — it never owns stores or
session policy.

**Known gap (important):** Example 21's README states *"Postgres and OIDC/
Keycloak configuration remain follow-up work; this example focuses on the
generated-host seam and dev/session-store foundation."* The `hostauth.ModeOIDC`
builder plumbing is therefore not yet proven end-to-end; that work is tracked
under `XGOJA-AUTH-KEYCLOAK-MFA`. See Decision D2.

### 4.5 The current build/release chain in this repo

- `Dockerfile` — multi-stage, builds **only** `cmd/goja-repl` (the "essay" web
  app) into `ghcr.io/.../go-go-goja`. It does not build any auth host.
- `.github/workflows/publish-image.yaml` — on push to `main` and on PRs: builds
  the essay frontend, runs `go test ./...`, builds/pushes the essay image to
  `ghcr.io/go-go-golems/go-go-goja` with a `sha-<short>` tag, and (on `main`)
  calls `scripts/open_gitops_pr.py --config deploy/gitops-targets.json ...`.
- `deploy/gitops-targets.json` — a single target `goja-essay-prod` pointing at
  `gitops/kustomize/goja-essay/deployment.yaml` (container `goja-essay`).
- `scripts/open_gitops_pr.py` — repo-local helper that patches the image field
  in the named GitOps manifest and opens a PR.

> Note: there is **no** `gitops/kustomize/goja-essay/` package in the cluster
> repo today. The essay target is aspirational and currently a no-op if the
> GitOps path is missing. The auth-host target we add must not repeat this:
> the cluster package (§10) must exist before the first real rollout.

### 4.6 The cluster (`yolo.scapegoat.dev`)

Single-node Hetzner K3s + Argo CD, defined in `2026-03-27--hetzner-k3s`.
Relevant shared services and endpoints:

| Thing | Where |
| --- | --- |
| Argo CD | `argocd` namespace; reconciles `gitops/applications/*.yaml` |
| Keycloak (in-cluster, real) | `https://auth.yolo.scapegoat.dev`, per-app realms `auth.yolo.scapegoat.dev/realms/<app>` |
| Vault + Vault Secrets Operator (VSO) | internal `http://vault.vault.svc.cluster.local:8200`; operator `https://vault.yolo.scapegoat.dev` |
| PostgreSQL (shared) | admin secret at Vault path `infra/postgres/cluster` |
| cert-manager | ClusterIssuer `letsencrypt-prod-dns01-digitalocean` |
| Ingress | Traefik; app hostnames follow `<app>.yolo.scapegoat.dev` |
| Argo project for apps | `prod-apps` |

The canonical cross-repo operator sequence is documented in
`2026-03-27--hetzner-k3s/docs/app-runtime-secrets-and-identity-provisioning-playbook.md`.

### 4.7 The gold-standard reference package: `go-go-host`

The single best thing to copy is `gitops/kustomize/go-go-host/` (a real,
running Keycloak+Vault+Postgres app). Its manifest set is the template for our
new package:

```text
go-go-host/
  namespace.yaml                 argocd sync-wave -3
  serviceaccount.yaml            -2  (references image-pull secret)
  db-bootstrap-serviceaccount.yaml -2
  vault-connection.yaml          -2  address: http://vault.vault.svc.cluster.local:8200
  vault-auth.yaml                -2  VaultAuth k8s role = serviceAccount
  db-bootstrap-vault-auth.yaml   -2
  runtime-secret.yaml            -1  VaultStaticSecret apps/<app>/beta/runtime -> Secret
  image-pull-secret.yaml         -1  VaultStaticSecret apps/<app>/beta/image-pull -> dockerconfigjson
  postgres-admin-secret.yaml     -1  VaultStaticSecret infra/postgres/cluster
  db-bootstrap-script-configmap.yaml 0  (creates app DB + role via psql)
  db-bootstrap-job.yaml          1   argocd hook Sync, BeforeHookCreation
  configmap.yaml                 1   config.yaml with ${ENV} placeholders
  persistentvolumeclaim.yaml     2   local-path, same wave as Deployment
  deployment.yaml                2   image sha-*, readiness /readyz, liveness /healthz
  service.yaml                   2
  certificate.yaml               2   cert-manager, dnsNames <app>.yolo.scapegoat.dev
  ingress.yaml                   3   traefik, TLS -> certificate secret
  kustomization.yaml             lists all, namespace + commonLabels
```

Its `configmap.yaml` shows the config contract an app must accept:

```yaml
config.yaml: |
  listenAddr: "0.0.0.0:8080"
  publicBaseUrl: "https://hosting.yolo.scapegoat.dev"
  oidcIssuer: "https://auth.yolo.scapegoat.dev/realms/go-go-host"
  oidcClientId: "go-go-host-dashboard"
  oidcRedirectPath: "/app/auth/callback"
  devAuth: false
  ...
```

and the `Deployment` reads the DB DSN from the VSO-rendered `Secret`:

```yaml
env:
  - name: GO_GO_HOST_CONTROL_DB_DSN
    valueFrom:
      secretKeyRef: { name: go-go-host-runtime, key: dsn }
```

### 4.8 The Keycloak terraform pattern

File: `terraform/keycloak/apps/go-go-host/envs/k3s-beta/`.

Each app gets its own realm at `auth.yolo.scapegoat.dev/realms/<app>` with a
browser client (standard flow), optional CLI device client, optional GitHub
identity provider, and a platform-admin role/user. Backend state is remote
(S3 `go-go-golems-tf-state`). The repo provides a scaffolder:

```bash
make scaffold-browser-app APP=<new-app> PUBLIC_APP_URL=https://<new-app>.yolo.scapegoat.dev
```

See `terraform/docs/shared-keycloak-platform-playbook.md`. For K3s apps the
playbook recommends a dedicated `k3s-parallel` (or `k3s-beta`, as `go-go-host`
uses) environment rather than mutating the older hosted env.

## 5. Gap analysis

Mapping §2.2 gaps to concrete work items:

| # | Gap | Owner repo | Fix |
| --- | --- | --- | --- |
| G1 | Example host is example-shaped | go-go-goja | Promote to `cmd/goja-auth-host` with prod defaults (§6.2) |
| G2 | No auth-host container image | go-go-goja | Add Dockerfile build target (§6.3) |
| G3 | Auth-host not in CI publish | go-go-goja | Extend `publish-image.yaml` (§6.4) |
| G4 | No GitOps target for auth host | go-go-goja | Add entry to `deploy/gitops-targets.json` (§6.5) |
| G5 | No Kustomize package | cluster repo | New `gitops/kustomize/goja-auth-host/` (§10) — **approval gated** |
| G6 | No Argo Application | cluster repo | New `gitops/applications/goja-auth-host.yaml` (§10) |
| G7 | No Keycloak realm/client | terraform repo | New `keycloak/apps/goja-auth-host/envs/k3s-beta` (§11) — **approval gated** |
| G8 | No Vault secrets/roles | cluster repo + Vault | Seed `apps/goja-auth-host/beta/{runtime,image-pull}`, k8s auth roles (§12) |

## 6. Proposed architecture (in-repo)

### 6.1 Naming and endpoints

- **Binary / image / app:** `goja-auth-host`
- **Image:** `ghcr.io/go-go-golems/goja-auth-host:sha-<short>`
- **Public URL:** `https://goja-auth.yolo.scapegoat.dev`
- **Namespace:** `goja-auth-host`
- **Keycloak realm:** `goja-auth-host` → issuer `https://auth.yolo.scapegoat.dev/realms/goja-auth-host`
- **Keycloak client:** `goja-auth-host-web` (confidential, standard flow)
- **Redirect URI:** `https://goja-auth.yolo.scapegoat.dev/auth/callback`
- **Post-logout URI:** `https://goja-auth.yolo.scapegoat.dev/` and `/*`

> Decision D1 (§7) records why we use a confidential client + server-side
> sessions instead of a public SPA client.

### 6.2 The production host: `cmd/goja-auth-host`

Create `cmd/goja-auth-host/main.go` by promoting example 19 with these
changes. Every change is justified by a production requirement.

```text
cmd/goja-auth-host/
  main.go        promoted + productionized host
  config.go      (optional) Glazed config struct mirroring go-go-host config.yaml
  routes/        go:embed the JS route script (or import the generated xgojaruntime)
```

Productionization checklist (each item maps to example 19's known gaps):

1. **Listen on `0.0.0.0:8080`**, not `127.0.0.1:8790` (Pods must bind all interfaces).
2. **Secure sessions.** `sessionauth.Config.AllowInsecureHTTP = false`. Behind
   Traefik TLS the request is HTTPS at L7; the host must also honour
   `X-Forwarded-Proto` so the cookie is marked `Secure`. Set `SameSite=Lax`
   (needed for the OIDC redirect top-level GET to keep the cookie).
3. **Graceful shutdown.** Replace `server.ListenAndServe()` with
   `signal.NotifyContext` + `server.Shutdown(ctx)` so Argo rollouts drain
   in-flight requests (go-go-host uses `strategy: Recreate`; still drain).
4. **`/readyz`.** Example 19 only has `/healthz`. Argo/K3s readiness needs a
   distinct `/readyz` (DB ping) vs liveness `/healthz` (process alive).
5. **Config from env/file, not flags.** Read a Glazed/YAML config like
   go-go-host's `config.yaml` (mounted from ConfigMap) plus secrets from env
   (rendered by VSO). Keep the example's env names but formalize them:
   `KEYCLOAK_ISSUER`, `KEYCLOAK_CLIENT_ID`, `KEYCLOAK_CLIENT_SECRET`,
   `SESSION_DB_DSN`, `AUDIT_DB_DSN`, `APPAUTH_DB_DSN`, `CAPABILITY_DB_DSN`,
   and add `GOJA_AUTH_HOST_PUBLIC_BASE_URL` (for `RedirectURL`), `GOJA_AUTH_HOST_LISTEN`.
6. **RedirectURL derived from public base URL**, not the listen address:
   `${PUBLIC_BASE_URL}/auth/callback`.
7. **Remove demo seeding** (`o1`, `demo@example.test`). Seeding belongs in the
   db-bootstrap job or a first-run migration, not the binary.
8. **Routes from the generated seam (recommended).** Use `xgoja generate` to
   emit `internal/xgojaruntime` (example 21 style) and embed it, instead of
   `os.ReadFile("scripts/server.js")`. This is the "xgoja generated" contract
   the task asks for. See Decision D2 for the two-step path.

Pseudocode for the promoted `run`:

```go
func run(ctx context.Context, cfg Config) error {
    ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
    defer stop()

    // 1) Stores (Postgres in prod; ApplySchema on startup).
    appStores, err := newAppStore(ctx, cfg.AppDBDSN)          // from example 19
    capabilitySvc, _, err := newCapabilityService(ctx, cfg.CapabilityDBDSN)
    sessions, _, err := newSessionManager(cfg)                // AllowInsecureHTTP=false
    auditSink, _, err := newAuditSink(cfg.AuditDBDSN)

    // 2) OIDC (keycloakauth) against the real issuer.
    handlers, err := keycloakauth.New(ctx, keycloakauth.Config{
        IssuerURL:      cfg.Issuer,                 // https://auth.yolo.scapegoat.dev/realms/goja-auth-host
        ClientID:       cfg.ClientID,               // goja-auth-host-web
        ClientSecret:   cfg.ClientSecret,           // from VSO Secret
        RedirectURL:    cfg.PublicBaseURL + "/auth/callback",
        Scopes:         []string{"openid", "profile", "email"},
        AfterLoginURL:  "/",
        AfterLogoutURL: "/",
        SessionManager: sessions,
        UserNormalizer: appNormalizer(appStores),   // upsert app user from OIDC sub/email
    })

    // 3) Host: planned routes, deny-by-default authz.
    host := gojahttp.NewHost(gojahttp.HostOptions{
        Dev: false, RejectRawRoutes: true,
        Auth: gojahttp.AuthOptions{
            Authenticator: sessions, CSRF: sessions,
            Resources:  appauth.Resolver{Store: appStores.store},
            Authorizer: appauth.Authorizer{Memberships: appStores.store},
            Audit:      auditSink,
        },
    })

    // 4) Routes: prefer the generated xgojaruntime bundle (xgoja generate),
    //    fallback: go:embed scripts/server.js and vm.RunString.
    bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
        ConfigureServices: func(s *app.HostServices) { /* nothing; we wire host directly */ },
    })
    // mount bundle's express module onto `host`, then run declared routes.

    // 5) ServeMux: OIDC endpoints + host (planned) + probes.
    mux := http.NewServeMux()
    mux.Handle("GET /auth/login", handlers.LoginHandler())
    mux.Handle("GET /auth/callback", handlers.CallbackHandler())
    mux.Handle("POST /auth/logout", handlers.LogoutHandler())
    mux.Handle("GET /auth/session", sessionHandler(sessions))
    mux.Handle("GET /healthz", healthz())            // 200 ok
    mux.Handle("GET /readyz", readyz(stores...))     // ping DB
    mux.Handle("/", host)                             // planned routes + index

    srv := &http.Server{Addr: cfg.Listen, Handler: behindProxy(mux),
        ReadHeaderTimeout: 5*time.Second, ...}
    go srv.ListenAndServe()
    <-ctx.Done()
    return srv.Shutdown(shutdownCtx(ctx))
}
```

### 6.3 The container image: extend `Dockerfile`

The existing `Dockerfile` is a 3-stage build for `cmd/goja-repl`. Add an
auth-host target. Two viable shapes:

- **(Recommended) Multi-target single Dockerfile** using a shared Go builder
  stage and two final stages. Keeps one `docker build` context and one
  `.dockerignore`.
- **Separate `Dockerfile.auth-host`** if the auth host has no frontend (it
  doesn't), which is simpler and avoids the `node`/`pnpm` stages entirely.

Because the auth host has **no web frontend**, a dedicated `Dockerfile.auth-host`
is the cleanest. Sketch:

```dockerfile
# syntax=docker/dockerfile:1
FROM golang:1.26-bookworm AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Auth host needs CGO for sqlite fallback? We use Postgres in prod -> CGO off is fine,
# but the sqlite store (local dev) needs CGO. Keep CGO_ENABLED=1 to support both.
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/goja-auth-host ./cmd/goja-auth-host

FROM debian:12-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN groupadd --gid 65532 goja && useradd --uid 65532 --gid goja --shell /usr/sbin/nologin --create-home goja
WORKDIR /app
COPY --from=go-builder /app/bin/goja-auth-host /app/goja-auth-host
USER goja:goja
EXPOSE 8080
ENTRYPOINT ["/app/goja-auth-host"]
# Real flags/env come from the ConfigMap + Secret at runtime.
```

> The repo's existing image uses uid/gid `65532`; the `go-go-host` deployment
> uses `fsGroup: 10001`. Pick one and keep the Pod `securityContext.fsGroup`
> consistent with the image's gid (§10).

### 6.4 CI: extend `.github/workflows/publish-image.yaml`

Today the workflow builds/publishes one essay image and opens one GitOps PR.
Add a parallel `auth-host` job (or a matrix) that:

1. builds `./cmd/goja-auth-host`,
2. tags `ghcr.io/go-go-golems/goja-auth-host:sha-<short>`,
3. on `main`, runs `scripts/open_gitops_pr.py` against the new target in
   `deploy/gitops-targets.json`.

Minimal addition (sketch):

```yaml
  auth-host:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod, cache: true }
      - run: go test ./...
      - id: meta
        uses: docker/metadata-action@v6
        with:
          images: ghcr.io/${{ github.repository_owner }}/goja-auth-host
          tags: [ "type=sha,prefix=sha-,format=short", "type=raw,value=main,enable={{is_default_branch}}" ]
      - uses: docker/login-action@v4
        if: github.event_name != 'pull_request'
        with: { registry: ghcr.io, username: ${{ github.actor }}, password: ${{ secrets.GITHUB_TOKEN }} }
      - uses: docker/build-push-action@v7
        with: { context: ., file: ./Dockerfile.auth-host, push: ${{ github.event_name != 'pull_request' }},
                load: ${{ github.event_name == 'pull_request' }}, tags: ${{ steps.meta.outputs.tags }}, cache-from: type=gha, cache-to: type=gha,mode=max }
      - name: Open GitOps PR
        if: github.event_name != 'pull_request' && github.ref == 'refs/heads/main'
        env:
          GH_TOKEN: ${{ secrets.GITOPS_PR_TOKEN }}
          GHCR_IMAGE: ghcr.io/${{ github.repository_owner }}/goja-auth-host
        run: |
          python3 scripts/open_gitops_pr.py \
            --config deploy/gitops-targets.json \
            --target goja-auth-host-prod \
            --image "${GHCR_IMAGE}:sha-${GITHUB_SHA::7}" \
            --push --open-pr
```

> The repo currently uses the repo-local `scripts/open_gitops_pr.py` with a
> long-lived `GITOPS_PR_TOKEN` secret. The platform direction (see
> `infra-tooling/docs/platform/source-repo-to-gitops-pr.md`) is to move to the
> shared reusable workflow `go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml@<ref>`
> with Vault-backed GitHub Actions OIDC. That migration is **optional for this
> ticket** but recommended; record it in Decision D3.

### 6.5 The GitOps target: `deploy/gitops-targets.json`

Append a second target. Keep the existing (essay) entry; do not delete it.

```json
{
  "targets": [
    {
      "name": "goja-essay-prod",
      "gitops_repo": "wesen/2026-03-27--hetzner-k3s",
      "gitops_branch": "main",
      "manifest_path": "gitops/kustomize/goja-essay/deployment.yaml",
      "container_name": "goja-essay"
    },
    {
      "name": "goja-auth-host-prod",
      "gitops_repo": "wesen/2026-03-27--hetzner-k3s",
      "gitops_branch": "main",
      "manifest_path": "gitops/kustomize/goja-auth-host/deployment.yaml",
      "container_name": "goja-auth-host"
    }
  ]
}
```

Validate locally:

```bash
python3 ~/code/wesen/go-go-golems/infra-tooling/scripts/gitops/validate_gitops_targets.py deploy/gitops-targets.json
```

## 7. Decision records

### Decision D1: Confidential OIDC client + opaque server-side sessions

- **Context:** Example 19 uses a confidential Keycloak client with a server-side
  opaque session cookie. Tokens never reach the browser. The platform also has
  public SPA clients (go-go-host dashboard) and a CLI device client.
- **Options considered:**
  1. Confidential client + server-side sessions (example 19 model).
  2. Public client + browser-held tokens (SPA model).
- **Decision:** Use (1), the example-19 model.
- **Rationale:** The auth host is a server-rendered planned-route app, not a
  browser-token SPA. Server-side sessions give us revocation, CSRF binding,
  and keep refresh tokens off the client. This is also what the existing,
  tested code already does.
- **Consequences:** We must store the OIDC client secret in Vault (runtime
  secret) and run a Postgres-backed session store. Multi-instance rollout
  needs a shared transaction store for the callback (see Risks).
- **Status:** accepted

### Decision D2: Generated route seam now; generated OIDC builder later

- **Context:** The task asks for an "xgoja-generated" server. Example 21 is the
  generated seam but its README says OIDC/Postgres are follow-up work
  (`XGOJA-AUTH-KEYCLOAK-MFA`). Example 19 has complete OIDC + Postgres but is
  hand-wired.
- **Options considered:**
  1. Ship example-19 host as-is (hand-wired OIDC, loose `scripts/server.js`).
  2. Wait for `hostauth.ModeOIDC` builder, then ship example-21 seam only.
  3. **Hybrid:** use the generated `internal/xgojaruntime` package to embed and
     mount the JS route module (the "generated" contract), but wire OIDC +
     Postgres stores using example 19's proven `keycloakauth.New` path.
- **Decision:** (3) Hybrid.
- **Rationale:** Satisfies "xgoja-generated" now (routes come from an
  `xgoja generate` artifact), ships proven OIDC/Postgres, and defers only the
  cosmetic "everything flows through `hostauth` ModeOIDC" refactor to the MFA
  ticket.
- **Consequences:** Two wiring paths coexist briefly (host-direct OIDC vs
  `hostauth` config). The eventual migration to `hostauth.ModeOIDC` must keep
  the `keycloakauth.Config` shape identical so routes/normalizer don't change.
- **Status:** accepted

### Decision D3: Keep repo-local GitOps helper for now; migrate to shared workflow later

- **Context:** `publish-image.yaml` calls repo-local `scripts/open_gitops_pr.py`
  with `secrets.GITOPS_PR_TOKEN`. `infra-tooling` offers a shared reusable
  workflow + `open-gitops-pr` action with Vault-backed OIDC.
- **Options considered:**
  1. Keep repo-local helper for this ticket.
  2. Migrate to the shared workflow now.
- **Decision:** (1) now, (2) as a fast-follow.
- **Rationale:** Minimizes blast radius for the first auth-host rollout; the
  shared migration is orthogonal and already documented.
- **Consequences:** We rely on `GITOPS_PR_TOKEN` existing as a source-repo
  secret until the migration.
- **Status:** accepted

## 8. Key runtime flows

### 8.1 Browser login (OIDC Authorization Code + PKCE)

```text
Browser                Go host (goja-auth-host)            Keycloak
  │                       │                                  │
  │  GET /  (click Login) │                                  │
  ├──────────────────────▶│                                  │
  │  302 -> /auth/login   │                                  │
  │──────────────────────▶│  generate state+nonce+PKCE       │
  │  302 -> Keycloak      │  /auth?client_id=goja-auth-host- │
  │                       │       web&redirect_uri=.../auth/ │
  │                       │       callback&code_challenge=...│
  ├──────────────────────────────────────────────────────────▶│
  │  login form + creds  │                                  │
  │◀──────────────────────────────────────────────────────────┤
  │  302 -> /auth/callback?code=...&state=...                │
  │──────────────────────▶│  verify state/nonce              │
  │                       │  token exchange (code -> tokens) │
  │                       │  verify ID token (issuer, aud,   │
  │                       │   nonce, signature via JWKS)      │
  │                       │  UserNormalizer: upsert app user,│
  │                       │   derive UserSession             │
  │                       │  create opaque session (Postgres)│
  │  Set-Cookie: sid=...  │                                  │
  │  (Secure,HttpOnly,    │                                  │
  │   SameSite=Lax)       │                                  │
  │◀──────────────────────┤                                  │
  │                       │  tokens stay server-side         │
```

After this, every request carries only the opaque `sid` cookie. The Go host
resolves the session, CSRF-verifies unsafe methods, runs the `Authorizer`
against `appauth` resources/memberships, and writes an `audit` record.

### 8.2 Release/rollout flow (cross-repo)

```text
dev pushes to go-go-goja main
  -> GitHub Actions: go test, build cmd/goja-auth-host
  -> docker build+push ghcr.io/go-go-golems/goja-auth-host:sha-<short>
  -> scripts/open_gitops_pr.py patches
       gitops/kustomize/goja-auth-host/deployment.yaml image -> sha-<short>
  -> opens PR in wesen/2026-03-27--hetzner-k3s
operator merges PR
  -> Argo CD detects change on gitops/applications/goja-auth-host.yaml path
  -> Argo syncs the kustomize package (sync waves: ns -> secrets -> config -> deploy)
  -> VSO renders Secret from Vault, Traefik serves TLS, cert-manager issues cert
  -> Pod pulls image (image-pull secret), starts, /readyz -> 200
  -> https://goja-auth.yolo.scapegoat.dev/ live
```

## 9. Implementation plan (in-repo, phased)

Phases 1–4 are in-repo and within this ticket's modification scope. Phases 5–7
are the out-of-repo operator actions (§10–§12), listed for completeness and
approval.

### Phase 1 — Promote the host (go-go-goja)

1. Create `cmd/goja-auth-host/main.go` from example 19's `cmd/host/main.go`.
2. Apply the §6.2 productionization checklist (bind `:8080`, secure cookies,
   graceful shutdown, `/readyz`, config-from-env, derived RedirectURL, drop
   demo seed).
3. Add `cmd/goja-auth-host/config.go` for the Glazed/YAML config struct
   mirroring go-go-host's `config.yaml`.

**Validate:** `go build ./cmd/goja-auth-host && go test ./...`

### Phase 2 — Generated route seam (go-go-goja)

1. Add an `xgoja.yaml` (copy example 21) that emits
   `cmd/goja-auth-host/internal/xgojaruntime` from a `routes/` JS source
   (the planned routes from example 19's `scripts/server.js`, minus demo bits).
2. `make`-local or `go generate` to regenerate the package.
3. In `main.go`, import the generated package and mount its express module
   onto the `Host` (replacing `os.ReadFile`/`vm.RunString`).

**Validate:** `make -C examples/xgoja/21-generated-host-auth smoke` still passes
(regression); the new host serves `/healthz`, `/me` (401 unauth), and a planned
project route.

### Phase 3 — Image + CI (go-go-goja)

1. Add `Dockerfile.auth-host` (§6.3). Update `.dockerignore` to keep
   `ttmp/`, `examples/` dist, etc. excluded.
2. Extend `.github/workflows/publish-image.yaml` with the `auth-host` job (§6.4).
3. Append the `goja-auth-host-prod` target to `deploy/gitops-targets.json` (§6.5)
   and run the validator.

**Validate (local):**
```bash
docker build -f Dockerfile.auth-host -t goja-auth-host:dev .
docker run --rm -p 8080:8080 -e KEYCLOAK_ISSUER=... goja-auth-host:dev
curl -fsS http://127.0.0.1:8080/healthz
```

### Phase 4 — Repo-local smoke + docs (go-go-goja)

1. Add a `Makefile` target `auth-host-smoke` that builds the image and curls
   `/healthz` + `/readyz` + expects `401` on `/me`.
2. Point the repo README / `examples/xgoja/README.md` at the new production
   guide (this ticket).

### Phase 5–7 — Out-of-repo (approval-gated; see §10–§12)

5. Cluster repo: Kustomize package + Argo Application + Vault policies/roles.
6. Terraform repo: Keycloak realm + browser client (`k3s-beta`).
7. Vault: seed runtime + image-pull secrets; bootstrap k8s auth.

## 10. Out-of-repo spec: cluster repo (`2026-03-27--hetzner-k3s`)

> **Approval required.** These files live outside `./go-go-goja/` and are NOT
> modified by this ticket. They are specified here so an operator can create
> them verbatim.

Create a new package by copying `gitops/kustomize/go-go-host/` and substituting
`go-go-host -> goja-auth-host` plus the config block below.

### 10.1 `gitops/kustomize/goja-auth-host/` manifest deltas vs `go-go-host`

- **namespace.yaml** — `name: goja-auth-host`.
- **deployment.yaml** —
  - `image: ghcr.io/go-go-golems/goja-auth-host:sha-<initial-sha>`
  - `containerPort: 8080`, probes `/readyz` (readiness), `/healthz` (liveness)
  - `args: ["--config", "/etc/goja-auth-host/config.yaml"]`
  - env from `goja-auth-host-runtime` secret keys: `SESSION_DB_DSN`,
    `AUDIT_DB_DSN`, `APPAUTH_DB_DSN`, `CAPABILITY_DB_DSN`, `KEYCLOAK_CLIENT_SECRET`
  - `securityContext.fsGroup` matching the image gid (65532 if §6.3 image used).
- **configmap.yaml** —
  ```yaml
  config.yaml: |
    listenAddr: "0.0.0.0:8080"
    publicBaseUrl: "https://goja-auth.yolo.scapegoat.dev"
    oidcIssuer: "https://auth.yolo.scapegoat.dev/realms/goja-auth-host"
    oidcClientId: "goja-auth-host-web"
    oidcRedirectPath: "/auth/callback"
    oidcLogoutRedirectPath: "/"
    oidcScopes: ["openid","profile","email"]
    secureCookies: true
    logLevel: "info"
  ```
- **certificate.yaml + ingress.yaml** — host `goja-auth.yolo.scapegoat.dev`,
  ClusterIssuer `letsencrypt-prod-dns01-digitalocean`, ingressClassName `traefik`.
- **runtime-secret.yaml** — `VaultStaticSecret` path `apps/goja-auth-host/beta/runtime`.
- **image-pull-secret.yaml** — `VaultStaticSecret` path
  `apps/goja-auth-host/beta/image-pull`, type `kubernetes.io/dockerconfigjson`
  (only if the GHCR package is private; the go-go-golems org packages are
  public by default — confirm before adding).
- **postgres-admin-secret.yaml** — reuse `infra/postgres/cluster`.
- **db-bootstrap-** job/script — create role `goja_auth_host` + database
  `goja_auth_host`, reading app DB name/user/password from the runtime secret.
- **persistentvolumeclaim.yaml** — only if the host writes local files; the
  auth host is stateless aside from Postgres, so this can be **omitted**.
  (Decision: keep the app stateless; all persistence in Postgres.)

### 10.2 `gitops/applications/goja-auth-host.yaml`

Copy `go-go-host.yaml`, change `metadata.name`, namespace, `spec.source.path`
to `gitops/kustomize/goja-auth-host`, keep `spec.project: prod-apps`, keep
labels `scapegoat.dev/tier: app`, `has-database: true`, `has-ingress: true`.

### 10.3 Vault auth (cluster repo `vault/`)

- `vault/policies/kubernetes/goja-auth-host.hcl` — read `apps/goja-auth-host/beta/*`.
- `vault/policies/kubernetes/goja-auth-host-db-bootstrap.hcl` — read
  `infra/postgres/cluster` + `apps/goja-auth-host/beta/runtime`.
- `vault/roles/kubernetes/goja-auth-host.json` + `...-db-bootstrap.json`.
- Bootstrap: `./scripts/bootstrap-vault-kubernetes-auth.sh`.

## 11. Out-of-repo spec: terraform repo (`terraform`)

> **Approval required.**

```bash
cd /home/manuel/code/wesen/terraform
make scaffold-browser-app APP=goja-auth-host PUBLIC_APP_URL=https://goja-auth.yolo.scapegoat.dev
```

Then edit the generated `keycloak/apps/goja-auth-host/envs/k3s-beta/` to match
go-go-host's env shape:

- `realm_name = "goja-auth-host"`
- `dashboard_client_id = "goja-auth-host-web"`
- `public_app_url = "https://goja-auth.yolo.scapegoat.dev"`
- `valid_redirect_uris` includes `https://goja-auth.yolo.scapegoat.dev/auth/callback`
- `valid_post_logout_redirect_uris` includes `https://goja-auth.yolo.scapegoat.dev/` and `/*`
- `web_origins` includes `https://goja-auth.yolo.scapegoat.dev`
- Backend key `keycloak/apps/goja-auth-host/k3s-beta/terraform.tfstate`
- Provider: `keycloak_url = "https://auth.yolo.scapegoat.dev"`, admin creds from
  `kubectl -n keycloak get secret keycloak-bootstrap-admin`.

> The browser-client module is **confidential** (`access_type = CONFIDENTIAL`)
  and carries `client_secret`. That secret is the value the host reads as
  `KEYCLOAK_CLIENT_SECRET` via Vault. Apply, then seed the Vault runtime secret
  with it in the same operator session (see playbook).

Apply sequence (operator):
```bash
export AWS_PROFILE=manuel
export TF_VAR_keycloak_url=https://auth.yolo.scapegoat.dev
export TF_VAR_keycloak_username=...   # from keycloak-bootstrap-admin secret
export TF_VAR_keycloak_password=...
terraform -chdir=keycloak/apps/goja-auth-host/envs/k3s-beta init
terraform -chdir=keycloak/apps/goja-auth-host/envs/k3s-beta validate
terraform -chdir=keycloak/apps/goja-auth-host/envs/k3s-beta plan
terraform -chdir=keycloak/apps/goja-auth-host/envs/k3s-beta apply
```

## 12. Out-of-repo spec: Vault secrets

> **Approval required.**

Seed (operator, after terraform apply produces the client secret):

```bash
export VAULT_ADDR=https://vault.yolo.scapegoat.dev
vault login -method=oidc role=operators

# Runtime secret consumed by the VSO VaultStaticSecret at apps/goja-auth-host/beta/runtime
vault kv put kv/apps/goja-auth-host/beta/runtime \
  keycloak_client_id="goja-auth-host-web" \
  keycloak_client_secret="<from terraform output / state>" \
  session_db_dsn="postgres://goja_auth_host:<pw>@postgres.postgres.svc.cluster.local:5432/goja_auth_host?sslmode=disable" \
  appauth_db_dsn="<same db; or separate>" \
  audit_db_dsn="<same db; or separate>" \
  capability_db_dsn="<same db; or separate>" \
  database="goja_auth_host" \
  username="goja_auth_host" \
  password="<generated>"

# Optional, only if the GHCR image is private
vault kv put kv/apps/goja-auth-host/beta/image-pull \
  server="ghcr.io" username="go-go-golems-ci" password="..." auth="..."
```

Then bootstrap k8s auth (§10.3) and confirm a `Secret` appears in-namespace:

```bash
kubectl -n goja-auth-host get secret goja-auth-host-runtime -o yaml
```

## 13. Test strategy

### 13.1 Unit (in-repo, already run by CI)

- `go test ./pkg/gojahttp/...` covers route registry, enforcer, auth plan,
  middleware, planned dispatch.
- `examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py` is
  a full Keycloak+Postgres end-to-end smoke (login, CSRF, capability invite,
  SQL persistence). Reuse its shape for the promoted host.

### 13.2 Image smoke (CI / local)

```bash
docker run -d --rm --name a -p 127.0.0.1:8080:8080 goja-auth-host:dev
curl -fsS http://127.0.0.1:8080/healthz            # 200
curl -fsS http://127.0.0.1:8080/readyz             # 200 (DB not wired in local -> degrade gracefully)
curl -i    http://127.0.0.1:8080/me                # 401 (no session)
```

### 13.3 Cluster validation (after rollout)

```bash
kubectl -n argocd get application goja-auth-host -w     # Healthy Synced
kubectl -n goja-auth-host get pods,jobs,ingress,secret
curl -I https://goja-auth.yolo.scapegoat.dev/readyz     # HTTP/2 200
curl https://auth.yolo.scapegoat.dev/realms/goja-auth-host/.well-known/openid-configuration
# browser: / redirects to Keycloak login; after login /me returns the actor
```

## 14. Risks, alternatives, open questions

### Risks

- **Multi-instance callback.** Example 19's README flags this: with >1 replica,
  the OIDC callback state must be in a shared store (Postgres transaction
  store), not in-process memory. Mitigation: `keycloakauth.Config.TransactionStore`
  backed by Postgres; start with `replicas: 1` + `strategy: Recreate`.
- **Cookie `Secure` behind Traefik.** Must honour `X-Forwarded-Proto`. If the
  host marks cookies Secure but sees `http` internally, login "works" but the
  cookie is dropped. Mitigation: a small `behindProxy` middleware
  (`r = r.WithContext(...)` trusting forwarded proto) — verify in staging.
- **`local-path` PVC sync waves.** If we keep a PVC, put it in the same wave as
  the Deployment (playbook warns about `WaitForFirstConsumer` binding). We
  avoid this by being stateless (Decision in §10.1).
- **Public vs private GHCR.** If the image is private and the image-pull secret
  is missing, the Pod hits `ImagePullBackOff` even after a successful GitOps PR.
- **Stale essay target.** `deploy/gitops-targets.json` already references a
  non-existent `goja-essay` package; keep it but ensure the auth-host package
  exists before the first `main` run, or CI will open a PR against a missing
  path.

### Alternatives considered

- Deploy example 19 unmodified. Rejected: example-shaped, dev cookies, no
  probes, no graceful shutdown.
- Use the Coolify-hosted Keycloak (`auth.scapegoat.dev`) instead of the
  in-cluster one. Rejected for K3s apps; the platform standard is
  `auth.yolo.scapegoat.dev` with a `k3s-beta`/`k3s-parallel` env.

### Open questions (for the operator / user)

1. Confirm the desired public hostname (`goja-auth.yolo.scapegoat.dev`) and
   whether the GHCR image should be private (affects §10.1 image-pull + §12).
2. Confirm whether the four stores (session/audit/appauth/capability) share one
   Postgres database (`goja_auth_host`) or should be split.
3. Confirm whether to migrate the publish workflow to the shared
   `infra-tooling` reusable workflow + Vault OIDC now or as a fast-follow (D3).
4. Approval to create the cluster-repo, terraform-repo, and Vault changes
   described in §10–§12.

## 15. References

### This repo (go-go-goja)

- `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go` — host to promote.
- `examples/xgoja/19-express-keycloak-auth-host/scripts/server.js` — planned routes.
- `examples/xgoja/19-express-keycloak-auth-host/README.md` — production notes.
- `examples/xgoja/21-generated-host-auth/xgoja.yaml`, `cmd/host/main.go` — generated seam template.
- `pkg/xgoja/hostauth/config.go` — host-owned auth config.
- `pkg/gojahttp/auth/keycloakauth/keycloakauth.go` — `Config`, `OIDCClaims`, `New`.
- `pkg/gojahttp/host.go` — `NewHost`, `HostOptions`, `AuthOptions`.
- `Dockerfile`, `.dockerignore`, `Makefile`, `.goreleaser.yaml` — current build.
- `.github/workflows/publish-image.yaml`, `scripts/open_gitops_pr.py`, `deploy/gitops-targets.json` — current release chain.

### Cluster repo (`2026-03-27--hetzner-k3s`)

- `gitops/kustomize/go-go-host/` — copy this package.
- `gitops/applications/go-go-host.yaml` — copy this Argo Application.
- `docs/app-runtime-secrets-and-identity-provisioning-playbook.md` — operator sequence.
- `docs/app-deployment-pipeline.md` — layout rules.

### Terraform repo (`terraform`)

- `keycloak/apps/go-go-host/envs/k3s-beta/` — copy this env.
- `keycloak/modules/browser-client/main.tf` — confidential client module.
- `docs/shared-keycloak-platform-playbook.md` — scaffolder + apply rules.

### infra-tooling (`go-go-golems/infra-tooling`)

- `docs/platform/source-repo-to-gitops-pr.md` — the cross-repo contract.
- `.github/workflows/publish-ghcr-image.yml`, `actions/open-gitops-pr/` — shared reusable pieces.
- `templates/github/publish-image-ghcr.template.yml`, `examples/platform/image-gitops-targets.example.json`.

### Sibling tickets (go-go-goja ttmp)

- `XGOJA-AUTH-STORES` — durable stores/migrations.
- `XGOJA-AUTH-KEYCLOAK-MFA` — Keycloak hardening, OIDC transaction store, MFA.
- `XGOJA-AUTH-PROD-DOCS` — production guide prose + policy-adapter evaluation.
