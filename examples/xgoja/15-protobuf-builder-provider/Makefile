REPO_ROOT := $(abspath ../../..)
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja
BIN := $(CURDIR)/dist/protobuf-builder-provider

.PHONY: generate test doctor list-modules build smoke clean

generate:
	cd $(REPO_ROOT) && go generate ./examples/xgoja/15-protobuf-builder-provider/proto

test:
	cd $(REPO_ROOT) && go test ./examples/xgoja/15-protobuf-builder-provider/... -count=1

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

list-modules:
	$(XGOJA) list-modules -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work

smoke: test doctor list-modules

clean:
	rm -rf $(CURDIR)/dist
