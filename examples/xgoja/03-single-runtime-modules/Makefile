.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/single-runtime-modules
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
	$(BIN) eval 'require("path").basename("/tmp/single.txt")'
	$(BIN) eval 'typeof require("fs").readFileSync === "function"'
	cd $(CURDIR) && $(BIN) run $(CURDIR)/scripts/host-run.js

clean:
	rm -rf $(CURDIR)/dist $(CURDIR)/single-runtime-modules-host.txt /tmp/xgoja-single-runtime-modules-unexpected.out
