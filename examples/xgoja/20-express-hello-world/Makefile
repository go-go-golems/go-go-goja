.PHONY: smoke serve

EXAMPLE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
REPO_ROOT := $(abspath $(EXAMPLE_DIR)/../../..)
LISTEN ?= 127.0.0.1:18790
SCRIPT ?= $(EXAMPLE_DIR)/scripts/server.js

smoke:
	cd $(REPO_ROOT) && GOWORK=off go run ./examples/xgoja/20-express-hello-world/cmd/host --script $(SCRIPT) --smoke

serve:
	cd $(REPO_ROOT) && GOWORK=off go run ./examples/xgoja/20-express-hello-world/cmd/host --script $(SCRIPT) --listen $(LISTEN)
