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


## 2026-06-20

Implemented Phase 3 typed grants and first-class programmatic agent model with in-memory store/service and tests (commit 5800dd7).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/agent.go — Agent model
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/memory_store.go — Concurrency-safe in-memory AgentStore for tests/dev generated hosts (commit 5800dd7)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go — AuthResult now carries typed GrantSet in addition to scope strings (commit 5800dd7)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/grants.go — Typed Grant and GrantSet normalization/matching/scope serialization (commit 5800dd7)


## 2026-06-20

Committed generated programauth log metadata emitted by repository generation hooks (commit 5412cc6).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/logcopter.go — Generated package log metadata for new programauth package (commit 5412cc6)


## 2026-06-20

Implemented Phase 4 API-token issue/list/revoke/authenticate, bearer parsing, composite bearer/session auth, and grant enforcement for planned routes (commit 00a1e86).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/api_token_integration_test.go — Planned-route API-token integration coverage for CSRF skip and grant denial (commit 00a1e86)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/composite.go — Composite bearer-first/session-fallback authenticator (commit 00a1e86)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/memory_token_store.go — In-memory API-token store with prefix lookup/revoke/touch/list behavior (commit 00a1e86)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/token.go — API-token model/service/hasher/bearer parser/authentication implementation (commit 00a1e86)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/enforcer.go — Enforces AuthResult grant intersection before app authorizer (commit 00a1e86)


## 2026-06-20

Phase 5: wired generated hostauth services with programauth stores/composite bearer auth and added fluent auth.grants/auth.agents/auth.tokens JavaScript builders (commit 432b628).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — Builds memory programauth stores and composite bearer/session auth for generated hostauth services
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/hostauth/hostauth_test.go — Covers auth.agents/auth.tokens issuance/list/revoke behavior from JavaScript
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/hostauth/programmatic.go — Exposes Go-owned fluent JavaScript builders for grants, agents, and API tokens


## 2026-06-20

Phase 6: added route auth requirements and Go/Express builders for agent, session-user, and anyOf restrictions (commit 84d9e3c).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go — Adds express.agent, express.sessionUser, and express.anyOf auth builders
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/typescript.go — Declares TypeScript support for route auth restriction builders
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go — Defines AuthRequirement route constraints and validates allowed auth methods/principal kinds
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/enforcer.go — Enforces route auth requirements after authentication and before CSRF/resource/authorization work


## 2026-06-20

Phase 7: Added access-token and rotating refresh-token family service with memory stores, access-token bearer auth, refresh reuse family revocation, and concurrency tests (commit 730b4dd).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/memory_oauth_token_store.go — Memory token-family stores
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/oauth_token.go — Access/refresh token family service
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/oauth_token_test.go — Token-family behavior tests


## 2026-06-20

Phase 8: implemented device authorization service, memory store, native start/token/approve handlers, generated hostauth wiring, and tests (commit 4758e78).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/device.go — Device authorization service
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/device_handlers.go — Native device handlers
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — Generated hostauth device service wiring


## 2026-06-20

Phase 9: added device authorization help docs, smoke coverage for generated native device endpoints, checked all tasks, and uploaded final reMarkable bundle to /ai/2026/06/20/XGOJA-PROGRAMMATIC-AUTH-DESIGN (commit a32d3eb).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/doc/28-device-authorization-programmatic-access.md — Device authorization help page
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/22-programmatic-agent-auth/README.md — Example documentation updated with device endpoints
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/22-programmatic-agent-auth/scripts/smoke.sh — Generated-host smoke coverage for device start/pending poll


## 2026-06-20

Ticket closed


## 2026-06-21

Phase 10: reopened ticket, added production-hardening tasks, and enforced device approval grant intersection with regression tests (commit 01615c9).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/device.go — Device approval grant narrowing
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/grants.go — GrantSet intersection


## 2026-06-21

Phase 11A: added SQL-backed programauth store schema and transaction contract design.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design-doc/01-sql-backed-programauth-stores-design.md — SQL store design

