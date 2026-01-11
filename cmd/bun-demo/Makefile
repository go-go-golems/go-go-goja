.PHONY: js-install js-typecheck js-bundle js-transpile js-clean go-build go-run-bun

all: go-run-bun

JS_DIR=js
BUN_ASSET_DIR=assets
BUN_ASSET=$(BUN_ASSET_DIR)/bundle.cjs

js-install:
	cd $(JS_DIR) && bun install

js-typecheck: js-install
	cd $(JS_DIR) && bun run typecheck

js-bundle: js-install
	cd $(JS_DIR) && bun run build
	mkdir -p $(BUN_ASSET_DIR)
	cp $(JS_DIR)/dist/bundle.cjs $(BUN_ASSET)

js-transpile: js-bundle
	cd $(JS_DIR) && bun x esbuild dist/bundle.cjs --target=es5 --format=cjs --outfile=dist/bundle.es5.cjs

js-clean:
	rm -rf $(JS_DIR)/dist

go-build: js-bundle
	go build .

go-run-bun: js-bundle
	go run .
