.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../../..)
EXAMPLE_DIR := $(abspath .)
BIN := $(EXAMPLE_DIR)/dist/personal-knowledge-inbox-api-client-server
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
HTTP_ADDR := 127.0.0.1:18792
BASE_URL := http://$(HTTP_ADDR)

.PHONY: doctor build smoke api-smoke clean

doctor:
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(EXAMPLE_DIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

smoke: doctor build
	$(MAKE) api-smoke

api-smoke:
	@set -eu; \
	db=$$(mktemp -u /tmp/xgoja-personal-inbox-api-XXXXXX.sqlite); \
	log=$$(mktemp); \
	health=$$(mktemp); \
	root=$$(mktemp); \
	capture1=$$(mktemp); \
	capture2=$$(mktemp); \
	list1=$$(mktemp); \
	list2=$$(mktemp); \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$db $$log $$health $$root $$capture1 $$capture2 $$list1 $$list2' EXIT; \
	$(BIN) serve inbox server --http-listen $(HTTP_ADDR) --db $$db >$$log 2>&1 & \
	pid=$$!; \
	for i in $$(seq 1 80); do \
		if curl -fsS $(BASE_URL)/healthz >$$health 2>/dev/null; then break; fi; \
		if ! kill -0 $$pid >/dev/null 2>&1; then cat $$log; exit 1; fi; \
		sleep 0.1; \
	done; \
	grep -q '"ok":true' $$health; \
	grep -q '"step":"04-api-client-server"' $$health; \
	curl -fsS $(BASE_URL)/ >$$root; \
	grep -q 'Hello from the Personal Knowledge Inbox API server.' $$root; \
	if $(BIN) verbs inboxctl capture --base-url $(BASE_URL) --title 'Missing URL' >/dev/null 2>&1; then echo 'client capture without url unexpectedly succeeded'; exit 1; fi; \
	if $(BIN) verbs inboxctl capture --base-url $(BASE_URL) --url https://example.com/missing-title >/dev/null 2>&1; then echo 'client capture without title unexpectedly succeeded'; exit 1; fi; \
	$(BIN) verbs inboxctl capture --base-url $(BASE_URL) --title 'First API item' --url https://example.com/first --note 'Read first' >$$capture1; \
	$(BIN) verbs inboxctl capture --base-url $(BASE_URL) --title 'Second API item' --url https://example.com/second --note 'Read second' >$$capture2; \
	$(BIN) verbs inboxctl list --base-url $(BASE_URL) >$$list1; \
	python3 $(EXAMPLE_DIR)/scripts/api_smoke.py $$capture1 $$list1 $(BIN) $(BASE_URL) $$list2

clean:
	rm -rf $(EXAMPLE_DIR)/dist
