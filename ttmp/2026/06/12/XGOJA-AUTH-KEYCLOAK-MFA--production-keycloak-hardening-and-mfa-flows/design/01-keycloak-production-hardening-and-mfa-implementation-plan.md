---
Title: Keycloak production hardening and MFA implementation plan
Ticket: XGOJA-AUTH-KEYCLOAK-MFA
Status: active
Topics:
    - goja
    - http
    - security
    - keycloak
    - oidc
    - architecture
    - postgres
    - gitops
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: Current Keycloak host example to harden and deploy temporarily from examples/ rather than promoting to cmd/goja-auth-host.
    - Path: examples/xgoja/19-express-keycloak-auth-host/scripts/server.js
      Note: JavaScript planned-route script copied into the temporary deployment image.
    - Path: pkg/doc/29-express-auth-user-guide.md
      Note: Documents planned auth and .mfaFresh route declarations.
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: OIDC login/callback/logout and transaction store seam.
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: App session authentication, secure-cookie settings, and MFA freshness enforcement.
ExternalSources: []
Summary: V2 plan for temporarily deploying the Keycloak auth example from examples/, backed by Postgres and real Keycloak, while keeping the long-term production hardening/MFA roadmap intact.
LastUpdated: 2026-06-16T19:05:00-04:00
WhatFor: Use when hardening keycloakauth/sessionauth for production browser login, deploying the temporary yolo demo, and implementing host-owned MFA freshness updates.
WhenToUse: Before changing Keycloak handler behavior, OIDC transaction handling, logout, secure cookie deployment, Postgres stores, Argo CD demo registration, or MFA challenge/session update flows.
---

# Keycloak production hardening and MFA implementation plan

## Executive summary

This document is the **v2** plan for the Keycloak/OIDC auth host work. It keeps
the original long-term hardening/MFA roadmap, but updates the near-term deployment
strategy based on the current decision: the yolo deployment is temporary and will
likely be removed after it has served its purpose.

The updated near-term recommendation is:

1. **Do not promote the example into `cmd/goja-auth-host` yet.** Build and deploy
   from `examples/xgoja/19-express-keycloak-auth-host/cmd/host` for the temporary
   demo.
2. **Use real Postgres for the deployed demo.** The app should run with one
   app-specific database/user and point the session, audit, appauth, and
   capability DSNs at that database.
3. **Patch the example with the missing deploy flags.** The current example is
   localhost-oriented. It needs flags/env for public base URL / redirect URL,
   listen address, secure-vs-insecure cookies, and post-login/logout paths.
4. **Register the Argo CD Application under `demo-apps`.** Because the
   AppProject has an explicit namespace allowlist, the new namespace must also be
   added to `gitops/projects/demo-apps.yaml`.

The original architectural boundary remains correct. `keycloakauth` verifies
Keycloak/OIDC and creates an opaque app session. `sessionauth` authenticates
planned routes and enforces MFA freshness against `Session.MFAAt`. What is still
missing for long-term production is durable OIDC transaction storage, logout
hardening, and host-owned MFA challenge/verification endpoints that refresh
`MFAAt`.

## V2 deployment decision: deploy from `examples/`, not `cmd/`

### Decision

For this temporary yolo deployment, build the image from:

```bash
go build -o /app/goja-auth-host ./examples/xgoja/19-express-keycloak-auth-host/cmd/host
```

Do **not** create `cmd/goja-auth-host` yet.

### Rationale

- The deployment is explicitly temporary and should be easy to delete.
- Example 19 already contains the desired OIDC + Postgres-store integration seam.
- Promoting into `cmd/` implies a more stable product surface, CLI contract, and
  documentation burden than this short-lived deployment needs.
- A future long-lived host can still be promoted after the demo validates the
  auth flow.

### Consequences

- The temporary Dockerfile must copy the example JS route script into the image
  and pass `--script /app/server.js`.
- The image/workflow should be named as a temporary/demo host, not as the final
  product command.
- Any production hardening learned here should be ported into the long-term host
  later rather than treating the example path as permanent.

## Scope

This ticket covers roadmap items 2 and 3 from the auth follow-up list:

1. Production Keycloak host hardening.
2. MFA story beyond enforcement: challenge, verification, session update, and docs.
3. **V2 near-term deployment plan:** temporary example-based yolo deployment with
   Postgres, real Keycloak, and Argo CD `demo-apps` registration.

