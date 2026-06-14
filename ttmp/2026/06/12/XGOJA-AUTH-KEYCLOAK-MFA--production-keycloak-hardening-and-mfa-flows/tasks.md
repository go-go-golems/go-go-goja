# Tasks

## Phase 1 — Production Keycloak settings and docs

- [ ] Document production Keycloak client settings for Authorization Code + PKCE.
- [ ] Document HTTPS redirect URI and web origin requirements.
- [ ] Document secure cookie settings and reverse-proxy assumptions.
- [ ] Document local example settings that must not be copied to production.

## Phase 2 — Durable OIDC transaction store

- [ ] Design SQL schema for `keycloakauth.TransactionStore`.
- [ ] Implement durable/shared transaction store.
- [ ] Ensure `Take` is one-time and replay-safe.
- [ ] Add TTL expiry and cleanup behavior.
- [ ] Test missing, expired, replayed, and valid transactions.

## Phase 3 — Logout hardening

- [ ] Decide whether Keycloak end-session support is required for this package.
- [ ] Keep app-session revocation mandatory.
- [ ] Add optional IdP logout/end-session redirect if accepted.
- [ ] Add tests for app logout and optional IdP logout URL behavior.
- [ ] Document front-channel/back-channel logout limitations.

## Phase 4 — MFA session update primitive

- [ ] Design API for marking MFA complete on a session.
- [ ] Extend `sessionauth.Store` or add an optional narrower interface for `MFAAt` updates.
- [ ] Implement `sessionauth.Manager` helper for updating `Session.MFAAt`.
- [ ] Test route denial before MFA update and success after update.
- [ ] Decide whether MFA completion should rotate session IDs.

## Phase 5 — MFA example flow

- [ ] Add a demo app-owned MFA challenge/verify endpoint to a host example.
- [ ] Add a route using `express.user().required().mfaFresh("10m")`.
- [ ] Add smoke coverage for stale/non-MFA denial.
- [ ] Add smoke coverage for MFA completion and route success.
- [ ] Emit or document audit events for MFA challenge and completion.

## Phase 6 — Production smoke and handoff

- [ ] Extend the Keycloak smoke or add a dedicated production MFA smoke.
- [ ] Validate with Keycloak + app session + CSRF + MFA freshness.
- [ ] Update Glazed help or production deployment docs.
- [ ] Run docmgr doctor and package/example tests.
