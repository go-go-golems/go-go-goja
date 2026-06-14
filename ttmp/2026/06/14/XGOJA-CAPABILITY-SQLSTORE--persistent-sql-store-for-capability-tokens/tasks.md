# Tasks

## Phase 0 — Ticket setup and design

- [x] Create docmgr ticket workspace
- [x] Add design document for SQL capability store
- [x] Add initial diary entry and relate files
- [ ] Commit ticket setup/design

## Phase 1 — SQL store implementation

- [ ] Add `pkg/gojahttp/auth/capability/sqlstore` package
- [ ] Add SQLite/Postgres schemas for `auth_capabilities`
- [ ] Implement `New`, `Schema`, `ApplySchema`, `Create`, `ByID`, `Redeem`, and `Revoke`
- [ ] Implement transaction/conditional-update based atomic single-use redemption
- [ ] Add SQLite contract tests via `capabilitytest.RunStoreContract`
- [ ] Add generated logcopter stub if required by lint
- [ ] Run `go test ./pkg/gojahttp/auth/capability/... -count=1`
- [ ] Commit capability SQL store

## Phase 2 — Demo and docs integration

- [ ] Wire Keycloak demo to create/use capability SQL store when SQL DSN is configured
- [ ] Add demo invite issue/accept routes or smoke-only flow using persisted capabilities
- [ ] Update Keycloak demo README with capability persistence notes
- [ ] Run Keycloak smoke and targeted package tests
- [ ] Update diary/changelog and commit demo/docs integration