It depends on durable auth stores. For the temporary demo, we want Postgres now;
for long-term production, we still need a durable OIDC transaction store.

## Current-state evidence

Relevant files:

```text
pkg/gojahttp/auth/keycloakauth/keycloakauth.go
pkg/gojahttp/auth/keycloakauth/README.md
pkg/gojahttp/auth/sessionauth/sessionauth.go
pkg/gojahttp/auth/sessionauth/sessionauth_test.go
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml
examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py
examples/xgoja/19-express-keycloak-auth-host/scripts/server.js
pkg/doc/29-express-auth-user-guide.md
```

Current behavior:

- Keycloak login uses OIDC Authorization Code + PKCE.
- Callback verifies state, code, ID token, and nonce.
- `UserNormalizer` maps Keycloak claims into an app session projection.
- Browser receives an app session cookie, not IdP tokens.
- `sessionauth.Manager.Authenticate` enforces `SecuritySpec.MFAFreshWithin`
  against `Session.MFAAt`.
- Example 19 already has optional Postgres DSNs for sessions, audit records,
  appauth users/resources, and capability tokens.
- Example 19 still has localhost-oriented defaults and always creates the session
  manager with `AllowInsecureHTTP: true`, which is not acceptable behind HTTPS.
- `keycloakauth.TransactionStore` still defaults to in-memory storage if no
  store is supplied.

## Required example-host Glazed command/config updates

Before deploying the example, patch
`examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go` with deploy-safe
configuration. Use **Glazed** for all flags, environment variables, and config
file fields. Do not add another raw `flag`-based command surface.

The temporary host should become a small Glazed program even though it remains
under `examples/`. This keeps the example aligned with the rest of the repo and
lets `glazed-lint` catch command/field/schema mistakes in `make lint`,
pre-commit, and pre-push.

Minimum settings:

| Flag | Env | Purpose | Demo default |
| --- | --- | --- | --- |
| `--listen` | `LISTEN_ADDR` | Bind address in the pod. | `:8080` |
| `--script` | `SCRIPT_PATH` | JS route script path copied into image. | `/app/server.js` |
| `--issuer` | `KEYCLOAK_ISSUER` | Real Keycloak realm issuer. | none; required in deployment |
| `--client-id` | `KEYCLOAK_CLIENT_ID` | Keycloak confidential client ID. | `goja-auth-host-demo` |
| `--client-secret` | `KEYCLOAK_CLIENT_SECRET` | Keycloak confidential client secret. | from Kubernetes Secret |
| `--public-base-url` | `PUBLIC_BASE_URL` | External HTTPS origin used to derive callback URL. | `https://goja-auth.yolo.scapegoat.dev` |
| `--redirect-url` | `KEYCLOAK_REDIRECT_URL` | Explicit override for callback URL. | optional; default `<public-base-url>/auth/callback` |
| `--after-login-url` | `AFTER_LOGIN_URL` | Local path after successful login. | `/` |
| `--after-logout-url` | `AFTER_LOGOUT_URL` | Local path after logout. | `/` |
| `--allow-insecure-http` | `ALLOW_INSECURE_HTTP` | Use insecure dev cookie settings for local HTTP only. | `false` in cluster |
| `--session-db-dsn` | `SESSION_DB_DSN` | Postgres DSN for app sessions. | from DB Secret |
| `--audit-db-dsn` | `AUDIT_DB_DSN` | Postgres DSN for audit records. | from DB Secret |
| `--app-db-dsn` | `APPAUTH_DB_DSN` | Postgres DSN for appauth users/resources. | from DB Secret |
| `--capability-db-dsn` | `CAPABILITY_DB_DSN` | Postgres DSN for capability tokens. | from DB Secret |

Implementation notes:

- `RedirectURL` passed to `keycloakauth.New` must not be derived from
  `cfg.Listen`. Behind ingress, `:8080` is not the browser-visible origin.
- Model `public-base-url` as the normal operator-facing setting and
  `redirect-url` as an explicit advanced override.
- Use `--redirect-url` if set; otherwise require `--public-base-url` and derive
  `<public-base-url>/auth/callback`.
