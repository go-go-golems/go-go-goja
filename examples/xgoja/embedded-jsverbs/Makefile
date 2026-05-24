.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/embedded-jsverbs
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: smoke doctor list build run prove-self-contained clean

smoke: doctor list build run

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

list:
	$(XGOJA) list-modules -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

run:
	$(BIN) repl 'require("hello").greet("embedded")'
	$(BIN) verbs tools embedded-greet --name embedded

prove-self-contained: build
	tmp=$$(mktemp -d); cp $(BIN) $$tmp/embedded-jsverbs; cd $$tmp && ./embedded-jsverbs verbs tools embedded-greet --name embedded

clean:
	rm -rf $(CURDIR)/dist
