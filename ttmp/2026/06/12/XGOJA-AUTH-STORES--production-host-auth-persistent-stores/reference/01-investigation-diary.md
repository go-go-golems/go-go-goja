---
Title: Investigation diary
Ticket: XGOJA-AUTH-STORES
Status: active
Topics:
    - goja
    - http
    - security
    - keycloak
    - oidc
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/19-express-keycloak-auth-host/Makefile
      Note: Adds Postgres/session DSN variables to smoke and serve targets (commit e53d063)
    - Path: examples/xgoja/19-express-keycloak-auth-host/README.md
      Note: Documents Postgres-backed Keycloak smoke (commit e53d063)
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: Wires optional Postgres-backed sessionauth/sqlstore into example host (commit e53d063)
    - Path: examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml
      Note: Adds Postgres to Keycloak smoke stack for SQL session testing (commit e53d063)
    - Path: examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh
      Note: Starts/waits for Postgres and passes session DSN to host (commit e53d063)
    - Path: pkg/gojahttp/auth/appauth/appauth.go
      Note: Clones appauth memory store inputs/outputs for contract parity (commit 22eb7d6)
    - Path: pkg/gojahttp/auth/audit/audit.go
      Note: Adds audit.MemoryStore and record clone isolation (commit 22eb7d6)
    - Path: pkg/gojahttp/auth/audit/sqlstore/README.md
      Note: Operational SQL query examples for audit records (commit 8821692)
    - Path: pkg/gojahttp/auth/audit/sqlstore/schema.go
      Note: SQLite/Postgres audit schema DDL (commit 8821692)
    - Path: pkg/gojahttp/auth/audit/sqlstore/sqlstore.go
      Note: SQL-backed audit.Store implementation (commit 8821692)
    - Path: pkg/gojahttp/auth/audit/sqlstore/sqlstore_test.go
      Note: Contract
    - Path: pkg/gojahttp/auth/capability/capability.go
      Note: Clones capability used/revoked timestamps for store isolation (commit 22eb7d6)
    - Path: pkg/gojahttp/auth/internal/appauthtest/store_contract.go
      Note: Reusable appauth store contract harness (commit 22eb7d6)
    - Path: pkg/gojahttp/auth/internal/audittest/store_contract.go
      Note: Reusable audit.Store contract tests (commit 22eb7d6)
    - Path: pkg/gojahttp/auth/internal/capabilitytest/store_contract.go
      Note: Reusable capability.Store contract tests including atomic redemption (commit 22eb7d6)
    - Path: pkg/gojahttp/auth/internal/sessionauthtest/store_contract.go
      Note: Reusable sessionauth.Store contract tests (commit 22eb7d6)
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: Clones session MFA/revocation timestamps for store isolation (commit 22eb7d6)
    - Path: pkg/gojahttp/auth/sessionauth/sqlstore/schema.go
      Note: SQLite/Postgres session schema DDL (commit 304f833)
    - Path: pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go
      Note: SQL-backed sessionauth.Store implementation (commit 304f833)
    - Path: pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore_test.go
      Note: Contract and full-projection tests for SQL session store (commit 304f833)
ExternalSources: []
Summary: Chronological diary for production auth persistent store planning.
LastUpdated: 2026-06-12T21:10:00-04:00
WhatFor: Use when resuming work on SQL/Postgres-backed stores for sessions, audit, capabilities, and app auth domain data.
WhenToUse: Before implementing or reviewing persistent auth store work.
---





# Diary

## Goal

Capture the planning and implementation history for production persistent stores backing the gojahttp host-auth packages.

## Step 1: Create persistent auth store ticket

This step created a focused ticket for the first next auth work item: replacing in-memory host-auth stores with durable production stores. The ticket intentionally keeps storage behind existing Go interfaces so Express route declarations remain unchanged.

The ticket separates persistent storage from Keycloak/MFA hardening. That keeps the first implementation path concrete: sessions, audit, capabilities, and app-owned auth data need durable stores before the production Keycloak example can become a realistic deployment reference.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket for 1. Create a new ticket for 2 and 3.\n\nIs body schema relevant for auth / security?"

**Assistant interpretation:** Create one docmgr ticket for persistent auth stores, create a second ticket for production Keycloak hardening plus MFA flows, and answer whether body/schema validation belongs in the auth/security roadmap.