- `sessionauth.New(sessionauth.Config{AllowInsecureHTTP: ...})` should receive
  the Glazed setting value. In cluster, `AllowInsecureHTTP=false` so the secure
  `__Host-app` cookie is used.
- Keep `ALLOW_INSECURE_HTTP=true` only for Docker Compose / localhost smoke.
- Keep replicas at `1` until a durable/shared OIDC transaction store exists.

### Glazed command shape

Turn the example host into a Glazed command, not a hand-written `flag` program.
The command can still live in `examples/xgoja/19-express-keycloak-auth-host`; it
just uses the same command-description and settings decoding pattern as
`goja-repl` and `xgoja`.

Recommended layout:

```text
examples/xgoja/19-express-keycloak-auth-host/cmd/host/
  main.go          # root command + logging/help wiring
  serve.go         # Glazed serve command + settings struct
```

Recommended settings struct:

```go
type serveSettings struct {
    Listen          string `glazed:"listen"`
    Script          string `glazed:"script"`
    Issuer          string `glazed:"issuer"`
    ClientID        string `glazed:"client-id"`
    ClientSecret    string `glazed:"client-secret"`
    PublicBaseURL   string `glazed:"public-base-url"`
    RedirectURL     string `glazed:"redirect-url"`
    AfterLoginURL   string `glazed:"after-login-url"`
    AfterLogoutURL  string `glazed:"after-logout-url"`
    AllowInsecureHTTP bool `glazed:"allow-insecure-http"`
    SessionDBDSN    string `glazed:"session-db-dsn"`
    AuditDBDSN      string `glazed:"audit-db-dsn"`
    AppDBDSN        string `glazed:"app-db-dsn"`
    CapabilityDBDSN string `glazed:"capability-db-dsn"`
}
```

Recommended fields:

```go
cmds.WithFlags(
    fields.New("listen", fields.TypeString,
        fields.WithDefault(envOr("LISTEN_ADDR", ":8080")),
        fields.WithHelp("Listen address for the in-pod HTTP server")),
    fields.New("script", fields.TypeString,
        fields.WithDefault(envOr("SCRIPT_PATH", "/app/server.js")),
        fields.WithHelp("JavaScript route script to load")),
    fields.New("issuer", fields.TypeString,
        fields.WithDefault(os.Getenv("KEYCLOAK_ISSUER")),
        fields.WithHelp("OIDC issuer URL for the Keycloak realm")),
    fields.New("client-id", fields.TypeString,
        fields.WithDefault(envOr("KEYCLOAK_CLIENT_ID", "goja-auth-host-demo")),
        fields.WithHelp("OIDC client id")),
    fields.New("client-secret", fields.TypeString,
        fields.WithDefault(os.Getenv("KEYCLOAK_CLIENT_SECRET")),
        fields.WithHelp("OIDC client secret for the confidential client")),
    fields.New("public-base-url", fields.TypeString,
        fields.WithDefault(os.Getenv("PUBLIC_BASE_URL")),
        fields.WithHelp("External HTTPS base URL, for example https://goja-auth.yolo.scapegoat.dev")),
    fields.New("redirect-url", fields.TypeString,
        fields.WithDefault(os.Getenv("KEYCLOAK_REDIRECT_URL")),
        fields.WithHelp("Explicit OIDC callback URL; defaults to <public-base-url>/auth/callback")),
    fields.New("after-login-url", fields.TypeString,
        fields.WithDefault(envOr("AFTER_LOGIN_URL", "/")),
        fields.WithHelp("Local path to redirect to after successful login")),
    fields.New("after-logout-url", fields.TypeString,
        fields.WithDefault(envOr("AFTER_LOGOUT_URL", "/")),
        fields.WithHelp("Local path to redirect to after logout")),
    fields.New("allow-insecure-http", fields.TypeBool,
        fields.WithDefault(envBool("ALLOW_INSECURE_HTTP", false)),
        fields.WithHelp("Use insecure localhost cookie settings; must be false behind HTTPS ingress")),
    fields.New("session-db-dsn", fields.TypeString,
        fields.WithDefault(os.Getenv("SESSION_DB_DSN")),
        fields.WithHelp("Postgres DSN for server-side app sessions")),
    fields.New("audit-db-dsn", fields.TypeString,
        fields.WithDefault(os.Getenv("AUDIT_DB_DSN")),
        fields.WithHelp("Postgres DSN for audit records")),
    fields.New("app-db-dsn", fields.TypeString,
        fields.WithDefault(os.Getenv("APPAUTH_DB_DSN")),
        fields.WithHelp("Postgres DSN for appauth users, memberships, and resources")),
    fields.New("capability-db-dsn", fields.TypeString,
        fields.WithDefault(os.Getenv("CAPABILITY_DB_DSN")),
        fields.WithHelp("Postgres DSN for capability tokens")),
)
```

