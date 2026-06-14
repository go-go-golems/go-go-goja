# Tasks

## Phase 0 — Ticket setup and design

- [x] Create docmgr ticket workspace
- [x] Add design document for SQL appauth store
- [x] Add initial diary entry and relate files
- [ ] Commit ticket setup/design

## Phase 1 — SQL store implementation

- [ ] Add `pkg/gojahttp/auth/appauth/sqlstore` package
- [ ] Add SQLite/Postgres schemas for app users, tenants, memberships, and resources
- [ ] Implement `New`, `Schema`, `ApplySchema`, and seeding helpers
- [ ] Implement `appauth.UserStore` methods including disabled-user OIDC upsert behavior
- [ ] Implement `MembershipStore` methods with revoked-membership filtering
- [ ] Implement `ResourceStore` methods with JSON claims clone isolation
- [ ] Add SQLite contract tests via `appauthtest.RunStoreContract`
- [ ] Add generated logcopter stub if required by lint
- [ ] Run `go test ./pkg/gojahttp/auth/appauth/... -count=1`
- [ ] Commit appauth SQL store

## Phase 2 — Demo and docs integration

- [ ] Replace Keycloak demo in-memory appauth store with SQL-backed appauth store when DSN is configured
- [ ] Seed demo users/memberships/resources into SQL
- [ ] Update Keycloak smoke to verify SQL appauth rows or behavior
- [ ] Update Keycloak demo README with appauth persistence notes
- [ ] Run Keycloak smoke and targeted package tests
- [ ] Update diary/changelog and commit demo/docs integration
