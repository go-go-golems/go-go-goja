.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: doctor generate smoke memory-smoke sqlite-smoke clean

smoke: doctor generate memory-smoke sqlite-smoke

doctor:
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/xgoja.yaml

generate:
	rm -rf $(EXAMPLE_DIR)/internal/xgojaruntime
	$(XGOJA) generate \
		-f $(EXAMPLE_DIR)/xgoja.yaml \
		--output $(EXAMPLE_DIR)/internal/xgojaruntime \
		--package xgojaruntime

memory-smoke: generate
	@set -eu; \
	addr=127.0.0.1:$$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'); \
	log=$$(mktemp); \
	cd $(REPO_ROOT) && GOWORK=off go run ./examples/xgoja/21-generated-host-auth/cmd/host serve sites demo --http-listen $$addr >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log /tmp/xgoja-generated-host-auth-health.json /tmp/xgoja-generated-host-auth-me.txt' EXIT; \
	for i in $$(seq 1 80); do \
		if curl -fsS http://$$addr/healthz >/tmp/xgoja-generated-host-auth-health.json 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"ok":true' /tmp/xgoja-generated-host-auth-health.json; \
	grep -q '"auth":"generated-host"' /tmp/xgoja-generated-host-auth-health.json; \
	curl -fsS http://$$addr/ | grep -q 'generated host auth demo'; \
	status=$$(curl -sS -o /tmp/xgoja-generated-host-auth-me.txt -w '%{http_code}' http://$$addr/me); \
	test "$$status" = "401"; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

sqlite-smoke: generate
	@set -eu; \
	addr=127.0.0.1:$$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'); \
	db=$$(mktemp -u /tmp/xgoja-generated-host-auth.XXXXXX.sqlite); \
	log=$$(mktemp); \
	cd $(REPO_ROOT) && XGOJA_AUTH_STORE=sqlite XGOJA_AUTH_SQLITE_DSN=$$db GOWORK=off go run ./examples/xgoja/21-generated-host-auth/cmd/host serve sites demo --http-listen $$addr >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log $$db /tmp/xgoja-generated-host-auth-health.json /tmp/xgoja-generated-host-auth-me.txt' EXIT; \
	for i in $$(seq 1 80); do \
		if curl -fsS http://$$addr/healthz >/tmp/xgoja-generated-host-auth-health.json 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"ok":true' /tmp/xgoja-generated-host-auth-health.json; \
	status=$$(curl -sS -o /tmp/xgoja-generated-host-auth-me.txt -w '%{http_code}' http://$$addr/me); \
	test "$$status" = "401"; \
	test -s $$db; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

clean:
	rm -rf $(EXAMPLE_DIR)/internal/xgojaruntime
