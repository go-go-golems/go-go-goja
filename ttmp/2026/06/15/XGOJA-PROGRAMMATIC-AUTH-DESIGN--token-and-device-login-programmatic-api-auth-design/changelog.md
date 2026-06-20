# Changelog

## 2026-06-15

- Initial workspace created


## 2026-06-15

Created detailed token/device-login programmatic API auth implementation guide and diary.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/01-token-and-device-login-programmatic-api-auth-implementation-guide.md — Primary implementation guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/reference/01-implementation-diary.md — Diary for the design ticket


## 2026-06-15

Uploaded XGOJA Programmatic Auth Design bundle to reMarkable at /ai/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/01-token-and-device-login-programmatic-api-auth-implementation-guide.md — Uploaded as primary bundle document


## 2026-06-18

Researched OWASP/IETF/NIST/GitHub best practices and added a deep review/design for opinionated Go-owned fluent JavaScript programmatic-agent auth APIs.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md — Primary best-practice review and revised API design
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/reference/01-implementation-diary.md — Diary updated with research/review step
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/sources/02-owasp-api-security-top-10-2023.md — Downloaded OWASP API Security overview source
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/sources/11-ietf-rfc9700-oauth-security-bcp.md — Downloaded OAuth security BCP source


## 2026-06-18

Uploaded revised programmatic auth best-practice review bundle to reMarkable at /ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN after fixing Mermaid rendering.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md — Uploaded as primary revised review document
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/reference/01-implementation-diary.md — Included in uploaded bundle


## 2026-06-20

Promoted rate limiting to a first-class planned-route primitive in the programmatic auth review, including .rateLimit builder APIs, RateLimitSpec, pre/post-auth enforcer stages, invariants, tests, and decision record.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go — Future JavaScript builder integration point for express.rateLimit and route .rateLimit
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/enforcer.go — Future pre-auth and post-auth rate-limit enforcement point
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md — Updated with route-level rate limiting primitive design
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/reference/01-implementation-diary.md — Diary Step 3 records rate-limit design update


## 2026-06-20

Re-uploaded the revised programmatic auth review bundle with the route-level rate limiting update to reMarkable at /ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md — Updated bundle source


## 2026-06-20

Implemented Phase 1 rate limiting: RateLimitSpec/RateLimiter core, fixed-window memory limiter, pre/post planned-route enforcement, Go and Express builders, generated-host wiring, tests, and commit 1486dbb.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go — Exposes Go-owned express.rateLimit fluent builders and route .rateLimit methods (commit 1486dbb)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/enforcer.go — Runs pre-auth and post-auth route limit checks and maps exhausted budgets to 429 responses (commit 1486dbb)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/ratelimit.go — Core rate-limit model
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/services.go — Provides a default in-memory limiter for generated hostauth services (commit 1486dbb)


## 2026-06-20

Implemented Phase 2 AuthResult plumbing and redacted ctx.auth projection for planned routes (commit 1add4b5).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/typescript.go — Adds AuthInfo to planned Express TypeScript declarations (commit 1add4b5)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go — Defines AuthMethod
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/enforcer.go — Prefers ResultAuthenticator
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/planned_dispatch.go — Adds Auth to SecureContext and exposes safe ctx.auth to JavaScript (commit 1add4b5)

