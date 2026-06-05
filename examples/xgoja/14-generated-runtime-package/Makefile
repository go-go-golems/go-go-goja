REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)

.PHONY: generate smoke clean

generate:
	rm -rf $(EXAMPLE_DIR)/internal/xgojaruntime
	cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja generate \
		-f $(EXAMPLE_DIR)/xgoja.yaml \
		--output $(EXAMPLE_DIR)/internal/xgojaruntime \
		--package xgojaruntime

smoke: generate
	cd $(REPO_ROOT) && GOWORK=off go run ./examples/xgoja/14-generated-runtime-package/cmd/host | grep 'hello package host'

clean:
	rm -rf $(EXAMPLE_DIR)/internal/xgojaruntime
