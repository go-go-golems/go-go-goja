.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../../..)
EXAMPLE_DIR := $(abspath .)
BIN := $(EXAMPLE_DIR)/dist/personal-knowledge-inbox-hello-web-server
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
HTTP_ADDR := 127.0.0.1:18790

.PHONY: doctor build smoke serve-smoke clean

doctor:
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(EXAMPLE_DIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

smoke: doctor build
	$(BIN) verbs inbox hello --name tutorial | grep -q 'Hello, tutorial!'
	$(MAKE) serve-smoke

serve-smoke:
	@set -eu; \
	log=$$(mktemp); \
	health=$$(mktemp); \
	root=$$(mktemp); \
	$(BIN) serve inbox server --http-listen $(HTTP_ADDR) >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log $$health $$root' EXIT; \
	for i in $$(seq 1 80); do \
		if curl -fsS http://$(HTTP_ADDR)/healthz >$$health 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"ok":true' $$health; \
	grep -q '"step":"02-hello-web-server"' $$health; \
	curl -fsS http://$(HTTP_ADDR)/ >$$root; \
	grep -q 'Hello from the Personal Knowledge Inbox web server.' $$root; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

clean:
	rm -rf $(EXAMPLE_DIR)/dist
