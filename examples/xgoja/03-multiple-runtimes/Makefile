.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/multiple-runtimes
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
	$(BIN) eval 'require("path").basename("/tmp/safe.txt")'
	@if $(BIN) eval 'require("fs")' >/tmp/xgoja-multiple-runtimes-unexpected.out 2>&1; then \
		echo "expected safe eval runtime to reject require(\"fs\")"; \
		cat /tmp/xgoja-multiple-runtimes-unexpected.out; \
		exit 1; \
	else \
		echo "safe eval runtime rejected fs as expected"; \
	fi
	cd $(CURDIR) && $(BIN) run $(CURDIR)/scripts/host-run.js

clean:
	rm -rf $(CURDIR)/dist $(CURDIR)/multiple-runtimes-host.txt /tmp/xgoja-multiple-runtimes-unexpected.out
