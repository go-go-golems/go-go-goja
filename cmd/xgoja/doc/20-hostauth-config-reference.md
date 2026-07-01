---
Title: "hostauth configuration reference"
Slug: hostauth-config-reference
Short: "Reference for generated-host auth mode, session cookies, and store configuration."
Topics:
- xgoja
- auth
- config
- gojahttp
Commands:
- xgoja
Flags:
- --auth-mode
- --auth-default-store-driver
- --auth-default-store-dsn
- --auth-default-store-apply-schema
- --auth-oidc-issuer-url
- --auth-oidc-client-id
- --auth-oidc-client-secret
- --auth-oidc-public-base-url
- --auth-oidc-redirect-url
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`hostauth.Config` is host infrastructure configuration. It is not the JavaScript route policy DSL. JavaScript declares which routes require authentication, resources, CSRF, and actions; `hostauth.Config` tells a generated or embedded host which auth services and persistence backends to construct.

The configuration lives in `pkg/xgoja/hostauth`. Generated HTTP `serve` commands expose it as Glazed `--auth-*` fields when the generated runtime plan or embedding host installs `hostauth.ServiceFactoryKey`.

## Status of OIDC mode

`auth.mode=oidc` is implemented for generated HTTP `serve` hosts. At startup, `hostauth` resolves session/store settings, discovers the OIDC issuer, builds native Go handlers, and the HTTP provider mounts them before the JavaScript app fallback:

```text
GET  /auth/login
GET  /auth/callback
GET  /auth/logout
POST /auth/logout
GET  /auth/session
```

The browser receives an opaque app-session cookie. ID/access tokens stay server-side. JavaScript routes continue to declare authorization intent with planned Express auth builders such as `express.user().required()` and `.allow("action.name")`; OIDC is only the login mechanism.

Generic hostauth does not own demo application routes such as `/auth/audit`, `/orgs/o1/invites`, or `/org-invites/accept`. Generated applications should implement those in JavaScript with planned routes and the high-level `require("auth")` APIs. See `xgoja help generated-auth-javascript-apis` for the audit and capability builders.

See `examples/xgoja/21-generated-host-auth` for a generated binary fixture that uses top-level `auth.mode=oidc`, smoke-tests `/auth/login` with a fake discovery provider, and includes a real Docker Compose Keycloak/Postgres smoke.

## Nested YAML shape

The native Go configuration type is nested:

```yaml
auth:
  mode: dev
  session:
    cookie:
      allow-insecure-http: false
      name: ""
      same-site: lax
      path: /
    idle-timeout: 8h
    absolute-timeout: 24h
  stores:
    default:
      driver: postgres
      dsn: postgres://user:pass@postgres.postgres.svc.cluster.local:5432/app?sslmode=disable
      apply-schema: true
    session: {}
    audit: {}
    appauth: {}
    capability: {}
  oidc:
    issuer-url: https://auth.example.test/realms/demo
    client-id: demo-app
    client-secret: ""
    public-base-url: https://demo.example.test
    redirect-url: ""
    scopes: [profile, email]
    after-login-url: /
    after-logout-url: /
```

The top-level fields are:

| Field | Type | Meaning |
| --- | --- | --- |
| `mode` | `none`, `dev`, `oidc` | Selects the auth infrastructure shape. |
| `session` | object | Controls server-side app-session cookies and timeouts. |
| `stores` | object | Configures session, audit, appauth, and capability persistence. |
| `oidc` | object | Configures browser OIDC login when `mode=oidc`. |

## Modes

| Mode | Current behavior |
| --- | --- |
| `none` | No generated auth services are built. Store config is ignored. Use for public-only hosts. |
| `dev` | Builds development auth services and configured stores. Use for local generated-host auth demos. |
| `oidc` | Builds configured stores, discovers the OIDC issuer, and mounts native login/callback/logout handlers before the app host. |

Use `dev` for local app authorization work and `oidc` for browser login through Keycloak or another OIDC provider. Example 19 remains a hand-composed production reference, while example 21 is the generated OIDC fixture.

## Session configuration

`SessionConfig` controls app-session behavior after a user is authenticated.

| YAML field | Glazed field | Meaning |
| --- | --- | --- |
| `session.cookie.allow-insecure-http` | `--auth-session-cookie-allow-insecure-http` | Allows non-Secure cookies for localhost HTTP demos. Keep false behind ingress. |
| `session.cookie.name` | `--auth-session-cookie-name` | Optional cookie name. Empty uses the session manager default. |
| `session.cookie.same-site` | `--auth-session-cookie-same-site` | `lax`, `strict`, `none`, `default`, or empty. Empty defaults to Lax. |
| `session.cookie.path` | `--auth-session-cookie-path` | Cookie path. Empty defaults to `/`; non-empty values must start with `/`. |
| `session.idle-timeout` | `--auth-session-idle-timeout` | Optional Go duration such as `8h`. |
| `session.absolute-timeout` | `--auth-session-absolute-timeout` | Optional Go duration such as `24h`. |

Timeout values use `time.ParseDuration`. Values must be positive when present.

## OIDC configuration

`OIDCConfig` controls browser login when `auth.mode=oidc`.

