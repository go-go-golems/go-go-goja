# Tasks

## Phase 0 — Ticket setup and design

- [x] Create docmgr ticket workspace
- [x] Add design document for SQL capability store
- [x] Add initial diary entry and relate files
- [x] Commit ticket setup/design

## Phase 1 — SQL store implementation

- [x] Add `pkg/gojahttp/auth/capability/sqlstore` package
- [x] Add SQLite/Postgres schemas for `auth_capabilities`
- [x] Implement `New`, `Schema`, `ApplySchema`, `Create`, `ByID`, `Redeem`, and `Revoke`
- [x] Implement transaction/conditional-update based atomic single-use redemption
- [x] Add SQLite contract tests via `capabilitytest.RunStoreContract`
- [x] Add generated logcopter stub if required by lint
- [x] Run `go test ./pkg/gojahttp/auth/capability/... -count=1`
- [x] Commit capability SQL store

## Phase 2 — Demo and docs integration

- [x] Wire Keycloak demo to create/use capability SQL store when SQL DSN is configured
- [x] Add demo invite issue/accept routes or smoke-only flow using persisted capabilities
- [x] Update Keycloak demo README with capability persistence notes
- [x] Run Keycloak smoke and targeted package tests
- [x] Update diary/changelog and commit demo/docs integration
