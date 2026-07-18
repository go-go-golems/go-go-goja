.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../../..)
EXAMPLE_DIR := $(abspath .)
BIN := $(EXAMPLE_DIR)/dist/personal-knowledge-inbox-sqlite-cli
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
HTTP_ADDR := 127.0.0.1:18791

.PHONY: doctor build smoke serve-smoke cli-smoke clean

doctor:
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(EXAMPLE_DIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

smoke: doctor build
	$(BIN) verbs inbox hello --name tutorial | grep -q 'Hello, tutorial!'
	$(MAKE) serve-smoke
	$(MAKE) cli-smoke

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
	grep -q '"step":"03-sqlite-cli-inbox"' $$health; \
	curl -fsS http://$(HTTP_ADDR)/ >$$root; \
	grep -q 'Hello from the Personal Knowledge Inbox web server.' $$root; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

cli-smoke:
	@set -eu; \
	db=$$(mktemp -u /tmp/xgoja-personal-inbox-XXXXXX.sqlite); \
	capture1=$$(mktemp); \
	capture2=$$(mktemp); \
	list1=$$(mktemp); \
	list2=$$(mktemp); \
	trap 'rm -f $$db $$capture1 $$capture2 $$list1 $$list2' EXIT; \
	if $(BIN) verbs inbox capture --db $$db --title 'Missing URL' >/dev/null 2>&1; then echo 'capture without url unexpectedly succeeded'; exit 1; fi; \
	if $(BIN) verbs inbox capture --db $$db --url https://example.com/missing-title >/dev/null 2>&1; then echo 'capture without title unexpectedly succeeded'; exit 1; fi; \
	$(BIN) verbs inbox capture --db $$db --title 'First item' --url https://example.com/first --note 'Read first' >$$capture1; \
	$(BIN) verbs inbox capture --db $$db --title 'Second item' --url https://example.com/second --note 'Read second' >$$capture2; \
	$(BIN) verbs inbox list --db $$db >$$list1; \
	python3 $(EXAMPLE_DIR)/scripts/cli_smoke.py $$capture1 $$list1 $$db $(BIN) $$list2

clean:
	rm -rf $(EXAMPLE_DIR)/dist
