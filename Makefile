.PHONY: docker-lint lint lintmax golangci-lint-install gosec govulncheck install-generate-tools test test-inspector fuzz fuzz-seeds build goreleaser tag-major tag-minor tag-patch release bump-glazed install-modules glazed-lint-build glazed-lint

all: build

VERSION=v0.1.14
GLAZED_LINT_BIN ?= $(CURDIR)/.bin/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= $(shell GOWORK=off go list -m -f '{{.Version}}' github.com/go-go-golems/glazed 2>/dev/null)
GLAZED_LINT_FLAGS ?= -glazedclilint.allow-paths=pkg/analysis/,pkg/cli/,pkg/cmds/fields/,pkg/cmds/logging/,pkg/cmds/sources/,pkg/help/
GLAZED_LINT_DIRS ?= ./cmd/... ./internal/... ./pkg/...
GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
GOLANGCI_LINT_BIN ?= $(CURDIR)/.bin/golangci-lint
DEFAULT_PLUGIN_DIR ?= $(HOME)/.go-go-goja/plugins/examples
EXAMPLE_PLUGIN_NAMES := greeter clock validator kv system-info failing

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run -v

golangci-lint-install:
	mkdir -p $(dir $(GOLANGCI_LINT_BIN))
	GOBIN=$(dir $(GOLANGCI_LINT_BIN)) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
glazed-lint-build:
	@echo "Building glazed-lint from go.mod tool dependency..."
	@echo "Installing $(GLAZED_LINT_PKG) (glazed $(GLAZED_VERSION))"
	@mkdir -p $(dir $(GLAZED_LINT_BIN))
	@GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)

glazed-lint: glazed-lint-build
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)


lint: golangci-lint-install glazed-lint-build
	$(GOLANGCI_LINT_BIN) run -v
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

lintmax: golangci-lint-install glazed-lint-build
	$(GOLANGCI_LINT_BIN) run -v --max-same-issues=100
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude=G101,G304,G301,G306 -exclude-generated -exclude-dir=.history ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

install-generate-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	go install ./cmd/protoc-gen-goja-builder

test: install-generate-tools
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

build: install-generate-tools
	go generate ./...
	go build ./...

install-modules:
	mkdir -p "$(DEFAULT_PLUGIN_DIR)"
	@for name in $(EXAMPLE_PLUGIN_NAMES); do \
		echo "installing $$name -> $(DEFAULT_PLUGIN_DIR)/goja-plugin-examples-$$name"; \
		go build -o "$(DEFAULT_PLUGIN_DIR)/goja-plugin-examples-$$name" "./plugins/examples/$$name"; \
	done
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

install:
	GOWORK=off go build -o ~/.local/bin/xgoja ./cmd/xgoja
	GOWORK=off go build -o ~/.local/bin/goja-repl ./cmd/goja-repl

.PHONY: fuzz fuzz-seeds

.PHONY: bump-go-go-golems
bump-go-go-golems:
	@deps="$$(awk '/^require[[:space:]]+github\.com\/go-go-golems\// { print $$2 } /^[[:space:]]*github\.com\/go-go-golems\// { print $$1 }' go.mod | sort -u)"; \
	if [ -z "$$deps" ]; then \
		echo "No github.com/go-go-golems dependencies in go.mod"; \
	else \
		echo "Bumping go-go-golems dependencies:"; \
		echo "$$deps"; \
		for dep in $$deps; do GOWORK=off go get "$${dep}@latest"; done; \
	fi
	GOWORK=off go mod tidy

.PHONY: logcopter-generate
logcopter-generate:
	GOWORK=off go generate ./...

.PHONY: logcopter-check
logcopter-check:
	GOWORK=off go tool logcopter-gen -area-prefix go-go-golems.go-go-goja -strip-prefix github.com/go-go-golems/go-go-goja -check ./cmd/... ./pkg/... ./modules/...