**Inferred user intent:** Split the next auth roadmap into actionable tickets while clarifying the security relevance of request body validation.

**Commit (code):** N/A — ticket setup only.

### What I did
- Created ticket `XGOJA-AUTH-STORES`.
- Added design document `design/01-persistent-auth-store-implementation-plan.md`.
- Added this investigation diary.
- Added phased tasks for store contracts, session SQL store, audit SQL store, capability SQL store, appauth SQL store, and example integration.

### Why
- The current host-auth examples use in-memory stores by design.
- Production deployments need durable session, audit, capability, user, tenant, membership, and resource state.
- Store work is a separable foundation for later Keycloak and MFA hardening.

### What worked
- The ticket workspace and initial docs were created successfully with docmgr.
- The design maps directly to existing package seams: `sessionauth.Store`, `audit.Store`, `capability.Store`, and appauth store interfaces.

### What didn't work
- N/A.

### What I learned
- The persistent-store work is best scoped as interface-backed adapters, not as changes to Express or JavaScript route planning.
- Capability token persistence has the strongest transactional requirement because single-use redemption must be atomic.

### What was tricky to build
- The main planning nuance was deciding whether appauth persistence should be generic or app-specific. The ticket currently proposes a starter store while preserving replaceable domain interfaces.

### What warrants a second pair of eyes
- Whether Postgres should be the default implementation target or whether the repo wants a different persistence abstraction.
- Whether appauth SQL should live in this repo or only in examples.
- Whether migrations should be embedded SQL files or Go-rendered DDL.

### What should be done in the future
- Implement store contract tests before writing SQL adapters.
- Decide on concrete migration tooling and test database strategy.

### Code review instructions
- Start with the design document and tasks in this ticket.
- Compare proposed stores against `pkg/gojahttp/auth/sessionauth`, `audit`, `capability`, and `appauth` interfaces.

### Technical details
- Primary design doc:
  ```text
  ttmp/2026/06/12/XGOJA-AUTH-STORES--production-host-auth-persistent-stores/design/01-persistent-auth-store-implementation-plan.md
  ```


## Step 2: Add reusable auth store contract tests

This step started the active implementation work for `XGOJA-AUTH-STORES` by extracting reusable contract tests for the existing host-auth store interfaces. The goal was to make the current in-memory stores prove the behavior that future SQL/Postgres adapters must preserve before any database code is written.

The contract tests also exposed clone-isolation gaps in the in-memory stores. I fixed those while adding the tests so future persistent adapters have a clear baseline: stores must not let caller-owned input maps, slices, byte slices, or timestamp pointers mutate stored auth state after insert or through returned values.

### Prompt Context

**User prompt (verbatim):** "yeah add a ticket, then work on the auth stores ticket. keep a detailed diary, commit at appropriate intervals."

**Assistant interpretation:** Create the remaining auth follow-up ticket, then begin implementing the persistent-store ticket with detailed diary updates and focused commits.

**Inferred user intent:** Preserve the rest of the auth roadmap in docmgr while making concrete implementation progress on production auth stores.

**Commit (code):** `22eb7d6` — "Add auth store contract tests"; `c495b26` — "Add generated logcopter stubs for auth store tests"

### What I did
- Added reusable contract test helpers:
  - `pkg/gojahttp/auth/internal/sessionauthtest/store_contract.go`
  - `pkg/gojahttp/auth/internal/audittest/store_contract.go`
  - `pkg/gojahttp/auth/internal/capabilitytest/store_contract.go`
  - `pkg/gojahttp/auth/internal/appauthtest/store_contract.go`
- Added memory-store contract test entrypoints:
  - `pkg/gojahttp/auth/sessionauth/memory_store_contract_test.go`
  - `pkg/gojahttp/auth/audit/memory_store_contract_test.go`
  - `pkg/gojahttp/auth/capability/memory_store_contract_test.go`
  - `pkg/gojahttp/auth/appauth/memory_store_contract_test.go`
- Added `audit.MemoryStore` so `audit.Store` has a concrete in-memory implementation with `Snapshot()` for contract testing.
- Tightened clone isolation in memory stores:
  - `sessionauth.cloneSession` now clones `MFAAt` and `RevokedAt` pointers.
  - `capability.cloneCapability` now clones `UsedAt` and `RevokedAt` pointers.
  - `appauth.MemoryStore` now clones users, memberships, and resources on add/load paths.
  - `audit.MemoryStore` clones `Record.Attributes` recursively on insert and snapshot.