The above uses environment-backed defaults because this is a small temporary
host. If the demo grows, add Glazed config-file sources next so the same settings
can be supplied by YAML as well as flags/env.

### Handling `public-base-url`

`public-base-url` is not an OIDC protocol field. It is deployment topology:
"what HTTPS origin does the browser see?" Treat it as a **host setting** used to
compute the OIDC callback URL.

Validation rules:

```go
func resolveRedirectURL(settings serveSettings) (string, error) {
    if strings.TrimSpace(settings.RedirectURL) != "" {
        return settings.RedirectURL, requireHTTPSUnlessInsecure(settings.RedirectURL, settings.AllowInsecureHTTP)
    }
    publicBase := strings.TrimRight(strings.TrimSpace(settings.PublicBaseURL), "/")
    if publicBase == "" {
        return "", errors.New("public-base-url or redirect-url is required")
    }
    if err := requireHTTPSUnlessInsecure(publicBase, settings.AllowInsecureHTTP); err != nil {
        return "", err
    }
    return publicBase + "/auth/callback", nil
}
```

Rules:

- In cluster, require `https://`.
- Only allow `http://` when `allow-insecure-http=true` for localhost/Compose.
- Normalize by trimming a trailing slash.
- Do not infer from `listen`; bind address and public origin are separate
  concepts.

### Runtime pseudocode

```go
settings := serveSettings{}
if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
    return err
}
redirectURL, err := resolveRedirectURL(settings)
if err != nil {
    return err
}

sessions, err := sessionauth.New(sessionauth.Config{
    Store:             store,
    AllowInsecureHTTP: settings.AllowInsecureHTTP,
})

keycloakHandlers, err := keycloakauth.New(ctx, keycloakauth.Config{
    IssuerURL:      settings.Issuer,
    ClientID:       settings.ClientID,
    ClientSecret:   settings.ClientSecret,
    RedirectURL:    redirectURL,
    AfterLoginURL:  settings.AfterLoginURL,
    AfterLogoutURL: settings.AfterLogoutURL,
    SessionManager: sessions,
    UserNormalizer: normalizer,
})
```

### Glazed lint/tooling requirement

`glazed-lint` should be part of the repository toolchain and run through
`make lint`; Lefthook already runs `make lint` on pre-commit/pre-push. The repo
should track the analyzer as a Go tool dependency so the version is explicit in
`go.mod`, and the Makefile should install/use that tool instead of an ad-hoc
untracked binary.

## Postgres requirement for the temporary deployment

We want Postgres for the temporary deployment. Use one app-specific database and
one app-specific login role rather than four separate databases.

Recommended names:

```text
namespace: goja-auth-host-demo
database:  goja_auth_host_demo
role/user: goja_auth_host_demo_app
```

Point all four runtime DSNs at the same database/role:

```text
SESSION_DB_DSN=postgres://goja_auth_host_demo_app:<password>@postgres.postgres.svc.cluster.local:5432/goja_auth_host_demo?sslmode=disable
AUDIT_DB_DSN=postgres://goja_auth_host_demo_app:<password>@postgres.postgres.svc.cluster.local:5432/goja_auth_host_demo?sslmode=disable
APPAUTH_DB_DSN=postgres://goja_auth_host_demo_app:<password>@postgres.postgres.svc.cluster.local:5432/goja_auth_host_demo?sslmode=disable
CAPABILITY_DB_DSN=postgres://goja_auth_host_demo_app:<password>@postgres.postgres.svc.cluster.local:5432/goja_auth_host_demo?sslmode=disable
```

This matches the example's current store split while keeping database operations
simple for a disposable demo. Each store calls `ApplySchema`, so separate schemas
are not required for this short-lived deployment.

