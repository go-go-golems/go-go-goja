---
Title: "Auth stores and persistence reference"
Slug: auth-stores-reference
Short: "Choose and configure session, audit, appauth, and capability stores for gojahttp auth hosts."
Topics:
- xgoja
- gojahttp
- auth
- stores
- sqlite
- postgres
Commands:
- xgoja
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The auth host uses four store families. They are separate because they answer different questions: who has a session, what happened, which app resources exist, and which capability tokens are valid. They can share one physical PostgreSQL database, but they should remain separate concepts in code and documentation.

## Store families

| Store | Package | Responsibility |
| --- | --- | --- |
| Session | `pkg/gojahttp/auth/sessionauth` | Opaque app sessions, CSRF token state, and session metadata. |
| Audit | `pkg/gojahttp/auth/audit` | Security-relevant records for route execution and authorization decisions. |
| AppAuth | `pkg/gojahttp/auth/appauth` | App users, tenants, memberships, and resources used by authorization. |
| Capability | `pkg/gojahttp/auth/capability` | Bearer-like capability tokens for invite and delegated-action flows. |

The JavaScript route plan does not choose these stores. The Go host chooses them before requests are served.

## Drivers

Generated hostauth supports three driver choices.

| Driver | Use case | Persistence | Production note |
| --- | --- | --- | --- |
| `memory` | Fast local demos and unit tests. | Process-local only. | Do not use for production sessions or multi-pod deployments. |
| `sqlite` | Local persistent smoke tests. | File-backed or in-memory SQLite. | Useful for one-process development; not a shared cluster store. |
| `postgres` | Kubernetes and shared runtime deployments. | Shared database. | Use for production-shaped auth hosts. |

Example 19 uses PostgreSQL in the live yolo deployment. The same DSN is supplied to all four store families:

```text
SESSION_DB_DSN      -> goja_auth_host_demo database
AUDIT_DB_DSN        -> goja_auth_host_demo database
APPAUTH_DB_DSN      -> goja_auth_host_demo database
CAPABILITY_DB_DSN   -> goja_auth_host_demo database
```

That is a deployment simplification, not a type collapse. The Go process still constructs four store implementations.

## Schema application

Each SQL store has an `ApplySchema` method. It creates the tables and indexes needed by that store family.

```go
store, err := sessionSQLStore.New(sessionSQLStore.Config{DB: db, Dialect: sessionSQLStore.DialectPostgres})
if err != nil {
    return err
}
if err := store.ApplySchema(ctx); err != nil {
    return err
}
```

In generated hostauth, the setting is `apply-schema`. In the direct example 19 host, schema application happens in the helper functions that construct each store.

Use `apply-schema` for demos and single-app databases where the app may own its schema. For stricter production environments, move schema changes into migrations and set `apply-schema` false after the schema exists.

## Store construction and DB sharing

`hostauth.BuildStores` opens SQL database handles by `(driver, dsn)`. If multiple store configs resolve to the same SQL driver and DSN, they share one `*sql.DB` handle.

```text
auth.stores.default.driver = postgres
auth.stores.default.dsn    = postgres://...

session    -> shared *sql.DB
appauth    -> shared *sql.DB
audit      -> shared *sql.DB
capability -> shared *sql.DB
```

This avoids unnecessary connection pools while preserving separate store interfaces.

## Session store

The session store supports the `sessionauth.Manager`. It stores opaque app sessions and the data needed to authenticate future requests. The browser sees a cookie; the server loads the session record.

A production OIDC host should keep sessions in PostgreSQL so that restarts do not log out every user. Multi-replica deployments need durable session storage and either durable OIDC transaction state or routing that guarantees callback affinity.

## Audit store

The audit store records security-relevant route outcomes. It is not a metrics sink. It is where the application can answer questions such as which actor attempted which action on which resource.

Use PostgreSQL when audit records need to survive restarts or be inspected after an incident.

## AppAuth store

The appauth store owns application identity and authorization state:

- users created or updated from OIDC claims,
- tenants and tenant slugs,
- memberships such as user `u1` has role `admin` in tenant `o1`,
- resources such as project `p1` belongs to tenant `o1`.

The Keycloak subject is not enough for authorization. It identifies the browser user. The appauth store maps that identity into the application's domain model.

## Capability store

The capability store backs flows where a token represents a limited action. The token is not a session. It is a constrained grant with its own lifecycle and reuse checks.

The service exposes two read paths with different side effects:

| Operation | Store behavior | Use case |
| --- | --- | --- |
| Validate / lookup | Checks token hash, purpose, expiry, revocation, and used state without mutating the record. | Preview a token or check expected resource/type before consuming it. |
| Consume / redeem | Performs the same checks and then marks a single-use token as used. | Accept an invite or perform the delegated action exactly once. |

Example 21 uses the capability store from JavaScript through `auth.capabilities.issue(...)` and `auth.capabilities.consume(...)`. Its compose smoke verifies capability behavior by issuing an invite, accepting it, and then confirming that reusing the same token returns `409 Conflict`.

## PostgreSQL DSN examples

For cluster deployments with the shared Postgres service:

```text
postgres://goja_auth_host_demo_app:<password>@postgres.postgres.svc.cluster.local:5432/goja_auth_host_demo?sslmode=disable
```

For local Docker Compose:

```text
postgres://goja:goja@127.0.0.1:15432/goja?sslmode=disable
```

For SQLite local smoke:

```text
/tmp/xgoja-generated-host-auth.sqlite
```

SQLite DSNs are interpreted by `github.com/mattn/go-sqlite3`. PostgreSQL DSNs are interpreted by `github.com/lib/pq`.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| Store construction fails with unsupported driver | The driver is not `memory`, `sqlite`, or `postgres`. | Fix the driver value before startup. |
| SQL store fails during `ApplySchema` | The app role lacks DDL permission or the DSN points at the wrong database. | Check DB bootstrap, role ownership, and DSN. |
| Auth works until restart, then users are logged out | Sessions are in memory. | Use `postgres` or `sqlite` for the session store. |
| Authorization fails after login | User exists but memberships/resources are not seeded. | Check the OIDC normalizer and appauth seed logic. |
| Capability accept works repeatedly | Capability reuse is not being persisted or checked. | Use the capability store, consume single-use tokens, and test reused-token behavior. |

## See also

- `xgoja help generated-auth-javascript-apis`
- `xgoja help hostauth-config-reference`
- `xgoja help express-auth-host-integration-guide`
- `goja-repl help auth-module-guide`
- `goja-repl help express-auth-examples`
- `goja-repl help deploying-an-express-auth-host`