- Checked tasks 1-4 in `XGOJA-AUTH-STORES`.

### Why
- SQL/Postgres adapters need executable contracts before implementation so behavior does not drift from the host-auth semantics.
- Sessions and capabilities contain security-sensitive mutable state; stores must not expose internal state through pointer/map/slice aliasing.
- Capability single-use redemption must be proven atomic, so the contract includes a concurrent redemption test that future SQL stores must satisfy.

### What worked
- `go test ./pkg/gojahttp/auth/... -count=1` passes after moving contract invocation into external test packages.
- The pre-commit hook also ran lint and `go test ./...` successfully for commit `22eb7d6`.
- The contract helpers can be reused by future `sqlstore` packages without duplicating behavior assertions.

### What didn't work
- My first attempt put `TestMemoryStoreContract` in the same package test files while the reusable helpers imported those packages. That created Go import cycles.
- Failed command:
  ```bash
  go test ./pkg/gojahttp/auth/... -count=1
  ```
- Exact error shape:
  ```text
  # github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth
  package github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth
      imports github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/appauthtest from appauth_test.go
      imports github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth from store_contract.go: import cycle not allowed in test
  ```
- The same pattern failed for `audit`, `capability`, and `sessionauth`.
- Fix: move the contract entrypoint tests into separate external test packages (`appauth_test`, `audit_test`, `capability_test`, `sessionauth_test`) so the package under test is imported from the outside and the internal helper can import it without a cycle.

### What I learned
- Reusable test helpers that import the package under test should be called from external test packages, not from same-package tests, to avoid cycles.
- The current in-memory stores were mostly correct but did not fully clone pointer fields; this matters for future SQL parity because SQL adapters naturally deserialize fresh values while memory stores can accidentally share pointers.
- The audit package lacked an `audit.Store` memory implementation; `MemorySink` implements `gojahttp.AuditSink`, but SQL adapter contracts need a `Store` shape.

### What was tricky to build
- The tricky part was designing reusable contracts without making package import cycles or overfitting to memory-store helper methods. `appauth` needed a harness because its interfaces are split into `UserStore`, `MembershipStore`, and `ResourceStore`, while seeding data requires methods that are intentionally not part of those interfaces.
- The capability contract needed to test single-use atomicity without assuming a SQL implementation. I used concurrent `Redeem` calls and assert exactly one success with the rest returning `ErrUsed`; this gives the future SQL adapter a clear behavioral target.
- The audit store contract needed a query/snapshot hook because `audit.Store` is insert-only. I made the contract accept a `Snapshot` function in its harness so future SQL tests can implement the hook with a test query without widening the production interface.

### What warrants a second pair of eyes
- Whether `audit.MemoryStore` should remain a public package type or be kept test-only. I kept it public because it is a useful concrete `audit.Store` for tests and demos, mirroring other auth memory stores.
- Whether capability `Revoke` should become idempotent for missing IDs in a later step. The current contract preserves existing `ErrNotFound` behavior, even though the design doc notes idempotent revocation as desirable for production adapters.
- Whether clone isolation should also deep-copy arbitrary nested `sessionauth.Session.Claims` and `appauth.Resource.Claims` values beyond the top-level map. The current code preserves existing shallow claim behavior except where audit records needed recursive map cloning.

### What should be done in the future
- Start Phase 2 by choosing SQL package layout and migration shape for `sessionauth/sqlstore`.
- Reuse `sessionauthtest.RunStoreContract` against the SQL session store immediately after the first adapter compiles.
- Decide whether to tighten deep-copy semantics for arbitrary `Claims` values before SQL adapters make serialization boundaries more explicit.

