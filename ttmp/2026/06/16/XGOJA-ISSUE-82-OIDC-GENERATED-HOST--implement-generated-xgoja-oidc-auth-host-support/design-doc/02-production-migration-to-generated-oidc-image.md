---
Title: Production migration to generated OIDC image
Slug: production-migration-to-generated-oidc-image
Short: Guide for replacing the live example-19 auth host image with the generated example-21 OIDC xgoja serve image.
Topics:
- xgoja
- auth
- oidc
- deployment
- gitops
DocType: design-doc
Status: active
Intent: long-term
Ticket: XGOJA-ISSUE-82-OIDC-GENERATED-HOST
Created: 2026-06-16
Updated: 2026-06-16
---

# Production migration to generated OIDC image

## Executive summary

The live `goja-auth-host-demo` deployment currently runs the hand-written example-19 Keycloak host. The generated OIDC implementation makes it possible to replace that image with a binary built from `examples/xgoja/21-generated-host-auth/xgoja.yaml` while keeping the existing Keycloak realm/client, Vault runtime secret, Postgres database, Service, Ingress, and Argo CD application.

The replacement is not just an image tag swap. The generated binary has a different command and configuration contract: it runs `serve sites demo`, reads `--http-*` and `--auth-*` Glazed fields, and can receive those fields from environment variables using the configured `XGOJA_OIDC_DEMO` env prefix. Production should pass configuration through Kubernetes env vars sourced from Vault, not bake values or secrets into the Docker image.

## Current production contract

Current source image:

```text
Dockerfile.auth-host
  -> go build ./examples/xgoja/19-express-keycloak-auth-host/cmd/host
  -> ENTRYPOINT ["/app/goja-auth-host", "serve"]
  -> CMD ["--listen", ":8080", "--script", "/app/server.js"]
```

Current K3s Deployment passes old flags and old env names:

```yaml
args:
  - --listen
  - :8080
  - --script
  - /app/server.js
env:
  - KEYCLOAK_ISSUER
  - KEYCLOAK_CLIENT_ID
  - KEYCLOAK_CLIENT_SECRET
  - PUBLIC_BASE_URL
  - SESSION_DB_DSN
  - AUDIT_DB_DSN
  - APPAUTH_DB_DSN
  - CAPABILITY_DB_DSN
```

Those env vars are consumed by example-19's hand-written Glazed command defaults.

## Target generated contract

The generated image should be generic:

```text
Dockerfile.auth-host
  -> xgoja build -f examples/xgoja/21-generated-host-auth/xgoja.yaml
  -> ENTRYPOINT ["/app/goja-auth-host"]
  -> CMD ["serve", "sites", "demo"]
```

The Deployment should then either pass explicit flags or, preferably, set generated env variables. Because example 21 declares:

```yaml
app:
  envPrefix: XGOJA_OIDC_DEMO
```

these generated Glazed fields can be supplied as env vars:

| Generated field | Env var | Source |
| --- | --- | --- |
| `--http-listen` | `XGOJA_OIDC_DEMO_HTTP_LISTEN` | literal `:8080` |
| `--auth-mode` | `XGOJA_OIDC_DEMO_AUTH_MODE` | literal `oidc` |
| `--auth-default-store-driver` | `XGOJA_OIDC_DEMO_AUTH_DEFAULT_STORE_DRIVER` | literal `postgres` |
| `--auth-default-store-dsn` | `XGOJA_OIDC_DEMO_AUTH_DEFAULT_STORE_DSN` | Vault `dsn` |
| `--auth-default-store-apply-schema` | `XGOJA_OIDC_DEMO_AUTH_DEFAULT_STORE_APPLY_SCHEMA` | literal `true` |
| `--auth-session-cookie-allow-insecure-http` | `XGOJA_OIDC_DEMO_AUTH_SESSION_COOKIE_ALLOW_INSECURE_HTTP` | literal `false` |
| `--auth-oidc-issuer-url` | `XGOJA_OIDC_DEMO_AUTH_OIDC_ISSUER_URL` | Vault `keycloak_issuer` |
| `--auth-oidc-client-id` | `XGOJA_OIDC_DEMO_AUTH_OIDC_CLIENT_ID` | Vault `keycloak_client_id` |
| `--auth-oidc-client-secret` | `XGOJA_OIDC_DEMO_AUTH_OIDC_CLIENT_SECRET` | Vault `keycloak_client_secret` |
| `--auth-oidc-public-base-url` | `XGOJA_OIDC_DEMO_AUTH_OIDC_PUBLIC_BASE_URL` | Vault `public_base_url` |
| `--auth-oidc-after-login-url` | `XGOJA_OIDC_DEMO_AUTH_OIDC_AFTER_LOGIN_URL` | literal `/` |
| `--auth-oidc-after-logout-url` | `XGOJA_OIDC_DEMO_AUTH_OIDC_AFTER_LOGOUT_URL` | literal `/` |

