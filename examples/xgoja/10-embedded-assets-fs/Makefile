.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/embedded-assets-fs
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
OUT := $(CURDIR)/out.json
HTTP_ADDR := 127.0.0.1:18787

.PHONY: smoke doctor list build run serve-smoke prove-self-contained clean

smoke: doctor list build run

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

list:
	$(XGOJA) list-modules -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

run:
	rm -f $(OUT)
	$(BIN) run $(CURDIR)/scripts/read-assets.js
	@test "$$(cat $(OUT))" = '{"name":"embedded-assets-fs","ok":true}'

serve-smoke: build
	rm -rf $(CURDIR)/.xgoja-static
	@set -eu; \
	log=$$(mktemp); \
	$(BIN) run $(CURDIR)/scripts/serve-static-assets.js --http-listen $(HTTP_ADDR) --keep-alive >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log' EXIT; \
	for i in $$(seq 1 50); do \
		if curl -fsS http://$(HTTP_ADDR)/static/ >/tmp/xgoja-static-index.html 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q 'bundled static site' /tmp/xgoja-static-index.html; \
	curl -fsS http://$(HTTP_ADDR)/static/app.js | grep -q 'embedded asset bundle'; \
	curl -fsS http://$(HTTP_ADDR)/api/config | grep -q 'embedded-assets-fs'; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

prove-self-contained: build
	tmp=$$(mktemp -d); cp $(BIN) $$tmp/embedded-assets-fs; cp -R $(CURDIR)/scripts $$tmp/scripts; cd $$tmp && ./embedded-assets-fs run scripts/read-assets.js && test "$$(cat out.json)" = '{"name":"embedded-assets-fs","ok":true}'

clean:
	rm -rf $(CURDIR)/dist $(OUT) $(CURDIR)/.xgoja-static
