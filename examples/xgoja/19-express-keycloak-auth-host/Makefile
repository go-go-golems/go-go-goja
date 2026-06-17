REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)
HOST := ./examples/xgoja/19-express-keycloak-auth-host/cmd/host
SCRIPT := $(EXAMPLE_DIR)/scripts/server.js
LISTEN ?= 127.0.0.1:8790
KEYCLOAK_PORT ?= 18080
POSTGRES_PORT ?= 15432
ISSUER ?= http://127.0.0.1:$(KEYCLOAK_PORT)/realms/goja-demo
CLIENT_ID ?= goja-app
SESSION_DB_DSN ?= postgres://goja:goja@127.0.0.1:$(POSTGRES_PORT)/goja_auth?sslmode=disable
AUDIT_DB_DSN ?= $(SESSION_DB_DSN)
APPAUTH_DB_DSN ?= $(SESSION_DB_DSN)
CAPABILITY_DB_DSN ?= $(SESSION_DB_DSN)

.PHONY: keycloak-up keycloak-down serve test smoke

keycloak-up:
	KEYCLOAK_PORT=$(KEYCLOAK_PORT) POSTGRES_PORT=$(POSTGRES_PORT) docker compose -f $(EXAMPLE_DIR)/docker-compose.yml up -d
	@echo "Waiting for Keycloak discovery endpoint..."
	@for i in $$(seq 1 60); do \
		if curl -fsS $(ISSUER)/.well-known/openid-configuration >/dev/null 2>&1; then \
			echo "Keycloak ready at $(ISSUER)"; exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "Keycloak did not become ready"; exit 1

keycloak-down:
	docker compose -f $(EXAMPLE_DIR)/docker-compose.yml down -v

serve:
	cd $(REPO_ROOT) && GOWORK=off go run $(HOST) serve --script $(SCRIPT) --listen $(LISTEN) --issuer $(ISSUER) --client-id $(CLIENT_ID) --public-base-url http://$(LISTEN) --allow-insecure-http --session-db-dsn '$(SESSION_DB_DSN)' --audit-db-dsn '$(AUDIT_DB_DSN)' --app-db-dsn '$(APPAUTH_DB_DSN)' --capability-db-dsn '$(CAPABILITY_DB_DSN)'

test:
	cd $(REPO_ROOT) && GOWORK=off go test ./examples/xgoja/19-express-keycloak-auth-host/cmd/host -count=1

smoke:
	KEYCLOAK_PORT=$(KEYCLOAK_PORT) POSTGRES_PORT=$(POSTGRES_PORT) ISSUER=$(ISSUER) CLIENT_ID=$(CLIENT_ID) LISTEN=$(LISTEN) SESSION_DB_DSN='$(SESSION_DB_DSN)' AUDIT_DB_DSN='$(AUDIT_DB_DSN)' APPAUTH_DB_DSN='$(APPAUTH_DB_DSN)' CAPABILITY_DB_DSN='$(CAPABILITY_DB_DSN)' $(EXAMPLE_DIR)/scripts/smoke.sh
