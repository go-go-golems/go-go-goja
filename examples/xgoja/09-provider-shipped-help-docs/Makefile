.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/provider-shipped-help-docs
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
	$(BIN) help loupedeck-js-api-reference | grep -F "Loupedeck JavaScript runtime API reference"
	$(BIN) help loupedeck-js-api-reference | grep -F "loupedeck/state"
	$(BIN) help loupedeck-js-first-live-script | grep -F "Build your first live Loupedeck JavaScript script"

clean:
	rm -rf $(CURDIR)/dist
