# Tasks

## TODO

### Phase 0 — Freeze contracts and establish a production test boundary

- [x] Confirm whether the first production deployment target is one durable process or multiple replicas; record the supported topology in hostauth configuration documentation.
- [x] Define the generic OIDC terminology migration (`keycloakauth` and `keycloakSub` to issuer-neutral names) as an intentional API change, with no compatibility adapter unless explicitly approved.
- [x] Add a strict tiny-idp integration fixture with a registered callback URI, post-logout redirect URI, seeded user, and TLS configuration.
- [x] Write a test matrix covering login success, callback replay, nonce mismatch, missing transaction, expired transaction, restart during login, logout, and redirect validation.

### Phase 1 — Implement durable OIDC authorization transactions

- [x] Define a storage-neutral transaction-store package contract with atomic single-use retrieval and expiry semantics.
- [x] Implement SQL schema, migrations, and a SQL transaction store for SQLite and PostgreSQL.
- [x] Make `Take` atomically claim/delete only an unexpired row and return a stable not-found/expired outcome without exposing security material.
- [x] Wire the store through `hostauth.StoresConfig` and `BuildHostAuthServices`; remove the implicit memory default in production mode.
- [x] Add unit, race, expiry, double-callback, concurrent-callback, and restart tests.

### Phase 2 — Make production configuration explicit and reject unsafe combinations

- [ ] Add a hostauth production validation/preflight API that rejects memory stores, insecure HTTP, invalid public URLs, and automatic production schema application.
- [ ] Define cookie, reverse-proxy, key management, database migration, audit retention, and readiness requirements in a deployment reference.
- [ ] Make rate-limiter selection configurable; support either a distributed implementation or an explicit one-replica mode with a documented limitation.
- [ ] Add a machine-readable readiness report that identifies the configured auth topology without leaking secrets.

### Phase 3 — Deliver the tiny-idp production example

- [ ] Create an xgoja application with a small UI, JSON API, SQLite/PostgreSQL application storage, and explicit domain actions.
- [ ] Configure tiny-idp as the OIDC issuer and demonstrate registered redirect and post-logout URIs.
- [ ] Implement application-owned device start, approval UI, polling, refresh, revocation, and CLI API calls.
- [ ] Add Playwright/browser and command-line smoke tests against the strict tiny-idp fixture.
- [ ] Publish an operator runbook and a developer tutorial that explain the two authorization layers.

### Phase 4 — Strengthen application credential lifecycle and observability

- [ ] Review `programauth` token-family persistence and replace compensating cleanup with one cross-table transaction if its stores share a SQL database.
- [ ] Add structured audit events and metrics for OIDC transaction outcomes, device-flow states, token rotation, rate-limit decisions, and logout.
- [ ] Define retention, redaction, and incident-debugging queries; verify no state, nonce, verifier, authorization code, or bearer token reaches logs.
- [ ] Add negative tests for account crossover, grant escalation, replay, refresh reuse, and device-code enumeration.

### Phase 5 — Separately design native tiny-idp device-token acceptance

- [ ] Specify whether tiny-idp should expose OAuth introspection, JWT access tokens, or another formal resource-server validation contract.
- [ ] Design an `oidcresource` authenticator for issuer, audience, scope, DPoP, cache, revocation, and claims-to-actor mapping.
- [ ] Prove its semantics against tiny-idp with integration tests before exposing it in generated hosts.
- [ ] Keep this optional resource-server capability separate from xgoja application-owned `programauth` device credentials.