### Code review instructions
- Start with the contract helper packages under `pkg/gojahttp/auth/internal/*test`.
- Then review each memory-store contract entrypoint under the auth subpackages.
- Finally review the clone changes in:
  - `pkg/gojahttp/auth/sessionauth/sessionauth.go`
  - `pkg/gojahttp/auth/capability/capability.go`
  - `pkg/gojahttp/auth/appauth/appauth.go`
  - `pkg/gojahttp/auth/audit/audit.go`
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/... -count=1
  go test ./... -count=1
  ```

### Technical details
- Contract helper strategy:
  ```go
  sessionauthtest.RunStoreContract(t, func(testing.TB) sessionauth.Store {
      return sessionauth.NewMemoryStore()
  })
  ```
- Appauth contract harness strategy:
  ```go
  appauthtest.Harness{
      Users:       store,
      Memberships: store,
      Resources:   store,
      AddUser:     store.AddUser,
      AddMember:   store.AddMembership,
      AddResource: store.AddResource,
  }
  ```


## Step 3: Add the first SQL-backed session store

This step implemented the first durable auth store adapter: `sessionauth/sqlstore`. The store is intentionally behind the existing `sessionauth.Store` interface, so `sessionauth.Manager`, Keycloak callback code, and planned Express routes do not need to know whether sessions live in memory, SQLite, or Postgres.

The implementation includes SQLite-backed tests for fast local verification and a Postgres schema/placeholder path for production deployment. This gives the next auth-store phases a concrete pattern to follow: small `database/sql` subpackages, embedded DDL, contract tests, and focused behavior tests for full auth projections.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue implementing the auth stores ticket after the contract-test baseline, committing coherent chunks and documenting failures.

**Inferred user intent:** Move from planning into production persistence, starting with sessions as the foundation for Keycloak and app login flows.

**Commit (code):** `304f833` — "Add SQL session auth store"

### What I did
- Added package `pkg/gojahttp/auth/sessionauth/sqlstore`.
- Added `sqlstore.Store`, which implements `sessionauth.Store` with `database/sql`.
- Added dialect support for:
  - `DialectSQLite` for fast tests and local/demo usage.
  - `DialectPostgres` for production-shaped schema and placeholders.
- Added embedded DDL constants:
  - `SQLiteSchema`
  - `PostgresSchema`
- Added `ApplySchema(ctx)` for tests/examples/simple migrations.
- Implemented session operations:
  - `Create`
  - `Get`
  - `Touch`
  - `Rotate`
  - `Revoke`
- Added SQLite tests that run the reusable `sessionauthtest.RunStoreContract` against the SQL store.
- Added a full-projection persistence test for identity fields, tenant IDs, MFA timestamp, revoked timestamp, and claims.
- Added a small Postgres schema/placeholder test so the production dialect path is at least covered structurally.
- Checked tasks 5-8 in `XGOJA-AUTH-STORES`.

### Why
- Server-side app sessions are the first production persistence requirement: Keycloak login, opaque browser cookies, CSRF, MFA freshness, revocation, and rotation all depend on durable session state.
- Implementing sessions first validates the package layout and contract-test approach before moving to audit, capability, and appauth stores.
- SQLite tests keep the default test suite fast while preserving a Postgres-oriented schema for deployment work.

### What worked
- `go test ./pkg/gojahttp/auth/sessionauth/... -count=1` passed after implementation.
- `go test ./pkg/gojahttp/auth/... -count=1` passed before commit.
- The final pre-commit hook passed lint and `go test ./...` for commit `304f833`.
- `go generate` produced the expected `pkg/gojahttp/auth/sessionauth/sqlstore/logcopter.go`, which was included in the code commit.

### What didn't work
- The first commit attempt failed during the pre-commit lint phase because `db.Close()` in `TestPostgresSchemaAndPlaceholders` was deferred without checking the returned error.
- Failed command:
  ```bash
  git commit -m "Add SQL session auth store"
  ```
- Exact lint error:
  ```text
  pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore_test.go:70:16: Error return value of `db.Close` is not checked (errcheck)
      defer db.Close()
                    ^
  1 issues:
  * errcheck: 1
  ```
- Fix: replace `defer db.Close()` with `t.Cleanup(func() { _ = db.Close() })`, then rerun `gofmt`, package tests, and commit.

### What I learned
- The repo’s pre-commit path runs both lint and full tests, so test-only cleanup must still satisfy `errcheck`.
- Keeping the session store behind `database/sql` avoids introducing a Postgres driver dependency immediately, but the actual Postgres runtime still needs to be proven later with a real driver/container.
- JSON payload handling is a useful cross-dialect compromise for tenant IDs and claims, but we should verify Postgres JSONB parameter behavior in an integration test before calling the adapter production-complete.

### What was tricky to build
- The tricky part was balancing Postgres production shape with fast local tests. The package currently keeps the same Go code path for both dialects but uses dialect-specific placeholders and DDL.
- Another sharp edge was preserving the existing `sessionauth.Store` semantics. `Rotate` deletes the old ID and inserts the new session inside one SQL transaction, while `Revoke` remains idempotent for missing sessions to match the in-memory store contract.
- `audit`, `capability`, and `appauth` stores will need their own query/test harnesses; the session store was simpler because the interface already has `Get`.

### What warrants a second pair of eyes
- Whether `ApplySchema` should remain in the package API or move to examples/tests only once formal migrations exist.
- Whether the Postgres schema should use JSONB columns named `tenant_ids_json` / `claims_json` or cleaner names such as `tenant_ids` / `claims` before any public release.
- Whether `database/sql` is the right long-term abstraction or whether the repo should standardize on a migration/query helper before more stores are added.
- Whether `Revoke` should update only non-revoked sessions or always overwrite `revoked_at` as it does now.

### What should be done in the future
- Add a real Postgres integration test or smoke path for `sessionauth/sqlstore`.
- Document session store production cookie and migration requirements for task 9.
- Implement `audit/sqlstore` next, reusing the `database/sql` package layout and contract harness pattern.

### Code review instructions
- Start with `pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go` for interface implementation and transaction semantics.
- Review `pkg/gojahttp/auth/sessionauth/sqlstore/schema.go` for SQLite/Postgres DDL choices.
- Review `pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore_test.go` for contract and full-projection coverage.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/sessionauth/... -count=1
  go test ./pkg/gojahttp/auth/... -count=1
  ```

