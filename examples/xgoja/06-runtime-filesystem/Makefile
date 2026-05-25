.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/runtime-filesystem
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: smoke doctor list build run clean

smoke: doctor list build run

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

list:
	$(XGOJA) list-modules -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

run:
	$(BIN) eval 'require("hello").greet("filesystem")'
	$(BIN) run $(CURDIR)/scripts/run.js
	cd $(CURDIR) && $(BIN) verbs tools greet --name filesystem

clean:
	rm -rf $(CURDIR)/dist
