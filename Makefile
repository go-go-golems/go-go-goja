.PHONY: js-install js-bundle js-transpile js-clean go-build go-run-bun

all: build

VERSION=v0.1.14

JS_DIR=js
BUN_ASSET_DIR=cmd/bun-demo/assets
BUN_ASSET=$(BUN_ASSET_DIR)/bundle.cjs

js-install:
	cd $(JS_DIR) && bun install

js-bundle: js-install
	cd $(JS_DIR) && bun build --target=node --format=cjs --outfile=dist/bundle.cjs src/main.js --external=fs --external=exec --external=database
	mkdir -p $(BUN_ASSET_DIR)
	cp $(JS_DIR)/dist/bundle.cjs $(BUN_ASSET)

js-transpile: js-bundle
	cd $(JS_DIR) && bun x esbuild dist/bundle.cjs --target=es5 --format=cjs --outfile=dist/bundle.es5.cjs

js-clean:
	rm -rf $(JS_DIR)/dist

go-build: js-bundle
	go build ./...

go-run-bun: js-bundle
	go run ./cmd/bun-demo

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v

lint:
	golangci-lint run -v

lintmax:
	golangci-lint run -v --max-same-issues=100

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude=G101,G304,G301,G306 -exclude-dir=.history ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

test:
	go test ./...

build:
	go generate ./...
	go build ./...

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
	go mod tidy
