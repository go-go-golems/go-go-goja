.PHONY: js-install js-typecheck js-bundle js-bundle-tsx js-render-tsx js-bundle-split js-transpile js-clean go-build go-run-bun go-run-bun-tsx go-run-bun-split

all: go-run-bun

JS_DIR=js
BUN_ASSET_DIR=assets
BUN_ASSET=$(BUN_ASSET_DIR)/bundle.cjs
TSX_BUNDLE=$(BUN_ASSET_DIR)/tsx-bundle.cjs
TSX_HTML=$(BUN_ASSET_DIR)/tsx.html
JS_SPLIT_DIST=$(JS_DIR)/dist-split
BUN_ASSET_SPLIT_DIR=assets-split

js-install:
	cd $(JS_DIR) && bun install

js-typecheck: js-install
	cd $(JS_DIR) && bun run typecheck

js-bundle: js-install
	cd $(JS_DIR) && bun run build
	mkdir -p $(BUN_ASSET_DIR)
	cp $(JS_DIR)/dist/bundle.cjs $(BUN_ASSET)

js-bundle-tsx: js-install
	cd $(JS_DIR) && bun run build:tsx
	mkdir -p $(BUN_ASSET_DIR)
	cp $(JS_DIR)/dist/tsx-bundle.cjs $(TSX_BUNDLE)

js-render-tsx: js-bundle-tsx
	cd $(JS_DIR) && bun run render:tsx-html > dist/tsx.html
	cp $(JS_DIR)/dist/tsx.html $(TSX_HTML)

js-bundle-split: js-install
	cd $(JS_DIR) && bun run build:split
	rm -rf $(BUN_ASSET_SPLIT_DIR)
	mkdir -p $(BUN_ASSET_SPLIT_DIR)
	cp -R $(JS_SPLIT_DIST)/. $(BUN_ASSET_SPLIT_DIR)

js-transpile: js-bundle
	cd $(JS_DIR) && bun x esbuild dist/bundle.cjs --target=es5 --format=cjs --outfile=dist/bundle.es5.cjs

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
