.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/command-provider
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
	$(BIN) fixture bare --message bare-ok --fixture-value shared
	$(BIN) fixture write --message write-ok --fixture-value shared
	$(BIN) fixture rows --message rows-ok --fixture-value shared --output json
	$(BIN) fixture asset --fixture-value shared | grep 'host-service asset'

clean:
	rm -rf $(CURDIR)/dist
