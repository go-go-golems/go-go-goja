# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Create host-side auth planning ticket with production Keycloak/OIDC and dev/demo implementation roadmap

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/design-doc/01-host-side-user-auth-system-implementation-plan.md — Primary implementation plan
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/sources/01-keycloak-oidc-session-authz-host-notes.md — Imported source notes


## 2026-06-12

Phase 1: add devauth package and refactor runnable Express auth host example (commit 38871dc)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/16-express-auth-host/cmd/host/main.go — Example now uses devauth with login/logout smoke flow
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/devauth/devauth.go — Reusable in-memory dev auth kit


## 2026-06-12

Phase 2: add reusable sessionauth package for session-cookie authentication and CSRF (commit d939b95)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/sessionauth/sessionauth.go — Reusable session auth manager and memory store
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/sessionauth/sessionauth_test.go — Sessionauth validation coverage


## 2026-06-12

Phase 3: add Keycloak/OIDC auth handlers with fake issuer tests (commit f297487)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/keycloakauth/keycloakauth.go — OIDC login/callback/logout handlers
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/keycloakauth/keycloakauth_test.go — OIDC verification tests


## 2026-06-12

Phase 4: add appauth domain helpers and negative authorization tests (commit 952acb2)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/appauth/appauth.go — App-owned authorization helper package
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/appauth/appauth_test.go — Negative authz tests


## 2026-06-12

Phase 5: add audit normalization/sinks and capability token helpers (commit 4141b8a; generated loggers 61c101e)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/audit/audit.go — Audit helpers
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/capability/capability.go — Capability helpers


## 2026-06-12

Phase 6: add Docker Compose Keycloak host example with sessionauth, appauth, audit, and planned Express routes (commit 852780d)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/17-express-keycloak-auth-host/cmd/host/main.go — Keycloak host example
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/17-express-keycloak-auth-host/docker-compose.yml — Local Keycloak service


## 2026-06-12

Phase 6: add and run automated Keycloak login/CSRF/planned-route smoke (commit 4f966f3)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/17-express-keycloak-auth-host/scripts/keycloak_smoke.py — Login and route assertions
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/17-express-keycloak-auth-host/scripts/smoke.sh — Smoke lifecycle

