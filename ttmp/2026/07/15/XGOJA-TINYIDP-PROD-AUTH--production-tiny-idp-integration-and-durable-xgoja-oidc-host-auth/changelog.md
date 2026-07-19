---
# Changelog

## 2026-07-15

- Initial workspace created
- Added the production OIDC host-auth design, phased implementation plan, and evidence diary. The design identifies durable single-use OIDC transactions as the first required correctness change and separates application-owned device credentials from future native tiny-idp resource-server support.
- Validated the ticket successfully. The requested reMarkable bundle upload was blocked by the environment's external-transfer policy; the exact blocker and prepared bundle are recorded in the diary.
- Uploaded the approved documentation bundle successfully to `/ai/2026/07/15/XGOJA-TINYIDP-PROD-AUTH`.
- Completed Phase 0 and Phase 1 (commit `2d15d1d`): added strict tiny-idp production fixture tooling, a durable SQLite/PostgreSQL OIDC transaction store, atomic one-use callback consumption, generated-host configuration/Glazed wiring, and focused/race/full-suite coverage. The fixture's agent-runner network limitation is documented in the diary; a direct host-namespace TLS readiness probe passed.

## 2026-07-15

Completed intern-facing design guide and diary; related the OIDC handler, host builder/config, session storage, and programauth implementation seams.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/keycloakauth/keycloakauth.go — OIDC transaction design evidence

## 2026-07-15

Phase 2: enforce the single-node hostauth deployment contract, explicit memory rate-limit limitation, and safe /auth/readyz topology report (commit 543831b).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/preflight.go — Fail-closed production validation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/readiness.go — Safe readiness report

## 2026-07-15

Phase 3: complete the strict tiny-idp personal-inbox reference app with application-owned device refresh/revoke, CLI verbs, Python isolation smoke, and Playwright browser coverage.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/Makefile — Strict smoke target
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/device_handlers.go — Device credential lifecycle

## 2026-07-15

Phase 4: made shared-SQL OAuth access/refresh pair persistence atomic; added safe OIDC, device, refresh, revoke, logout, and rate-limit audit/metric observations; added retention/redaction operations guidance and focused negative security tests.
