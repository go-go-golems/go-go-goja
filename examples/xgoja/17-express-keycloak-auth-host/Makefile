REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)
HOST := ./examples/xgoja/17-express-keycloak-auth-host/cmd/host
SCRIPT := $(EXAMPLE_DIR)/scripts/server.js
LISTEN ?= 127.0.0.1:8790
ISSUER ?= http://127.0.0.1:18080/realms/goja-demo

.PHONY: keycloak-up keycloak-down serve test

keycloak-up:
	docker compose -f $(EXAMPLE_DIR)/docker-compose.yml up -d
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
	cd $(REPO_ROOT) && GOWORK=off go run $(HOST) --script $(SCRIPT) --listen $(LISTEN) --issuer $(ISSUER)

test:
	cd $(REPO_ROOT) && GOWORK=off go test ./examples/xgoja/17-express-keycloak-auth-host/cmd/host -count=1