### Technical details
- Basic setup:
  ```go
  db, _ := sql.Open("postgres", dsn)
  sessions, err := sqlstore.New(sqlstore.Config{
      DB:      db,
      Dialect: sqlstore.DialectPostgres,
  })
  ```
- Test/example schema application:
  ```go
  if err := sessions.ApplySchema(ctx); err != nil {
      return err
  }
  ```


## Step 4: Wire the Keycloak smoke to Postgres-backed sessions

This step made the Keycloak host example exercise the new SQL session store against a real Postgres container. The example already had a production-shaped Keycloak flow; adding Postgres to the same Docker Compose stack gives us an end-to-end smoke that proves Keycloak login, opaque app-session creation, CSRF, authorization, logout, and session revocation with durable app sessions rather than only `sessionauth.MemoryStore`.

The Postgres service is intentionally for the Go host's app sessions, not for replacing Keycloak's own dev database. That keeps the smoke focused on the new `sessionauth/sqlstore` implementation while preserving the fast Keycloak realm import path.

### Prompt Context

**User prompt (verbatim):** "add postgres to the keycloak docker-compose as well. that way you can test it!"

**Assistant interpretation:** Extend the Keycloak example compose stack with Postgres and wire the example host/smoke to use the SQL session store so the new persistent session implementation is exercised end-to-end.

**Inferred user intent:** Turn the SQL session store from a unit-tested adapter into something validated by the existing realistic Keycloak smoke.

**Commit (code):** `e53d063` — "Wire Keycloak smoke to Postgres sessions"

### What I did
- Added a `postgres` service to `examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml`.
- Exposed Postgres on `${POSTGRES_PORT:-15432}` with database/user/password `goja_auth` / `goja` / `goja`.
- Added a persistent compose volume `postgres-data`.
- Added `github.com/lib/pq` so the example host can open Postgres through `database/sql`.
- Added `--session-db-dsn` / `SESSION_DB_DSN` to the Keycloak host.
- Wired the host to use `sessionauth/sqlstore` when a DSN is provided and fall back to `sessionauth.MemoryStore` when it is not.
- Updated the example Makefile and smoke script so `make smoke` starts Keycloak + Postgres and passes the Postgres DSN to the host.
- Updated the example README to document Postgres and the new smoke behavior.
- Checked task 24 in `XGOJA-AUTH-STORES`.

### Why
- The SQL session store needs a real Postgres validation path, not only SQLite-backed contract tests.
- The Keycloak smoke is the best end-to-end path because it creates sessions through the actual OIDC callback and then uses those sessions for planned routes, CSRF, and logout.
- Keeping the DSN optional preserves the manual in-memory demo path while making the automated smoke persistent by default.

### What worked
- The Keycloak smoke passed with Postgres-backed app sessions:
  ```bash
  make -C examples/xgoja/19-express-keycloak-auth-host smoke
  ```
