.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
LISTEN ?= 127.0.0.1:18810
RUN := cd $(REPO_ROOT) && GOWORK=off go run ./examples/gojahttp/01-planned-auth

.PHONY: smoke serve

smoke:
	$(RUN) --smoke

serve:
	$(RUN) --listen $(LISTEN)
