.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/embedded-assets-fs
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
OUT := $(CURDIR)/out.json

.PHONY: smoke doctor list build run prove-self-contained clean

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

prove-self-contained: build
	tmp=$$(mktemp -d); cp $(BIN) $$tmp/embedded-assets-fs; cp -R $(CURDIR)/scripts $$tmp/scripts; cd $$tmp && ./embedded-assets-fs run scripts/read-assets.js && test "$$(cat out.json)" = '{"name":"embedded-assets-fs","ok":true}'

clean:
	rm -rf $(CURDIR)/dist $(OUT)
