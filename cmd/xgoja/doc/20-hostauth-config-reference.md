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
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`hostauth.Config` is host infrastructure configuration. It is not the JavaScript route policy DSL. JavaScript declares which routes require authentication, resources, CSRF, and actions; `hostauth.Config` tells a generated or embedded host which auth services and persistence backends to construct.

The configuration lives in `pkg/xgoja/hostauth`. Generated HTTP `serve` commands can expose it as Glazed `--auth-*` fields when a host injects `hostauth.ServiceFactoryKey`.

## Status of OIDC mode

`auth.mode=oidc` is accepted by the public enum, but it is not implemented in the generated-host resolver yet. `ResolveConfig` currently returns:

```text
hostauth: auth.mode=oidc is not implemented yet
```

Use the example 19 Keycloak host for production-shaped OIDC today. It calls `keycloakauth.New` directly and wires the stores by hand. The fully generated `auth.mode=oidc` path is tracked by GitHub issue #82.

This distinction matters because a generated binary may show `oidc` as a choice in `--auth-mode`, but selecting it will fail at startup until the resolver and service builder support it.

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
```

The top-level fields are:

| Field | Type | Meaning |
| --- | --- | --- |
| `mode` | `none`, `dev`, `oidc` | Selects the auth infrastructure shape. `oidc` is currently a hard-stop in generated hostauth. |
| `session` | object | Controls server-side app-session cookies and timeouts. |
| `stores` | object | Configures session, audit, appauth, and capability persistence. |

## Modes

| Mode | Current behavior |
| --- | --- |
| `none` | No generated auth services are built. Store config is ignored. Use for public-only hosts. |
| `dev` | Builds development auth services and configured stores. Use for local generated-host auth demos. |
| `oidc` | Reserved for production OIDC. Currently returns `ErrOIDCNotImplemented`. |

Use direct `keycloakauth.New` composition, as in `examples/xgoja/19-express-keycloak-auth-host`, until `oidc` is implemented for generated hosts.

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

## Validation rules

`ResolveConfig` enforces these rules:

- Unknown modes fail with `auth.mode: unsupported auth mode`.
- `auth.mode=oidc` fails with `ErrOIDCNotImplemented`.
- Unknown SameSite values fail under `auth.session.cookie.same-site`.
- Cookie paths must start with `/`.
- Duration strings must parse as positive Go durations.
- SQL stores require a DSN.
- Unknown drivers fail under the relevant `auth.stores.<name>.driver` path.

The error path is part of the operator experience. Preserve it when adding new fields so config mistakes point to the exact setting.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `auth.mode: hostauth: auth.mode=oidc is not implemented yet` | Generated hostauth does not support OIDC yet. | Use example 19 direct `keycloakauth.New` composition or implement issue #82. |
| SQL store fails with missing DSN | Driver is `sqlite` or `postgres` and no DSN was provided. | Set `--auth-default-store-dsn` or a per-store DSN. |
| Cookie path validation fails | Path does not start with `/`. | Use `/` or a rooted path such as `/app`. |
| Session cookie is not accepted in local smoke | Running HTTP without local insecure override. | For localhost only, set `--auth-session-cookie-allow-insecure-http`. |
| Generated command has no `auth` section | The host did not inject a `hostauth.ServiceFactoryKey`. | Add the host service factory before constructing the HTTP `serve` command set. |

## See also

- `xgoja help auth-stores-reference`
- `xgoja help http-serve-command-reference`
- `xgoja help express-auth-host-integration-guide`
- `xgoja help auth-host-production-runbook`
- `goja-repl help express-auth-examples`
