.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
HOST := cd $(REPO_ROOT) && GOWORK=off go run ./examples/xgoja/18-express-auth-host/cmd/host
SCRIPT := $(CURDIR)/scripts/server.js
HTTP_ADDR := 127.0.0.1:18789

.PHONY: smoke serve

smoke:
	$(HOST) --script $(SCRIPT) --smoke

serve:
	$(HOST) --script $(SCRIPT) --listen $(HTTP_ADDR)