### How to provision DB/user

Use the cluster's existing Vault-backed Postgres bootstrap Job pattern:

1. Store generated app DB credentials in Vault, for example:

   ```text
   kv/apps/goja-auth-host-demo/demo/database
   ```

2. Render the app DB secret into the app namespace using Vault Secrets Operator.
3. Render the shared Postgres admin/bootstrap secret only into the bootstrap Job.
4. Run an idempotent bootstrap Job that creates/updates:

   ```sql
   CREATE ROLE goja_auth_host_demo_app LOGIN PASSWORD '<generated>';
   CREATE DATABASE goja_auth_host_demo OWNER goja_auth_host_demo_app;
   GRANT ALL PRIVILEGES ON DATABASE goja_auth_host_demo TO goja_auth_host_demo_app;
   ```

5. The long-running Deployment only receives the app runtime DB secret, not the
   shared Postgres admin credential.

Idempotent SQL shape:

```sql
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'app_user') THEN
    EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', :'app_user', :'app_password');
  ELSE
    EXECUTE format('ALTER ROLE %I WITH LOGIN PASSWORD %L', :'app_user', :'app_password');
  END IF;
END
$$;

SELECT 'CREATE DATABASE ' || quote_ident(:'app_db') || ' OWNER ' || quote_ident(:'app_user')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'app_db')\gexec
```

For the temporary demo, this is enough. If the app becomes long-lived, split
privileges more carefully and consider separate schemas or a migration tool.

## Keycloak client requirement

Create a dedicated Keycloak confidential client for the demo.

Recommended values:

```text
realm:        choose the real yolo realm for this demo
client_id:    goja-auth-host-demo
client_type:  confidential
redirect_uri: https://goja-auth.yolo.scapegoat.dev/auth/callback
web_origins:  https://goja-auth.yolo.scapegoat.dev
```

Store the client secret in Vault and sync it into the namespace as
`KEYCLOAK_CLIENT_SECRET`. Terraform is preferred for repeatability; manual
Keycloak setup is acceptable only if the ticket records the exact client settings
and a deletion step.

## Argo CD demo-app registration

Create a Kustomize package and Application in the cluster repo:

```text
gitops/kustomize/goja-auth-host-demo/
gitops/applications/goja-auth-host-demo.yaml
```

Application skeleton:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: goja-auth-host-demo
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
  labels:
    app.kubernetes.io/name: "goja-auth-host-demo"
    app.kubernetes.io/part-of: "demo-apps"
    app.kubernetes.io/managed-by: "argocd"
    scapegoat.dev/tier: "demo"
    scapegoat.dev/source-type: "kustomize"
    scapegoat.dev/has-database: "true"
    scapegoat.dev/has-persistent-storage: "false"
    scapegoat.dev/has-ingress: "true"
    scapegoat.dev/database-type: "postgres"
  annotations:
    scapegoat.dev/description: "Temporary go-go-goja Keycloak auth host demo"
spec:
  project: demo-apps
  destination:
    server: https://kubernetes.default.svc
    namespace: goja-auth-host-demo
  source:
    repoURL: https://github.com/wesen/2026-03-27--hetzner-k3s.git
    targetRevision: main
    path: gitops/kustomize/goja-auth-host-demo
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

Also update `gitops/projects/demo-apps.yaml` because `demo-apps` has explicit
namespace destinations:

```yaml
spec:
  destinations:
    - server: https://kubernetes.default.svc
      namespace: goja-auth-host-demo
```

Without this AppProject update, Argo CD will reject the new Application even if
the Application manifest is otherwise correct.

## Temporary Kustomize package shape

The temporary package should include:

```text
namespace.yaml
serviceaccount.yaml
vault-connection.yaml
vault-auth.yaml
runtime-secret.yaml              # VSO: Keycloak + app DB runtime secrets
postgres-admin-secret.yaml       # VSO: bootstrap Job only
db-bootstrap-serviceaccount.yaml
db-bootstrap-vault-auth.yaml
db-bootstrap-script-configmap.yaml
db-bootstrap-job.yaml
deployment.yaml
service.yaml
ingress.yaml
kustomization.yaml
```

Deployment requirements:

