.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/go-go-objects-docs
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: smoke doctor list build run clean

smoke: doctor list build run

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

list:
	$(XGOJA) list-modules -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT)

run:
	# Flow A: bundled help pages surface through the cobra `help` command.
	$(BIN) help go-go-objects-overview | grep -F "Run Cloudflare-style Durable Objects"
	$(BIN) help go-go-objects-docs-verb | grep -F "List, show, and serve"
	# Flow B: the docs verb lists, shows, and serves the same pages.
	$(BIN) durableobjects docs list --output json | grep -F "go-go-objects-js-api"
	$(BIN) durableobjects docs show go-go-objects-js-api | grep -F "Durable Objects JavaScript API"

serve: build
	$(BIN) durableobjects docs serve --addr 127.0.0.1:8788

clean:
	rm -rf $(CURDIR)/dist
