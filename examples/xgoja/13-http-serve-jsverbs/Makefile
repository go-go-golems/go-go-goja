.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/http-serve-jsverbs
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
HTTP_ADDR := 127.0.0.1:18788

.PHONY: smoke doctor build serve-smoke clean

smoke: doctor build serve-smoke

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

serve-smoke: build
	@set -eu; \
	log=$$(mktemp); \
	$(BIN) serve sites demo --http-listen $(HTTP_ADDR) >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log' EXIT; \
	for i in $$(seq 1 80); do \
		if curl -fsS http://$(HTTP_ADDR)/healthz >/tmp/xgoja-http-serve-health.json 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"ok":true' /tmp/xgoja-http-serve-health.json; \
	grep -q '"site":"demo"' /tmp/xgoja-http-serve-health.json; \
	curl -fsS http://$(HTTP_ADDR)/ | grep -q 'hello from an xgoja jsverb site'; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

clean:
	rm -rf $(CURDIR)/dist
