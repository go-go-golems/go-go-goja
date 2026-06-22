.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)
BIN := $(EXAMPLE_DIR)/dist/personal-knowledge-inbox
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: doctor build smoke clean

doctor:
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(EXAMPLE_DIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

smoke: doctor build
	$(BIN) verbs inbox hello --name tutorial | grep -q 'Hello, tutorial!'

clean:
	rm -rf $(EXAMPLE_DIR)/dist