- `replicas: 1` until the OIDC transaction store is durable/shared.
- `--listen :8080`.
- `--script /app/server.js`.
- `--public-base-url https://goja-auth.yolo.scapegoat.dev`.
- `--allow-insecure-http=false` (or omit if false by default).
- pass the four DSNs from the app runtime DB Secret.
- pass Keycloak issuer/client/secret from Secret/ConfigMap.
- readiness/liveness can hit `/healthz` if the JS script provides it; otherwise
  add a host-owned `/healthz` before deploying.

## Production Keycloak hardening

### OIDC transaction store

`keycloakauth.TransactionStore` currently has an in-memory default. Production
multi-instance deployments need a durable/shared transaction store so callbacks
work across replicas and login state survives restarts within the transaction
TTL.

For the temporary demo:

- keep replicas at `1`;
- accept in-memory OIDC transactions temporarily;
- still use Postgres for app sessions/audit/appauth/capabilities;
- keep a follow-up to add a SQL-backed `TransactionStore` before multi-replica
  or long-lived production use.

Implementation requirements for the future SQL transaction store:

- store state, nonce, PKCE verifier, redirect URL, and created timestamp;
- expire transactions after a short TTL;
- `Take` must be one-time;
- state/nonce/verifier must never be logged;
- integration tests should prove replay protection.

### Secure deployment settings

The example uses local HTTP settings. Production docs and code should make the
secure path obvious:

- HTTPS-only redirect URLs;
- secure cookies (`AllowInsecureHTTP=false` in cluster);
- appropriate `SameSite` mode;
- reverse proxy headers and trusted proxy guidance;
- Keycloak issuer URL consistency;
- redirect URI and web origin validation;
- secret handling for confidential clients.

### Logout and session lifecycle

The current logout clears/revokes the app session. Production hardening should
decide whether and how to support Keycloak end-session behavior.

Open design points:

- local app logout only vs app logout plus Keycloak end-session redirect;
- back-channel logout or front-channel logout support;
- whether refresh tokens are ever retained server-side;
- how logout audit events are recorded.

Initial direction: keep app-session logout mandatory and make Keycloak
end-session optional/configured. Do not expose IdP tokens to browser JavaScript.

## MFA flow design

### What exists today

The route plan can declare MFA freshness:

```javascript
app.post("/billing/payment-methods")
  .auth(express.user().required().mfaFresh("10m"))
  .csrf()
  .allow("billing.payment_method.update")
  .handle(handler)
```

The Go route plan carries that as `SecuritySpec.MFAFreshWithin`, and
`sessionauth.Manager.Authenticate` rejects sessions whose `MFAAt` is nil or
stale.

### What is missing

A host needs a way to set `Session.MFAAt` after the user completes a second
factor. That should be host-owned, not Express-owned, because MFA method choice
belongs to the application/IdP/deployment.

Possible MFA paths:

1. **Keycloak-required MFA during login.** Keycloak enforces MFA before the
   initial app session is created. The normalizer or handler marks `MFAAt` when
   verified claims indicate MFA was completed.
2. **App step-up MFA.** A planned route returns 401 for stale MFA. The browser
   visits an app-owned MFA challenge endpoint. Successful verification updates
   `Session.MFAAt`.
3. **Keycloak step-up prompt.** The app redirects to Keycloak with
   prompt/acr/max_age parameters, then updates `MFAAt` on return if the ID token
   satisfies the requested assurance.

Initial recommendation: support explicit app-owned MFA update hooks first, then
document how to integrate Keycloak-required MFA or step-up flows.

### Proposed sessionauth additions

Add a narrow session update API rather than exposing store internals:

```go
func (m *Manager) MarkMFAComplete(ctx context.Context, r *http.Request, at time.Time) error
```

This should:

- load and validate the current session;
- update `MFAAt` atomically;
- preserve revocation and expiry checks;
- optionally rotate the session ID after MFA completion if desired;
- emit audit events through a host-level caller, not directly from `sessionauth`
  unless an audit dependency is explicitly added.

This likely requires extending `sessionauth.Store` with an MFA update method or
adding a narrower optional interface.

### Error behavior

MFA freshness denial currently maps to `gojahttp.ErrUnauthenticated`, which means
planned routes return 401. That is correct if the client must complete additional
authentication. A future response body/header may distinguish `mfa_required`
from ordinary unauthenticated requests, but it should not leak sensitive details
by default.

