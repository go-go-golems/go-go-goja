.PHONY: docker-lint lint lintmax golangci-lint-install gosec govulncheck test test-inspector fuzz fuzz-seeds build goreleaser tag-major tag-minor tag-patch release bump-glazed install-modules

all: build

VERSION=v0.1.14
GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
GOLANGCI_LINT_BIN ?= $(CURDIR)/.bin/golangci-lint
DEFAULT_PLUGIN_DIR ?= $(HOME)/.go-go-goja/plugins/examples
EXAMPLE_PLUGIN_NAMES := greeter clock validator kv system-info failing

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run -v

golangci-lint-install:
	mkdir -p $(dir $(GOLANGCI_LINT_BIN))
	GOBIN=$(dir $(GOLANGCI_LINT_BIN)) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: golangci-lint-install
	$(GOLANGCI_LINT_BIN) run -v

lintmax: golangci-lint-install
	$(GOLANGCI_LINT_BIN) run -v --max-same-issues=100

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude=G101,G304,G301,G306 -exclude-generated -exclude-dir=.history ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

test:
	go generate ./...
	go test ./...

FUZZTIME ?= 30s
FUZZ_TARGETS := FuzzEvaluateRaw FuzzRewriteIsolated FuzzEvaluateInstrumented \
                 FuzzSessionSequence FuzzSessionSequenceRaw FuzzPersistenceRoundTrip FuzzGlobalSnapshot

fuzz-seeds: ## Run all seed regression tests (fast, no mutation)
	go test ./fuzz/ -run 'TestFuzz' -v -count=1

fuzz: fuzz-seeds ## Run all fuzz targets for $(FUZZTIME) each
	@echo "=== Fuzz targets: $(FUZZTIME) each ==="
	@for target in $(FUZZ_TARGETS); do \
		echo ">>> $$target ($(FUZZTIME))"; \
		go test ./fuzz/ -fuzz=$$target -fuzztime=$(FUZZTIME) -v -count=1 || exit 1; \
	done
	@echo "=== All fuzz targets passed ==="

test-inspector:
	GOWORK=off go test ./pkg/jsparse -count=1
	GOWORK=off go test ./cmd/inspector/... -count=1
	GOWORK=off go build ./cmd/inspector

build:
	go generate ./...
	go build ./...

install-modules:
	mkdir -p "$(DEFAULT_PLUGIN_DIR)"
	@for name in $(EXAMPLE_PLUGIN_NAMES); do \
		echo "installing $$name -> $(DEFAULT_PLUGIN_DIR)/goja-plugin-examples-$$name"; \
		go build -o "$(DEFAULT_PLUGIN_DIR)/goja-plugin-examples-$$name" "./plugins/examples/$$name"; \
	done
gen-dts:
	go run ./cmd/gen-dts --out ./cmd/bun-demo/js/src/types/goja-modules.d.ts --module fs,exec,database,events,node:events --strict

check-dts:
	go run ./cmd/gen-dts --out ./cmd/bun-demo/js/src/types/goja-modules.d.ts --module fs,exec,database,events,node:events --strict --check

goreleaser:
	goreleaser release --skip=sign --snapshot --clean

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/go-go-goja@$(shell svu current)

bump-glazed:
	go get github.com/go-go-golems/glazed@latest
	go get github.com/go-go-golems/clay@latest
	go get github.com/go-go-golems/bobatea@latest
	go mod tidy

.PHONY: fuzz fuzz-seeds