- Key smoke output:
  ```text
  ok postgres ready
  ok keycloak discovery http://127.0.0.1:18080/realms/goja-demo/.well-known/openid-configuration
  ok host health http://127.0.0.1:8790/healthz
  ok public health                200
  ok me before login              401
  ok login page                   200
  ok keycloak form login          200
  ok login redirected to host     http://127.0.0.1:8790/
  ok me after login               200
  ok session after login          200
  ok project missing csrf         403
  ok project update               200
  ok project missing              404
  ok logout                       204
  ok me after logout              401
  {"status": "PASS", "actorId": "user:57448654-7e66-43f3-9313-71c369c73b06", "csrfChecked": true}
  ```
- Package-level validation passed before commit:
  ```bash
  go test ./examples/xgoja/19-express-keycloak-auth-host/cmd/host ./pkg/gojahttp/auth/sessionauth/sqlstore -count=1
  ```
- Pre-commit also passed lint and `go test ./...` for `e53d063`.

### What didn't work
- N/A. The first smoke after adding Postgres passed.

### What I learned
- The existing Keycloak smoke was easy to extend because the host already centralizes session construction.
- `sessionauth/sqlstore.ApplySchema` is useful for examples and smokes because it avoids adding migration tooling before the store package layout is settled.
- This smoke now gives us practical coverage for Postgres JSONB inserts/reads in `auth_sessions`, which the unit tests could not prove with SQLite alone.

### What was tricky to build
- The main boundary was deciding what the Postgres service owns. I kept Keycloak in dev/import mode and used Postgres for the Go host's app sessions, because the goal was validating the new auth store rather than productionizing Keycloak's own database.
- Another subtlety was preserving manual usability: `--session-db-dsn` is optional, so developers can still run the host with in-memory sessions if they do not want Postgres.
- The smoke needed an explicit Postgres readiness loop via `docker compose exec -T postgres pg_isready` before starting the host, otherwise the host could fail on `db.Ping()` during startup.

### What warrants a second pair of eyes
- Whether the example should use Postgres for Keycloak's internal database too, or whether keeping Keycloak `start-dev` is preferable for smoke speed.
- Whether `ApplySchema` in the example is acceptable until a formal migration mechanism exists.
- Whether `SESSION_DB_DSN` should be documented as the default for `serve` or only for `smoke`.

### What should be done in the future
- Add a targeted Postgres integration test for `sessionauth/sqlstore` that can run independently of the Keycloak smoke.
- Continue with `audit/sqlstore` and then extend the same compose stack to persist audit records.
- Later, wire appauth and capability SQL stores so task 25 can be completed honestly.

### Code review instructions
- Start with `examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml` to see the new Postgres service.
- Review `cmd/host/main.go`, especially `newSessionManager`, to verify fallback and SQL setup behavior.
- Review `scripts/smoke.sh` to confirm it waits for Postgres and passes `--session-db-dsn`.
- Validate with:
  ```bash
  go test ./examples/xgoja/19-express-keycloak-auth-host/cmd/host ./pkg/gojahttp/auth/sessionauth/sqlstore -count=1
  make -C examples/xgoja/19-express-keycloak-auth-host smoke
  ```

### Technical details
- Default smoke DSN:
  ```text
  postgres://goja:goja@127.0.0.1:15432/goja_auth?sslmode=disable
  ```
- Host switch:
  ```go
  if sessionDBDSN == "" {
      return sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true})
  }
  store, _ := sqlstore.New(sqlstore.Config{DB: db, Dialect: sqlstore.DialectPostgres})
  _ = store.ApplySchema(ctx)
  return sessionauth.New(sessionauth.Config{Store: store, AllowInsecureHTTP: true})
  ```


## Step 5: Add the SQL-backed audit store

This step implemented the second durable auth store adapter: `audit/sqlstore`. The store persists normalized, redacted audit records behind the existing `audit.Store` interface, so planned-route audit sinks can move from console logs or memory to queryable SQL without changing Express route declarations.

The package follows the same pattern as the SQL session store: a small `database/sql` subpackage, dialect-specific DDL for SQLite and Postgres, contract tests using the reusable audit store harness, and additional tests for redaction and operational queries.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue implementing `XGOJA-AUTH-STORES` from the previous checkpoint, using the next incomplete store phase.

**Inferred user intent:** Keep making concrete progress through the persistent auth store plan without stopping after the session store.

