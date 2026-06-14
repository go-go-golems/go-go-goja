# Changelog

## 2026-06-14

- Initial workspace created


## 2026-06-14

Created generated-host auth config ticket, wrote detailed design, transferred backlog from XGOJA-HTTP-AUTH-CONFIG, and added phased task plan

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/14/XGOJA-GENERATED-HOST-AUTH-CONFIG--design-generated-host-auth-session-and-store-configuration/design-doc/01-generated-host-auth-session-and-store-configuration-design.md — Primary generated-host auth config design
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/14/XGOJA-GENERATED-HOST-AUTH-CONFIG--design-generated-host-auth-session-and-store-configuration/reference/01-implementation-diary.md — Initial design handoff diary
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/14/XGOJA-GENERATED-HOST-AUTH-CONFIG--design-generated-host-auth-session-and-store-configuration/tasks.md — Detailed phased task backlog


## 2026-06-14

Implemented initial hostauth package skeleton, config resolver, host-service lookup helpers, and unit tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/config.go — Config and resolved config types
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/lookup.go — Host service lookup helpers
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/lookup_test.go — Lookup tests
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/resolve.go — Config resolution
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/resolve_test.go — Resolver tests
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/services.go — Service keys and service payload contracts


## 2026-06-14

Step 2: added hostauth config skeleton and resolver (commit 2dee4df)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/config.go — Config and resolved config types (commit 2dee4df)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/logcopter.go — Generated package logcopter file (commit 2dee4df)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/lookup.go — Host service lookup helpers (commit 2dee4df)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/resolve.go — Config resolution and store inheritance (commit 2dee4df)


## 2026-06-14

Step 3: added hostauth store builders for memory, SQLite, and Postgres-backed auth stores (commit cc32556)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/stores.go — Store builder implementation (commit cc32556)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/stores_test.go — Store builder tests (commit cc32556)


## 2026-06-14

Step 4: added hostauth session manager builder, auth-options wiring, and lazy service factory (commit 5276bfb)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — Service factory implementation (commit 5276bfb)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder_test.go — Service factory tests (commit 5276bfb)
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/services.go — Services.Close lifecycle helper (commit 5276bfb)


## 2026-06-14

Step 5: wired hostauth service factories into HTTP serve and hot reload paths, preserving no-auth and external-host behavior (commit addd553)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve.go — HTTP serve integration for hostauth service factories
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve_test.go — HTTP serve and hot reload hostauth integration tests


## 2026-06-14

Pushed Step 5 HTTP hostauth integration and diary commits to task/goja-express-auth (commits addd553, 0ff8150)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve.go — Pushed implementation commit addd553

