# Tasks

## TODO

- [x] Task 1: Create the HTTP auth provider configuration design and implementation guide.
- [x] Task 2: Implement first-slice HTTP provider config fields (`dev-errors`, `reject-raw-routes`) with xgoja module config support.
- [x] Task 3: Add provider tests for static config, Glazed override mapping, and internal-host behavior.
- [x] Task 4: Update xgoja/HTTP documentation with the new provider config shape and production-auth boundary.
- [x] Task 5: Run targeted validation and record diary/changelog/bookkeeping.

## Transferred backlog: next phase (`auth.session` + `auth.stores` generated-host config)

The next-phase backlog has moved to `XGOJA-GENERATED-HOST-AUTH-CONFIG` so this ticket can close as the completed first-slice HTTP provider configuration work.

- [x] Task 6: Create a follow-up design/ticket for generated-host auth config rather than extending JavaScript import-time Express config. Transferred to `XGOJA-GENERATED-HOST-AUTH-CONFIG`.
- [x] Task 7: Define config structs/schema for `auth.mode`, `auth.session.cookie`, session timeouts, and `auth.stores.default` inheritance. Transferred to `XGOJA-GENERATED-HOST-AUTH-CONFIG` tasks 11-37.
- [x] Task 8: Implement parsing helpers for `same-site`, Go durations, secure-cookie defaults, and store inheritance/overrides. Transferred to `XGOJA-GENERATED-HOST-AUTH-CONFIG` tasks 18-37.
- [x] Task 9: Add a session manager builder that maps config into `sessionauth.Config` without weakening secure defaults. Transferred to `XGOJA-GENERATED-HOST-AUTH-CONFIG` tasks 50-58.
- [x] Task 10: Add store-builder skeletons for `memory`, `sqlite`, and `postgres`, initially wiring session and audit stores. Transferred to `XGOJA-GENERATED-HOST-AUTH-CONFIG` tasks 38-49.
- [x] Task 11: Use the new `CommandSetContext.Host` support in the follow-up design: command providers should be able to inspect contributed host services while building `serve`/auth commands. Transferred to `XGOJA-GENERATED-HOST-AUTH-CONFIG` tasks 59-76.
- [x] Task 12: Decide whether host wiring belongs in a generated-host template, a sibling auth provider, or an HTTP provider host-service contribution. Transferred to `XGOJA-GENERATED-HOST-AUTH-CONFIG` tasks 6-10 and 83-90.
- [x] Task 13: Defer OIDC/Keycloak config until session cookie hardening, persistent stores, and transaction-store semantics are stable. Transferred to `XGOJA-GENERATED-HOST-AUTH-CONFIG` tasks 107-111.