## Implementation phases

### Phase 0 — Temporary yolo demo from example 19

- Patch example 19 with the required deploy flags.
- Build a temporary image from `examples/xgoja/19-express-keycloak-auth-host/cmd/host`.
- Copy `scripts/server.js` to `/app/server.js`.
- Provision one Postgres database/user via the Vault-backed bootstrap Job.
- Create a Keycloak confidential client and sync its secret through Vault/VSO.
- Register Argo CD Application under `demo-apps` and add the namespace to
  `gitops/projects/demo-apps.yaml`.
- Deploy with `replicas: 1`.

### Phase 1 — Production Keycloak settings and docs

Document secure Keycloak client settings, redirect URI rules, HTTPS/cookie
requirements, and local-vs-production differences.

### Phase 2 — Durable OIDC transaction store

Add a SQL-backed `keycloakauth.TransactionStore` with one-time `Take`, TTL
cleanup, and replay tests.

### Phase 3 — Logout hardening

Add optional Keycloak end-session support if the design confirms it is needed.
Preserve app-session revocation as the mandatory logout behavior.

### Phase 4 — MFA session update primitive

Add a `sessionauth` API and store support for updating `Session.MFAAt`. Include
tests for updating, stale route rejection before update, and successful route
authentication after update.

### Phase 5 — MFA example flow

Add a small app-owned MFA example endpoint to the dev-auth or Keycloak example.
It can use a deliberately simple demo factor, but it must demonstrate the
important runtime behavior: route denied before step-up, MFA completion updates
the session, route allowed after step-up.

### Phase 6 — Production smoke and docs

Extend the Keycloak smoke or add a dedicated smoke for MFA freshness and
production settings.

## Testing strategy

Required tests should include:

```bash
go test ./pkg/gojahttp/auth/keycloakauth ./pkg/gojahttp/auth/sessionauth -count=1
go test ./examples/xgoja/19-express-keycloak-auth-host/cmd/host -count=1
make -C examples/xgoja/19-express-keycloak-auth-host smoke
```

Deployment validation for the temporary demo:

```bash
curl -fsS https://goja-auth.yolo.scapegoat.dev/healthz
curl -I https://goja-auth.yolo.scapegoat.dev/auth/login
kubectl -n argocd get application goja-auth-host-demo
kubectl -n goja-auth-host-demo get pods,svc,ingress,job,secret
kubectl -n goja-auth-host-demo logs deploy/goja-auth-host-demo --tail=100
```

Additional tests should cover:

- OIDC transaction replay rejection;
- expired OIDC transaction rejection;
- production cookie settings (`Secure`, `__Host-` cookie, HTTPS);
- `MFAAt` update persistence;
- planned route denial with stale MFA;
- planned route success after MFA update.

## Body/schema validation and auth boundary

Body/schema validation is security-relevant, but it is not authentication. It
belongs to request validation and authorization safety.

It matters for auth when authorization depends on request body fields. Examples:

- a body contains `tenantId`, `role`, `ownerId`, `resourceId`, or `permissions`;
- a route updates membership roles;
- a route creates an invite capability;
- a route performs partial updates where omitted vs null fields have different
  meaning.

Without schema validation, the authorizer may make decisions using untrusted or
ambiguous data, or the handler may mutate fields that were not intended to be
client-controlled. The safest long-term route-plan order is:

```text
authenticate actor
resolve path/query-bound resources
validate body schema and normalize typed input
authorize action against actor + resource + normalized input
verify CSRF for unsafe browser/session requests
run handler
record audit outcomes
```

The current auth work can proceed without body schemas because route
identity/resource/action enforcement already uses path/query bindings and
host-owned resource resolution. Body schemas should be a separate follow-up
ticket after persistent stores and Keycloak/MFA hardening, because it is a
broader request-validation layer, not the next blocker for authentication.

## References

- `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`
- `pkg/gojahttp/auth/sessionauth/sessionauth.go`
- `examples/xgoja/19-express-keycloak-auth-host`
- `pkg/doc/29-express-auth-user-guide.md`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/vault-backed-postgres-bootstrap-job-pattern.md`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/projects/demo-apps.yaml`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/goja-kanban.yaml`
