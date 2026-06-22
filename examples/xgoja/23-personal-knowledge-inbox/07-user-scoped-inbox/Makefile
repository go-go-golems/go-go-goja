.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../../..)
EXAMPLE_DIR := $(abspath .)
BIN := $(EXAMPLE_DIR)/dist/personal-knowledge-inbox-user-scoped-inbox
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
HTTP_ADDR := 127.0.0.1:18794
BASE_URL := http://$(HTTP_ADDR)
KEYCLOAK_PORT ?= 18087
ISSUER := http://127.0.0.1:$(KEYCLOAK_PORT)/realms/personal-inbox
COMPOSE := docker compose -f $(EXAMPLE_DIR)/compose.yaml
TINYIDP_ROOT ?= /home/manuel/workspaces/2026-06-20/ui-notebook-package/2026-06-22--mock-oidc-idp
TINYIDP_ADDR ?= 127.0.0.1:19088
TINYIDP_APP_ADDR ?= 127.0.0.1:19795
TINYIDP_ISSUER := http://$(TINYIDP_ADDR)
TINYIDP_BASE_URL := http://$(TINYIDP_APP_ADDR)

.PHONY: doctor build smoke direct-smoke keycloak-up keycloak-wait keycloak-down keycloak-smoke tinyidp-smoke clean

