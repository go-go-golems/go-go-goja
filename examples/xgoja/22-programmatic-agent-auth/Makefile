.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)
SERVER_BIN := $(EXAMPLE_DIR)/dist/programmatic-agent-auth-server
AGENT_BIN := $(EXAMPLE_DIR)/dist/programmatic-agent-auth-agent
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: smoke doctor build clean

smoke: doctor build
	$(EXAMPLE_DIR)/scripts/smoke.sh $(SERVER_BIN) $(AGENT_BIN)

doctor:
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/xgoja.yaml
	$(XGOJA) doctor -f $(EXAMPLE_DIR)/agent.xgoja.yaml

build:
	$(XGOJA) build -f $(EXAMPLE_DIR)/xgoja.yaml --output $(SERVER_BIN) --xgoja-replace $(REPO_ROOT) --keep-work
	$(XGOJA) build -f $(EXAMPLE_DIR)/agent.xgoja.yaml --output $(AGENT_BIN) --xgoja-replace $(REPO_ROOT) --keep-work

clean:
	rm -rf $(EXAMPLE_DIR)/dist
