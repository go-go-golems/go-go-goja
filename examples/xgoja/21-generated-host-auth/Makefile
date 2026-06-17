.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)
BIN := $(EXAMPLE_DIR)/dist/generated-oidc-host-auth
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: smoke doctor build oidc-smoke clean

smoke: doctor build oidc-smoke

doctor:
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(EXAMPLE_DIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

oidc-smoke: build
	@set -eu; \
	addr=127.0.0.1:$$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'); \
	issuer_port=$$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'); \
	issuer="http://127.0.0.1:$$issuer_port"; \
	log=$$(mktemp); \
	provider_log=$$(mktemp); \
	python3 $(EXAMPLE_DIR)/scripts/fake_oidc_provider.py --port $$issuer_port --issuer $$issuer >$$provider_log 2>&1 & \
	provider_pid=$$!; \
	for i in $$(seq 1 80); do \
		if curl -fsS $$issuer/.well-known/openid-configuration >/dev/null 2>&1; then break; fi; \
		if ! kill -0 $$provider_pid >/dev/null 2>&1; then cat $$provider_log; exit 1; fi; \
		sleep 0.1; \
	done; \
	$(BIN) serve sites demo \
	  --http-listen $$addr \
	  --auth-oidc-issuer-url $$issuer \
	  --auth-oidc-client-id generated-oidc-host-auth \
	  --auth-oidc-client-secret smoke-secret \
	  --auth-oidc-public-base-url http://$$addr \
	  >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid $$provider_pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; wait $$provider_pid >/dev/null 2>&1 || true; rm -f $$log $$provider_log /tmp/xgoja-generated-oidc-health.json /tmp/xgoja-generated-oidc-root.txt /tmp/xgoja-generated-oidc-app.js /tmp/xgoja-generated-oidc-styles.css /tmp/xgoja-generated-oidc-login.headers /tmp/xgoja-generated-oidc-me.txt /tmp/xgoja-generated-oidc-audit.txt' EXIT; \
	for i in $$(seq 1 80); do \
		if curl -fsS http://$$addr/healthz >/tmp/xgoja-generated-oidc-health.json 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"ok":true' /tmp/xgoja-generated-oidc-health.json; \
	grep -q '"auth":"generated-oidc"' /tmp/xgoja-generated-oidc-health.json; \
	curl -fsS http://$$addr/ >/tmp/xgoja-generated-oidc-root.txt; \
	grep -q '<title>go-go-goja generated OIDC auth host</title>' /tmp/xgoja-generated-oidc-root.txt; \
	grep -q '<script src="/static/app.js"></script>' /tmp/xgoja-generated-oidc-root.txt; \
	curl -fsS http://$$addr/static/app.js >/tmp/xgoja-generated-oidc-app.js; \
	grep -q 'async function loadAudit' /tmp/xgoja-generated-oidc-app.js; \
	grep -q '/orgs/o1/audit' /tmp/xgoja-generated-oidc-app.js; \
	curl -fsS http://$$addr/static/styles.css >/tmp/xgoja-generated-oidc-styles.css; \
	grep -q 'radial-gradient' /tmp/xgoja-generated-oidc-styles.css; \
	status=$$(curl -sS -D /tmp/xgoja-generated-oidc-login.headers -o /dev/null -w '%{http_code}' http://$$addr/auth/login); \
	test "$$status" = "302"; \
	grep -q "^Location: $$issuer/auth" /tmp/xgoja-generated-oidc-login.headers; \
	status=$$(curl -sS -o /tmp/xgoja-generated-oidc-me.txt -w '%{http_code}' http://$$addr/me); \
	test "$$status" = "401"; \
	status=$$(curl -sS -o /tmp/xgoja-generated-oidc-audit.txt -w '%{http_code}' http://$$addr/orgs/o1/audit); \
	test "$$status" = "401"; \
	kill $$pid $$provider_pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true; \
	wait $$provider_pid >/dev/null 2>&1 || true

clean:
	rm -rf $(EXAMPLE_DIR)/dist