## Proposed implementation

### Source repository changes

1. Update `Dockerfile.auth-host` to build the generated example-21 binary with `xgoja build` and copy only that binary into the runtime image.
2. Use a runtime base that supports the generated binary's dynamic glibc dependency. The generated host currently links dynamically because the auth store package includes sqlite support; use `gcr.io/distroless/base-debian12:nonroot` instead of `static-debian12` unless the build is later made fully static.
3. Set `ENTRYPOINT ["/app/goja-auth-host"]` and `CMD ["serve", "sites", "demo"]`.
4. Update `.github/workflows/publish-auth-host-image.yaml` to watch example 21, smoke example 21, and describe the image as generated OIDC.

### K3s GitOps changes

1. Keep `replicas: 1`.
2. Update the Deployment args to `serve sites demo` or remove explicit args and rely on image `CMD`. Prefer explicit args in GitOps for readability.
3. Replace old example-19 env names with generated `XGOJA_OIDC_DEMO_*` env vars backed by the same Vault runtime secret.
4. Keep Service, Ingress, VaultStaticSecret, DB bootstrap, and Keycloak resources unchanged.

### Image publication

For a pre-main production trial, build and push a branch image manually:

```bash
docker build -f Dockerfile.auth-host \
  -t ghcr.io/go-go-golems/go-goja-auth-host:sha-<short-sha> .
docker push ghcr.io/go-go-golems/go-goja-auth-host:sha-<short-sha>
```

Then update the K3s Deployment image to that tag.

### Validation

Minimum generated-host public smoke:

1. Argo app is `Synced Healthy`.
2. `/healthz` returns the generated example payload.
3. `/auth/login` returns a Keycloak redirect.
4. `/me` returns `401` before login.
5. Browser or scripted OIDC login returns to `/auth/callback` and creates an app session.
6. `/me` returns the logged-in actor after login.
7. `POST /auth/logout` clears the session.

The old full example-19 smoke also checks `/auth/session`, invite routes, and seeded project authorization flows. The generated production replacement now provides compatibility handlers for those demo endpoints so the existing public smoke can pass unchanged while the platform runs the generated image.

## Decisions

### Decision 1: Runtime values stay in Kubernetes/Vault

- **Status:** accepted
- **Decision:** Do not bake issuer URLs, client IDs, client secrets, DSNs, or public URLs into the Dockerfile.
- **Rationale:** The generated binary supports Glazed env sourcing. Kubernetes already receives these values from Vault.
- **Consequence:** The image remains portable; production behavior is visible in GitOps env vars.

### Decision 2: Use generated env vars instead of explicit secret-expanded args

- **Status:** accepted
- **Decision:** Configure the generated binary with `XGOJA_OIDC_DEMO_*` environment variables.
- **Rationale:** This directly exercises generated xgoja's env/config support and keeps secrets out of process arguments.
- **Consequence:** Operators must know the generated env prefix mapping; this guide records it.

### Decision 3: Keep the production cut full-smoke compatible

- **Status:** accepted
- **Decision:** Replace the platform image/command/config with generated OIDC and include demo compatibility handlers for `/auth/session`, `/orgs/o1/invites`, and `/org-invites/accept`.
- **Rationale:** The existing full public smoke is the strongest deployment confidence signal and should pass unchanged for the first production replacement.
- **Consequence:** A small amount of demo-specific native routing currently lives in hostauth until a cleaner generated-host extension point exists.

## Rollback

Rollback is an image/args/env revert in K3s GitOps:

1. Restore the previous image tag `ghcr.io/go-go-golems/go-goja-auth-host:sha-ba77afc`.
2. Restore old args `--listen :8080 --script /app/server.js`.
3. Restore old env names `KEYCLOAK_*`, `PUBLIC_BASE_URL`, and per-store DSNs.
4. Sync Argo.

Keycloak, Vault, Postgres, Service, and Ingress do not need rollback.

## Review checklist

- Docker image builds and `docker run ... --help` works.
- Generated example smoke still passes locally.
- K3s Deployment does not expose secrets in args.
- `XGOJA_OIDC_DEMO_AUTH_SESSION_COOKIE_ALLOW_INSECURE_HTTP=false` is explicitly set.
- Argo reaches `Synced Healthy`.
- Public health/login/me/logout checks pass.
