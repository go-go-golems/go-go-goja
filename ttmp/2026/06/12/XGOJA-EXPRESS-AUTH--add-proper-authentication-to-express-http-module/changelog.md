# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Created ticket, imported preliminary auth API ideas, and wrote MVP Express auth design/implementation guide

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Chronological investigation diary
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/01-auth-preliminary-api-ideas.md — Imported source analysis


## 2026-06-12

Validated ticket docs and uploaded XGOJA EXPRESS AUTH MVP Design bundle to reMarkable

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md — Fixed Mermaid label and included in uploaded bundle
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Updated diary with validation and upload outcome


## 2026-06-12

Added Express-style middleware/router auth alternative design with Go-owned security middleware and strict route coverage validation

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/02-express-style-middleware-auth-design-and-implementation-guide.md — New alternative design and implementation guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/index.md — Updated ticket overview and links
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Recorded Step 3 for the middleware/router design


## 2026-06-12

Uploaded updated XGOJA EXPRESS AUTH Design Options bundle including staged and middleware auth designs

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md — Included as companion design in updated bundle
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/02-express-style-middleware-auth-design-and-implementation-guide.md — Included in updated reMarkable bundle
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Updated with middleware design and upload notes


## 2026-06-12

Selected Go-backed fluent staged route builders and expanded implementation tasks into detailed phases

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Recorded implementation kickoff and selected direction
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/tasks.md — Detailed phased implementation task list


## 2026-06-12

Phase 1: added gojahttp RoutePlan model, auth interfaces, planned route registration, and validation tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go — New typed route-plan and auth service model
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan_test.go — Validation and planned registration tests
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/host.go — HostOptions Auth and RegisterPlanned support
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/route_registry.go — Route Plan field and AddPlanned support
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Recorded Phase 1 implementation


## 2026-06-12

Phase 2: added planned route dispatch, secure context, auth/resource/authorization enforcement, and host tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/host.go — Branches to planned dispatch for routes with RoutePlan
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/planned_dispatch.go — Planned route auth/resource/authorization dispatch and secure JS context
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/planned_dispatch_test.go — Host-level planned dispatch integration tests
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Recorded Phase 2 implementation and test failure/fix


## 2026-06-12

Phase 3: added Express Go-backed fluent route builders with strict auth/resource spec validation and integration tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go — Go-backed staged builder and spec object implementation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders_integration_test.go — Express builder integration tests
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/express.go — Exports user/resource builders and app.route
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Recorded Phase 3 implementation


## 2026-06-12

Phase 4: documented planned auth builders in TypeScript declarations and Express module docs

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/typescript.go — Updated generated TypeScript declarations for planned route builders and context
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/18-express-module.md — Documented planned auth route usage and host auth setup
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Recorded Phase 4 docs/types update


## 2026-06-12

Phase 5: added provider planned-route coverage, example route script, and final validation

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/15-express-planned-auth/README.md — Example notes and host auth caveat
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/15-express-planned-auth/scripts/server.js — Example planned auth route declarations
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/http_test.go — Provider test for planned public route registration through external host service
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Recorded final validation including VCS stamping workaround


## 2026-06-12

Marked phase commit bookkeeping tasks complete after implementation commits 99a2da3, e19ea0d, 3b1220f, and 13d4675

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/tasks.md — Phase commit bookkeeping updated


## 2026-06-12

Updated MVP design to keep .get/.post/... names but hard-cut them over to explicit planned routes requiring .public() or auth before .handle().

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md — Primary design updated for verb-helper hard cutover
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Diary records the hard-cutover decision update


## 2026-06-12

Implemented hard cutover for Express verb helpers: .get/.post/... now return planned builders, old handler overloads reject with migration guidance, tests/examples/docs migrated (commit 4492723).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders_integration_test.go — Regression coverage
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/express.go — Runtime API cutover
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/typescript.go — TypeScript API cutover
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/18-express-module.md — Docs migration guidance
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Implementation diary Step 12


## 2026-06-12

Added dedicated Glazed help docs for the Express auth framework and migration from raw verb handlers to planned auth routes (commit de09c15).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/18-express-module.md — Reference cross-links and troubleshooting
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/29-express-auth-user-guide.md — Express auth user guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/30-migrate-express-apps-to-planned-auth.md — Migration tutorial
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Diary records documentation validation and next step


## 2026-06-12

Added HostOptions.RejectRawRoutes strict mode plus planned route descriptor metadata, with docs/tests (commit 4f42a55).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/29-express-auth-user-guide.md — User guide strict-mode documentation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/host.go — Strict raw-route runtime enforcement
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/planned_dispatch_test.go — Strict-mode coverage
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/route_registry.go — Route diagnostics metadata
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Diary records Step 14


## 2026-06-12

Implemented planned route CSRF and audit hooks with Go host interfaces, builder methods, tests, examples, and docs (commit 61c858d).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go — JS builder API
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go — Plan model and interfaces
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/planned_dispatch.go — Runtime enforcement/emission
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/planned_dispatch_test.go — Regression coverage
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md — Diary records Step 15

