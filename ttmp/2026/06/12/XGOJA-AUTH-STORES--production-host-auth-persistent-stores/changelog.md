# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Create initial persistent auth store planning ticket with phased design and tasks

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-AUTH-STORES--production-host-auth-persistent-stores/design/01-persistent-auth-store-implementation-plan.md — Initial persistent store design
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-AUTH-STORES--production-host-auth-persistent-stores/tasks.md — Initial phased task list


## 2026-06-12

Phase 1: add reusable auth store contract tests and fix memory-store clone isolation (commit 22eb7d6)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/internal/appauthtest/store_contract.go — App auth store contract
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/internal/audittest/store_contract.go — Audit store contract
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/internal/capabilitytest/store_contract.go — Capability store contract
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/internal/sessionauthtest/store_contract.go — Session store contract


## 2026-06-12

Phase 2: add SQL-backed sessionauth store with SQLite tests and Postgres schema path (commit 304f833)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/sessionauth/sqlstore/schema.go — SQL session schema
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go — SQL session store implementation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore_test.go — SQL session store tests


## 2026-06-12

Wire Keycloak example smoke to Postgres-backed sessionauth/sqlstore for end-to-end validation (commit e53d063)

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go — SQL session store wiring
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml — Postgres service for smoke
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh — Postgres-backed smoke orchestration