**Commit (code):** `8821692` — "Add SQL audit store"

### What I did
- Added package `pkg/gojahttp/auth/audit/sqlstore`.
- Added `sqlstore.Store`, implementing `audit.Store` with `database/sql`.
- Added SQLite and Postgres DDL in `schema.go`.
- Added `ApplySchema(ctx)` for tests/examples/simple migrations.
- Added `Snapshot(ctx)` for tests and examples.
- Added `QueryByOutcome(ctx, outcome, limit)` as a small executable query helper for denied/failed operational checks.
- Added README query examples for denied/failed outcomes, tenant denials, and resource write history.
- Added tests that cover:
  - reusable `audittest.RunStoreContract`,
  - normalized field persistence,
  - recursive redaction before insert when using `audit.Sink`,
  - `QueryByOutcome` ordering and limits,
  - Postgres placeholder/schema shape.
- Checked tasks 10-14 in `XGOJA-AUTH-STORES`.

### Why
- Persistent audit records are required for production auth operations: incident review, denied-route investigation, CSRF failures, and capability/session lifecycle events.
- Audit storage should remain behind `audit.Store`; route dispatch and Express route plans should only emit `gojahttp.AuditEvent` values through the host-owned sink.
- Redaction needs to be proven before records hit durable storage, because audit tables are long-lived and often broadly queryable by operators.

### What worked
- The audit SQL store passed its package tests:
  ```bash
  go test ./pkg/gojahttp/auth/audit/... -count=1
  ```
- The full auth package test set passed:
  ```bash
  go test ./pkg/gojahttp/auth/... -count=1
  ```
- The pre-commit hook for `8821692` passed lint and `go test ./...`.

### What didn't work
- N/A. The implementation and tests passed on the first package-level run.

### What I learned
- The insert-only `audit.Store` interface is sufficient for production writes, but tests and examples need a read path. `Snapshot(ctx)` and `QueryByOutcome` live on the concrete SQL store without widening the production `audit.Store` interface.
- Redaction belongs in `audit.Sink`/`Normalizer`, not the SQL store. The SQL test validates the intended path by recording through `audit.Sink` and then verifying the stored JSON contains `[REDACTED]` but not raw secrets.
- Operational query examples are valuable documentation because audit consumers will often query the table directly rather than through Go APIs.

### What was tricky to build
- The main design nuance was avoiding a too-large query API. I added only `Snapshot` for tests/examples and `QueryByOutcome` for executable denied/failed examples; production applications can still build their own reports from the table.
- Another subtle point was preserving the separation between normalization/redaction and persistence. The SQL store intentionally stores what it receives; the host should wire `audit.Sink{Store: store}` so normalization is always applied before insert.

### What warrants a second pair of eyes
- Whether `attributes_json` should be renamed to `attributes` before stabilizing the schema.
- Whether `QueryByOutcome` belongs in the package API or should remain documented SQL only.
- Whether audit insert failures should remain best-effort in route dispatch once this store is wired into the Keycloak smoke.
- Whether additional indexes are needed for high-cardinality fields such as `request_id` or `action`.

### What should be done in the future
- Wire `audit/sqlstore` into the Keycloak/Postgres smoke so route audit events are persisted end-to-end.
- Add retention/partitioning guidance in production deployment docs.
- Continue with `capability/sqlstore`, where atomic single-use redemption is the critical behavior.

### Code review instructions
- Start with `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go` for insert/query behavior.
- Review `pkg/gojahttp/auth/audit/sqlstore/schema.go` for table/index choices.
- Review `pkg/gojahttp/auth/audit/sqlstore/sqlstore_test.go` for redaction and query coverage.
- Review `pkg/gojahttp/auth/audit/sqlstore/README.md` for operator query examples.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/audit/... -count=1
  go test ./pkg/gojahttp/auth/... -count=1
  ```

### Technical details
- Basic wiring:
  ```go
  store, err := sqlstore.New(sqlstore.Config{DB: db, Dialect: sqlstore.DialectPostgres})
  sink := audit.Sink{Store: store}
  host := gojahttp.NewHost(gojahttp.HostOptions{
      Auth: gojahttp.AuthOptions{Audit: sink},
  })
  ```
- Denied outcome query helper:
  ```go
  denied, err := store.QueryByOutcome(ctx, "denied", 100)
  ```
