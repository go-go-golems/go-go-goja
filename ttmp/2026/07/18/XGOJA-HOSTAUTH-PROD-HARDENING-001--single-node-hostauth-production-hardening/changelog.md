# Changelog

## 2026-07-18

- Initial workspace created


## 2026-07-18

Created evidence-backed intern implementation guide and investigation diary for scoped single-node hostauth hardening.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/device_handlers.go — Native device boundary documented
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — Composition boundary documented


## 2026-07-18

Validated ticket documentation, added the operations vocabulary topic, and uploaded the design-and-diary PDF bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/07/18/XGOJA-HOSTAUTH-PROD-HARDENING-001--single-node-hostauth-production-hardening/reference/01-investigation-diary.md — Records validation and reMarkable delivery


## 2026-07-18

Reframed the plan as measured complete product slices and added an ordered six-phase execution breakdown.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/07/18/XGOJA-HOSTAUTH-PROD-HARDENING-001--single-node-hostauth-production-hardening/design-doc/01-intern-implementation-guide-for-single-node-hostauth-hardening.md — Defines revised scope and phase exit criteria


## 2026-07-18

Phase 1: added canonical request-identity resolver primitives and migrated DTO, audit, limiter, and access-log consumers (commit 3b3b448).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/request_identity.go — Resolves trusted proxy identity


## 2026-07-18

Phase 1: wired validated hostauth proxy policy around the full generated ServeMux (commit 30bef69).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve.go — Native and planned routes now share request identity


## 2026-07-18

Phase 2: added redacted device request inspection and CSRF-protected terminal denial (commit 831887c).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/device_handlers.go — Native inspection and denial handlers


## 2026-07-18

Phase 2: completed device policy, native budgets, inspect/deny, owner-scoped agent management, and configuration wiring; refresh-family browser management remains separately designed.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/device_handlers.go — Phase 2 HTTP security boundary


## 2026-07-18

Phase 3: added dependency-aware readiness, independent liveness, and outage/recovery coverage (commits c2172ec and HEAD).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/readiness.go — Bounded dynamic SQL readiness


## 2026-07-18

Phase 4: implemented typed Express OAuth route declarations, fail-closed validation, exclusive external bearer verifier boundary, and redacted context/audit metadata.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go — OAuth route security contract


## 2026-07-18

Phase 5: corrected the single-node deployment runbook and captured race/build/lint validation evidence.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/doc/23-auth-host-production-runbook.md — Production deployment procedure


## 2026-07-18

Completed OAuth host composition, issuer-scoped identity mapping, user disablement, refresh-family lifecycle operations, credential-retention maintenance, metrics hooks, and the Express/TinyIDP example (commit d37a6b3). Repository pre-commit lint, generation, vet, and full Go tests passed.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go — Express OAuth route builder integration
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/maintenance.go — Credential retention maintenance service
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/oauth_verifier.go — OAuth verifier profiles and identity composition


## 2026-07-18

Security scan follow-up: replaced inline SQL placeholder concatenation with static SQLite/Postgres query constants for external identities and user disablement (commit 94241a2); gosec reports zero issues.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go — Parameterized dialect-specific appauth SQL


## 2026-07-18

CI follow-up: updated generated-host OIDC smoke to assert the current /healthz liveness contract (commit afef4da); local make oidc-smoke passes.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/Makefile — Smoke contract fix


## 2026-07-18

Addressed all PR #98 Codex review findings: OIDC transaction cleanup, honest logout failure handling, disabled-user enforcement across planned and native OIDC routes, and valid proxy/deployment CLI documentation (commit c9f227e).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/maintenance.go — Retention abstraction
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — Wires OIDC user enforcement and transaction maintenance

