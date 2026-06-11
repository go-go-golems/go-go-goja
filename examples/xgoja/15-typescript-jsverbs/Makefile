.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/typescript-jsverbs
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
HTTP_ADDR := 127.0.0.1:18789

.PHONY: smoke doctor build types serve-smoke run-smoke clean

smoke: doctor types build run-smoke serve-smoke

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

types:
	$(XGOJA) gen-dts -f $(CURDIR)/xgoja.yaml --out $(CURDIR)/js/types/xgoja-modules.d.ts --strict --xgoja-replace $(REPO_ROOT)

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

run-smoke: build
	@set -eu; \
	tmp=$$(mktemp -d); \
	trap 'rm -rf $$tmp' EXIT; \
	printf '%s\n' 'export const name: string = "xgoja run";' > $$tmp/helper.ts; \
	printf '%s\n' 'import { name } from "./helper";' 'globalThis.result = "hello " + name;' > $$tmp/main.ts; \
	$(BIN) run $$tmp/main.ts

serve-smoke: build
	@set -eu; \
	log=$$(mktemp); \
	$(BIN) serve sites demo --http-listen $(HTTP_ADDR) --hot-reload --hot-reload-watch-root $(CURDIR)/verbs --hot-reload-poll 50ms --hot-reload-debounce 50ms --hot-reload-smoke-path /healthz >$$log 2>&1 & \
	pid=$$!; \
	orig=$$(mktemp); \
	cp $(CURDIR)/verbs/sites.ts $$orig; \
	trap 'cp $$orig $(CURDIR)/verbs/sites.ts >/dev/null 2>&1 || true; kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log $$orig' EXIT; \
	for i in $$(seq 1 80); do \
		if curl -fsS http://$(HTTP_ADDR)/healthz >/tmp/xgoja-typescript-health.json 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"ok":true' /tmp/xgoja-typescript-health.json; \
	grep -q '"version":1' /tmp/xgoja-typescript-health.json; \
	python3 -c "from pathlib import Path; p=Path('$(CURDIR)/verbs/sites.ts'); s=p.read_text(); p.write_text(s.replace('const version = 1', 'const version = 2'))"; \
	for i in $$(seq 1 80); do \
		if curl -fsS http://$(HTTP_ADDR)/healthz >/tmp/xgoja-typescript-health.json 2>/dev/null && grep -q '"version":2' /tmp/xgoja-typescript-health.json; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"version":2' /tmp/xgoja-typescript-health.json; \
	curl -fsS http://$(HTTP_ADDR)/ | grep -q 'hello from TypeScript xgoja v2'; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

clean:
	rm -rf $(CURDIR)/dist $(CURDIR)/js/types/xgoja-modules.d.ts
