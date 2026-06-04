.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
BIN := $(CURDIR)/dist/geppetto-host-services
XGOJA := cd $(REPO_ROOT) && go run ./cmd/xgoja
PROFILE_REGISTRIES ?= $(HOME)/.config/pinocchio/profiles.yaml
PROFILE ?= gpt-5-nano
SESSION ?= xgoja-geppetto-host-services-example
DB ?= $(CURDIR)/dist/turns.db
EVENT_LOG ?= $(CURDIR)/dist/events.jsonl
BASIC_PROFILE_REGISTRIES ?= $(REPO_ROOT)/../pinocchio/examples/js/profiles/basic.yaml
BASIC_PROFILE ?= assistant

.PHONY: smoke doctor build help pinocchio-smoke live-smoke verify clean

smoke: doctor build help

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

help:
	$(BIN) verbs demo run --help

pinocchio-smoke: build
	$(BIN) verbs pinocchio profile-smoke xgoja-pinocchio-profile-smoke \
		--profile-registries $(BASIC_PROFILE_REGISTRIES) \
		--profile $(BASIC_PROFILE) \
		--output json

live-smoke: build
	mkdir -p $(CURDIR)/dist
	rm -f $(DB) $(EVENT_LOG)
	$(BIN) verbs demo run $(SESSION) \
		--profile-registries $(PROFILE_REGISTRIES) \
		--profile $(PROFILE) \
		--turns-db $(DB) \
		--event-log $(EVENT_LOG) \
		--output json
	$(MAKE) verify DB=$(DB) EVENT_LOG=$(EVENT_LOG)

verify:
	sqlite3 $(DB) 'select count(*) from geppetto_turns; select session_id, phase, length(payload) from geppetto_turns;'
	wc -l $(EVENT_LOG)
	head -5 $(EVENT_LOG)

clean:
	rm -rf $(CURDIR)/dist
