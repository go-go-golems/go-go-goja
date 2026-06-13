# Tasks

## Phase 1 — Store contracts

- [ ] Extract reusable contract tests for `sessionauth.Store`.
- [ ] Extract reusable contract tests for `audit.Store`.
- [ ] Extract reusable contract tests for `capability.Store`.
- [ ] Extract reusable contract tests for `appauth` user, membership, and resource stores.

## Phase 2 — Session persistence

- [ ] Add SQL/Postgres schema for sessions.
- [ ] Implement `sessionauth` SQL store.
- [ ] Test create/get/touch/revoke/rotate behavior.
- [ ] Test idle expiry, absolute expiry, revocation, and MFA timestamp persistence.
- [ ] Document production cookie settings and store requirements.

## Phase 3 — Audit persistence

- [ ] Add SQL/Postgres schema for audit records.
- [ ] Implement `audit.Store` SQL adapter.
- [ ] Test normalized field persistence.
- [ ] Test recursive redaction before insert.
- [ ] Add query examples for denied/failed route outcomes.

## Phase 4 — Capability persistence

- [ ] Add SQL/Postgres schema for capabilities.
- [ ] Implement `capability.Store` SQL adapter.
- [ ] Ensure raw capability tokens are never stored.
- [ ] Test expiry, revocation, wrong purpose, and not found behavior.
- [ ] Add concurrency test proving single-use redemption is atomic.

## Phase 5 — App auth persistence

- [ ] Add starter SQL schema for users, tenants, memberships, and resources.
- [ ] Implement app-owned `UserStore`, `MembershipStore`, and `ResourceStore` adapters.
- [ ] Test `UpsertFromOIDC` and Keycloak subject mapping.
- [ ] Test deny-by-default authorization behavior against SQL-backed memberships.

## Phase 6 — Example integration and docs

- [ ] Add Docker Compose Postgres wiring for the production auth example or a new store-focused example.
- [ ] Wire Keycloak + Postgres + sessionauth + appauth + audit + capability in a smoke target.
- [ ] Add a deployment guide covering migrations, cookie security, backups, and operational queries.
- [ ] Validate with package tests and example smokes.
