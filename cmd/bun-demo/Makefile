.PHONY: js-install js-typecheck js-bundle js-bundle-tsx js-render-tsx js-bundle-split js-transpile js-clean go-build go-run-bun go-run-bun-tsx go-run-bun-split

all: go-run-bun

JS_DIR=js
BUN_ASSET_DIR=assets
BUN_ASSET=$(BUN_ASSET_DIR)/bundle.cjs
TSX_BUNDLE=$(BUN_ASSET_DIR)/tsx-bundle.cjs
TSX_HTML=$(BUN_ASSET_DIR)/tsx.html
JS_SPLIT_DIST=$(JS_DIR)/dist-split
BUN_ASSET_SPLIT_DIR=assets-split
DAGGER?=dagger
DAGGER_RUN=$(DAGGER) run --
DAGGER_PIPELINE=$(DAGGER_RUN) go run ./dagger --project-dir .

js-install:
	$(DAGGER_PIPELINE) deps

js-typecheck:
	$(DAGGER_PIPELINE) typecheck

js-bundle:
	$(DAGGER_PIPELINE) bundle

js-bundle-tsx:
	$(DAGGER_PIPELINE) bundle-tsx

js-render-tsx:
	$(DAGGER_PIPELINE) render-tsx

js-bundle-split:
	$(DAGGER_PIPELINE) bundle-split

js-transpile:
	$(DAGGER_PIPELINE) transpile

js-clean:
	rm -rf $(JS_DIR)/dist
	rm -rf $(JS_SPLIT_DIST) $(BUN_ASSET_SPLIT_DIR)
	rm -f $(TSX_BUNDLE) $(TSX_HTML)

go-build: js-bundle
	go build .

go-run-bun: js-bundle
	go run .

go-run-bun-tsx: js-bundle-tsx
	go run . --entry ./assets/tsx-bundle.cjs

go-run-bun-split: js-bundle-split
	go run . --entry ./assets-split/app.js
