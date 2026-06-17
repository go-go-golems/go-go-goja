# Tasks

## Phase 0 — Ticket setup and design

- [x] Create docmgr ticket workspace
- [x] Add design document for SQL appauth store
- [x] Add initial diary entry and relate files
- [x] Commit ticket setup/design

## Phase 1 — SQL store implementation

- [x] Add `pkg/gojahttp/auth/appauth/sqlstore` package
- [x] Add SQLite/Postgres schemas for app users, tenants, memberships, and resources
- [x] Implement `New`, `Schema`, `ApplySchema`, and seeding helpers
- [x] Implement `appauth.UserStore` methods including disabled-user OIDC upsert behavior
- [x] Implement `MembershipStore` methods with revoked-membership filtering
- [x] Implement `ResourceStore` methods with JSON claims clone isolation
- [x] Add SQLite contract tests via `appauthtest.RunStoreContract`
- [x] Add generated logcopter stub if required by lint
- [x] Run `go test ./pkg/gojahttp/auth/appauth/... -count=1`
- [x] Commit appauth SQL store

## Phase 2 — Demo and docs integration

- [x] Replace Keycloak demo in-memory appauth store with SQL-backed appauth store when DSN is configured
- [x] Seed demo users/memberships/resources into SQL
- [x] Update Keycloak smoke to verify SQL appauth rows or behavior
- [x] Update Keycloak demo README with appauth persistence notes
- [x] Run Keycloak smoke and targeted package tests
- [x] Update diary/changelog and commit demo/docs integration
