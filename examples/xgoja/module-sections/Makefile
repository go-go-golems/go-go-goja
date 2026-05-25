.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/module-sections
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
	$(BIN) eval 'globalThis.fixtureValue' --fixture-value eval-ok
	$(BIN) run $(CURDIR)/scripts/check-fixture.js --fixture-value run-ok
	$(BIN) verbs tools check-fixture --fixture-value verb-ok

clean:
	rm -rf $(CURDIR)/dist
