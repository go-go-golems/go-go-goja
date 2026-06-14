.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/sourcegraph-runtime-aliases
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
HTTP_ADDR := 127.0.0.1:18790

.PHONY: smoke doctor build builtin-smoke serve-smoke prove-self-contained clean

smoke: doctor build builtin-smoke serve-smoke prove-self-contained

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

builtin-smoke: build
	$(BIN) verbs typed echo | tee /tmp/xgoja-sourcegraph-aliases-typed.txt
	grep -q 'typed:sourcegraph-aliases:function' /tmp/xgoja-sourcegraph-aliases-typed.txt
	$(BIN) verbs sources | grep -q 'ts-tools'

serve-smoke: build
	@set -eu; \
	log=$$(mktemp); \
	$(BIN) serve imports serve --http-listen $(HTTP_ADDR) >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log' EXIT; \
	for i in $$(seq 1 80); do \
		if curl -fsS http://$(HTTP_ADDR)/healthz >/tmp/xgoja-sourcegraph-health.json 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"ok":true' /tmp/xgoja-sourcegraph-health.json; \
	grep -q 'helper:health' /tmp/xgoja-sourcegraph-health.json; \
	curl -fsS http://$(HTTP_ADDR)/ | grep -q 'sourcegraph runtime aliases'; \
	curl -fsS http://$(HTTP_ADDR)/static/app.js | grep -q 'embedded asset bundle'; \
	curl -fsS http://$(HTTP_ADDR)/api/config | grep -q 'helper:config'; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

prove-self-contained: build
	@set -eu; \
	tmp=$$(mktemp -d); \
	cp $(BIN) $$tmp/sourcegraph-runtime-aliases; \
	cd $$tmp; \
	./sourcegraph-runtime-aliases verbs typed echo | grep -q 'typed:sourcegraph-aliases:function'

clean:
	rm -rf $(CURDIR)/dist
