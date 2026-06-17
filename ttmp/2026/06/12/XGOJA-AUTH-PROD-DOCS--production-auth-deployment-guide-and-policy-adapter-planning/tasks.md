# Tasks

## Production deployment guide

- [ ] Write Glazed help/docs page for production planned-auth deployment.
- [ ] Cover deployment topology: browser, reverse proxy/TLS, Go host, Keycloak, persistent stores.
- [ ] Document Keycloak realm/client settings and redirect URI/web origin rules.
- [ ] Document app session cookie settings, expiry, revocation, and rotation.
- [ ] Document CSRF expectations for unsafe browser/session routes.
- [ ] Document audit redaction, retention, and operational query examples.
- [ ] Document persistent store migration and backup expectations.
- [ ] Add a production readiness checklist.

## Policy adapter planning

- [ ] Evaluate Casbin as an optional `gojahttp.Authorizer` adapter.
- [ ] Evaluate OpenFGA as an optional relationship-based authorization adapter.
- [ ] Evaluate OPA as an optional policy-decision adapter.
- [ ] Document why `appauth.Authorizer` remains the default until persistent stores are proven.
- [ ] Identify adapter test fixtures based on planned route action/resource inputs.

## Example and docs process cleanup

- [ ] Add an xgoja example-numbering rule/checklist.
- [ ] Add branch merge checklist for `examples/xgoja` renumbering.
- [ ] Document TypeScript provider API regeneration after provider changes.
- [ ] Clarify split between generated-binary examples and host integration examples.
- [ ] Link the process note from relevant auth/xgoja docs.