| YAML field | Glazed field | Meaning |
| --- | --- | --- |
| `oidc.issuer-url` | `--auth-oidc-issuer-url` | OIDC issuer URL. Required for `auth.mode=oidc`. |
| `oidc.client-id` | `--auth-oidc-client-id` | OIDC client ID. Required for `auth.mode=oidc`. |
| `oidc.client-secret` | `--auth-oidc-client-secret` | Optional client secret for confidential clients. Prefer flags/env/secrets over committed YAML. |
| `oidc.public-base-url` | `--auth-oidc-public-base-url` | Browser-visible application origin. Callback defaults to `<public-base-url>/auth/callback`. |
| `oidc.redirect-url` | `--auth-oidc-redirect-url` | Advanced explicit callback override. Use only when the callback is not `<public-base-url>/auth/callback`. |
| `oidc.scopes` | `--auth-oidc-scopes` | Additional scopes. `openid` is added automatically by the OIDC layer. |
| `oidc.after-login-url` | `--auth-oidc-after-login-url` | Relative URL after successful login. Defaults to `/`. |
| `oidc.after-logout-url` | `--auth-oidc-after-logout-url` | Relative URL after logout. Defaults to `/`. |

`public-base-url` and `redirect-url` are deliberately separate from the listen
address. Do not derive browser callback URLs from `--http-listen`; listen may be
`:8080` inside Kubernetes while the browser-visible origin is
`https://app.example.test` behind ingress. HTTPS is required for non-localhost
OIDC callback URLs unless `session.cookie.allow-insecure-http` is true for a
local smoke test.

The default OIDC user normalizer upserts users through `appauth.UpsertFromOIDC`
and projects existing memberships into the app session. It does not grant demo
roles, tenants, or capabilities automatically.

## Store inheritance

`StoresConfig` has a `default` store and four per-store overrides. The per-store blocks inherit from `default` field by field.

```yaml
auth:
  stores:
    default:
      driver: postgres
      dsn: postgres://user:pass@postgres:5432/auth?sslmode=disable
      apply-schema: true
    audit:
      apply-schema: false
```

In this example, `audit` uses the default Postgres driver and DSN but overrides `apply-schema` to false. Empty per-store blocks inherit all default fields.

Store drivers are:

| Driver | Meaning |
| --- | --- |
| `memory` | Process-local in-memory state. Fast and disposable. Not suitable for multi-pod production sessions. |
| `sqlite` | Local SQL store using `github.com/mattn/go-sqlite3`. Useful for local persistence. |
| `postgres` | PostgreSQL store using `github.com/lib/pq`. Use this for cluster deployments. |

SQL stores require a non-empty DSN. `memory` ignores DSN.

## Flat Glazed fields

Generated commands expose a flat public shape because command-line flags should be readable:

```bash
./generated-host serve sites demo \
  --auth-mode dev \
  --auth-default-store-driver postgres \
  --auth-default-store-dsn "$DATABASE_URL" \
  --auth-default-store-apply-schema
```

The generated section slug is `auth`. Field names are prefixed with `auth-` so they do not collide with provider-specific flags.

The full store flag pattern is:

```text
--auth-default-store-driver
--auth-default-store-dsn
--auth-default-store-apply-schema
--auth-session-store-driver
--auth-session-store-dsn
--auth-session-store-apply-schema
--auth-audit-store-driver
--auth-audit-store-dsn
--auth-audit-store-apply-schema
--auth-appauth-store-driver
--auth-appauth-store-dsn
--auth-appauth-store-apply-schema
--auth-capability-store-driver
--auth-capability-store-dsn
--auth-capability-store-apply-schema
```

The OIDC flag pattern is:

```text
--auth-oidc-issuer-url
--auth-oidc-client-id
--auth-oidc-client-secret
--auth-oidc-public-base-url
--auth-oidc-redirect-url
--auth-oidc-scopes
--auth-oidc-after-login-url
--auth-oidc-after-logout-url
```

## Validation rules

`ResolveConfig` enforces these rules:

- Unknown modes fail with `auth.mode: unsupported auth mode`.
- OIDC mode requires `auth.oidc.issuer-url` and `auth.oidc.client-id`.
- OIDC mode requires either `auth.oidc.public-base-url` or `auth.oidc.redirect-url`.
- OIDC callback URLs must be HTTPS unless they are localhost HTTP with `auth.session.cookie.allow-insecure-http=true`.
- Unknown SameSite values fail under `auth.session.cookie.same-site`.
- Cookie paths must start with `/`.
- Duration strings must parse as positive Go durations.
- SQL stores require a DSN.
- Unknown drivers fail under the relevant `auth.stores.<name>.driver` path.

The error path is part of the operator experience. Preserve it when adding new fields so config mistakes point to the exact setting.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `auth.oidc.issuer-url is required` | OIDC mode has no issuer. | Set `--auth-oidc-issuer-url` or YAML `auth.oidc.issuer-url`. |
| `auth.oidc.public-base-url: public-base-url or redirect-url is required` | OIDC mode cannot derive the callback. | Set `--auth-oidc-public-base-url`; use `--auth-oidc-redirect-url` only for a custom callback. |
| `OIDC callback URL must use https` | Public callback URL is HTTP outside localhost. | Use HTTPS in production, or set insecure HTTP only for localhost smoke tests. |
| SQL store fails with missing DSN | Driver is `sqlite` or `postgres` and no DSN was provided. | Set `--auth-default-store-dsn` or a per-store DSN. |
| Cookie path validation fails | Path does not start with `/`. | Use `/` or a rooted path such as `/app`. |
| Session cookie is not accepted in local smoke | Running HTTP without local insecure override. | For localhost only, set `--auth-session-cookie-allow-insecure-http`. |
| Generated command has no `auth` section | The generated runtime plan has no top-level `auth:` block and the embedding host did not inject a `hostauth.ServiceFactoryKey`. | Add top-level `auth:` to `xgoja.yaml` or install the host service factory before constructing the HTTP `serve` command set. |

## See also

- `xgoja help generated-auth-javascript-apis`
- `xgoja help auth-stores-reference`
- `xgoja help http-serve-command-reference`
- `xgoja help express-auth-host-integration-guide`
- `xgoja help auth-host-production-runbook`
- `goja-repl help auth-module-guide`
- `goja-repl help express-auth-examples`