doctor:
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(EXAMPLE_DIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

# Default smoke stays local and fast: it validates the generated binary and the
# direct database CLI verbs. Run `make keycloak-smoke` for the Docker/OIDC flow.
smoke: doctor build direct-smoke

direct-smoke:
	@set -eu; \
	db=$$(mktemp -u /tmp/xgoja-personal-inbox-direct-XXXXXX.sqlite); \
	capture1=$$(mktemp); \
	capture2=$$(mktemp); \
	list1=$$(mktemp); \
	trap 'rm -f $$db $$capture1 $$capture2 $$list1' EXIT; \
	if $(BIN) verbs inboxctl capture --db $$db --title 'Missing URL' >/dev/null 2>&1; then echo 'direct capture without url unexpectedly succeeded'; exit 1; fi; \
	if $(BIN) verbs inboxctl capture --db $$db --url https://example.com/missing-title >/dev/null 2>&1; then echo 'direct capture without title unexpectedly succeeded'; exit 1; fi; \
	$(BIN) verbs inboxctl capture --db $$db --title 'First direct item' --url https://example.com/first --note 'Read first' >$$capture1; \
	$(BIN) verbs inboxctl capture --db $$db --title 'Second direct item' --url https://example.com/second --note 'Read second' >$$capture2; \
	$(BIN) verbs inboxctl list --db $$db >$$list1; \
	python3 -c 'import json,sys; c=json.load(open(sys.argv[1])); l=json.load(open(sys.argv[2])); assert c["ok"] is True, c; assert c["item"]["submittedByKind"] == "localCli", c; assert l["ok"] is True, l; assert len(l["items"]) == 2, l; print("ok direct cli capture/list")' $$capture1 $$list1

keycloak-up:
	KEYCLOAK_PORT=$(KEYCLOAK_PORT) $(COMPOSE) up -d
	$(MAKE) keycloak-wait

keycloak-wait:
	@set -eu; \
	for i in $$(seq 1 90); do \
		if curl -fsS $(ISSUER)/.well-known/openid-configuration >/dev/null 2>&1; then \
			echo "ok keycloak discovery $(ISSUER)"; \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "timed out waiting for keycloak discovery: $(ISSUER)" >&2; \
	exit 1

keycloak-down:
	$(COMPOSE) down -v

keycloak-smoke: doctor build keycloak-up
	@set -eu; \
	db=$$(mktemp -u /tmp/xgoja-personal-inbox-api-XXXXXX.sqlite); \
	authdb=$$(mktemp -u /tmp/xgoja-personal-inbox-auth-XXXXXX.sqlite); \
	log=$$(mktemp); \
	health=$$(mktemp); \
	root=$$(mktemp); \
	login_headers=$$(mktemp); \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$db $$authdb $$log $$health $$root $$login_headers; $(COMPOSE) down -v >/dev/null 2>&1 || true' EXIT; \
	for i in $$(seq 1 90); do \
		if curl -fsS $(ISSUER)/.well-known/openid-configuration >/dev/null 2>&1; then break; fi; \
		sleep 2; \
	done; \
	$(BIN) serve inbox server --http-listen $(HTTP_ADDR) --db $$db \
		--auth-oidc-issuer-url $(ISSUER) \
		--auth-oidc-client-id personal-inbox-local \
		--auth-oidc-public-base-url $(BASE_URL) \
		--auth-session-cookie-allow-insecure-http=true \
		--auth-default-store-driver sqlite \
		--auth-default-store-dsn $$authdb \
		--auth-default-store-apply-schema=true \
		>$$log 2>&1 & \
	pid=$$!; \
	for i in $$(seq 1 80); do \
		if curl -fsS $(BASE_URL)/healthz >$$health 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.25; \
	done; \
	grep -q '"ok":true' $$health; \
	grep -q '"step":"07-user-scoped-inbox"' $$health; \
	curl -fsS $(BASE_URL)/ >$$root; \
	grep -q 'alice-password' $$root; \
	curl -fsS $(BASE_URL)/static/app.js | grep -q '/auth/session'; \
	status=$$(curl -s -o /dev/null -w '%{http_code}' $(BASE_URL)/api/inbox); \
	test "$$status" = "401"; \
	curl -fsS -D $$login_headers -o /dev/null $(BASE_URL)/auth/login; \
	grep -qi 'location: .*127.0.0.1:$(KEYCLOAK_PORT).*personal-inbox' $$login_headers; \
	echo 'ok keycloak login smoke'

tinyidp-smoke: doctor build
	@set -eu; \
	if [ ! -d "$(TINYIDP_ROOT)" ]; then echo "TINYIDP_ROOT not found: $(TINYIDP_ROOT)" >&2; exit 1; fi; \
	db=$$(mktemp -u /tmp/xgoja-tinyidp-$(notdir $(EXAMPLE_DIR))-api-XXXXXX.sqlite); \
	authdb=$$(mktemp -u /tmp/xgoja-tinyidp-$(notdir $(EXAMPLE_DIR))-auth-XXXXXX.sqlite); \
	idp_log=$$(mktemp); \
	app_log=$$(mktemp); \
	tinyidp_bin=$$(mktemp); \
	root=$$(mktemp); \
	idp_pid=; \
	app_pid=; \
	trap 'if [ -n "$$idp_pid" ]; then kill $$idp_pid >/dev/null 2>&1 || true; wait $$idp_pid >/dev/null 2>&1 || true; fi; if [ -n "$$app_pid" ]; then kill $$app_pid >/dev/null 2>&1 || true; wait $$app_pid >/dev/null 2>&1 || true; fi; rm -f $$db $$authdb $$idp_log $$app_log $$tinyidp_bin $$root' EXIT; \
	(cd $(TINYIDP_ROOT) && GOWORK=off go build -o $$tinyidp_bin ./cmd/tinyidp); \
	$$tinyidp_bin serve \
		--addr $(TINYIDP_ADDR) \
		--issuer $(TINYIDP_ISSUER) \
		--client-id personal-inbox-local \
		--redirect-uris $(TINYIDP_BASE_URL)/auth/callback \
		>$$idp_log 2>&1 & \
	idp_pid=$$!; \
	for i in $$(seq 1 120); do \
		if curl -fsS $(TINYIDP_ISSUER)/.well-known/openid-configuration >/dev/null 2>&1; then break; fi; \
		if ! kill -0 $$idp_pid >/dev/null 2>&1; then cat $$idp_log; exit 1; fi; \
		sleep 0.25; \
	done; \
	curl -fsS $(TINYIDP_ISSUER)/.well-known/openid-configuration >/dev/null; \
	$(BIN) serve inbox server --http-listen $(TINYIDP_APP_ADDR) --db $$db \
		--auth-oidc-issuer-url $(TINYIDP_ISSUER) \
		--auth-oidc-client-id personal-inbox-local \
		--auth-oidc-public-base-url $(TINYIDP_BASE_URL) \
		--auth-session-cookie-allow-insecure-http=true \
		--auth-default-store-driver sqlite \
		--auth-default-store-dsn $$authdb \
		--auth-default-store-apply-schema=true \
		>$$app_log 2>&1 & \
	app_pid=$$!; \
	for i in $$(seq 1 120); do \
		if curl -fsS $(TINYIDP_BASE_URL)/healthz >/dev/null 2>&1; then break; fi; \
		if ! kill -0 $$app_pid >/dev/null 2>&1; then cat $$app_log; exit 1; fi; \
		sleep 0.25; \
	done; \
	curl -fsS $(TINYIDP_BASE_URL)/ >$$root; \
	grep -q 'alice-password' $$root; \
	status=$$(curl -s -o /dev/null -w '%{http_code}' $(TINYIDP_BASE_URL)/api/inbox); \
	test "$$status" = "401"; \
	python3 $(EXAMPLE_DIR)/../06-browser-login-keycloak/scripts/tinyidp_login_smoke.py --base-url $(TINYIDP_BASE_URL); \
	echo 'ok tinyidp replacement smoke'

clean:
	rm -rf $(EXAMPLE_DIR)/dist
