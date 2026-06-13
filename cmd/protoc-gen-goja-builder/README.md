# protoc-gen-goja-builder

`protoc-gen-goja-builder` generates Go companion files that expose protobuf messages as Goja-native fluent builders. The generated API keeps protobuf messages Go-backed: JavaScript code receives lightweight objects with hidden protobuf references, and Go code recovers concrete messages through `protogoja.MessageFromValue` instead of serializing through JSON/protojson.

## Generated surface

For each generated proto file, the plugin emits one `*_goja.pb.go` file with file-scoped helpers:

```go
func GojaBuilderFile<ProtoFile>ModuleName() string
func GojaBuilderFile<ProtoFile>MessageTypes() []string
func GojaBuilderFile<ProtoFile>TypeScriptModule(moduleName string) *spec.Module
func NewGojaBuilderFile<ProtoFile>Loader(moduleName string) require.ModuleLoader
func RegisterGojaBuilderFile<ProtoFile>Module(reg *require.Registry, moduleName string) error
func GojaBuilderFile<ProtoFile>Module(moduleName string) modules.NativeModule
func RegisterGojaBuilderFile<ProtoFile>MessageTypes(register func(string, proto.Message) error) error
```

For each message, it emits:

```go
func New<Message>GojaNamespace(vm *goja.Runtime) (*goja.Object, error)
func New<Message>GojaBuilder(vm *goja.Runtime) (*goja.Object, error)
```

The JavaScript namespace has `typeName`, `builder()`, `from(value)`, `is(value)`, and `clone(value)`. Builders expose fluent field setters, `clear<Field>()`, `build()`, `clone()`, repeated `add<Field>()`, map `put<Field>()`/`delete<Field>()`, explicit-presence `has<Field>()`, and real-oneof `which<Oneof>()`/`clear<Oneof>()` helpers.

## JavaScript usage

Given a generated module named `hashiplugin.contract.v1`:

```javascript
const pb = require("hashiplugin.contract.v1")

const manifest = pb.ModuleManifest.builder()
  .moduleName("demo")
  .version("v1")
  .addCapabilities("tools")
  .exports([
    pb.ExportSpec.builder()
      .name("run")
      .kind(pb.ExportKind.EXPORT_KIND_FUNCTION)
  ])
  .build()
```

`manifest` is not plain JSON. It is a Goja object carrying a hidden `proto.Message` reference. Go host code can extract the concrete message without a JSON round trip.

## Raw `require.Registry` integration

Generated files can register themselves in a raw Goja CommonJS registry:

```go
package host

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
)

func installRawRegistry() (*goja.Runtime, *require.Registry, error) {
	vm := goja.New()
	reg := require.NewRegistry()

	if err := contract.RegisterGojaBuilderFileJsmoduleProtoModule(reg, "hashiplugin.contract.v1"); err != nil {
		return nil, nil, err
	}
	reg.Enable(vm)
	return vm, reg, nil
}
```

After registration, scripts can call:

```javascript
const pb = require("hashiplugin.contract.v1")
const req = pb.InvokeRequest.builder().exportName("run").build()
```

## `engine.NativeModuleRegistrar` integration

For engine-managed runtimes, wrap the generated loader in `engine.NativeModuleRegistrar` and pass it to the runtime factory builder:

```go
package host

import (
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
)

func newFactory() (*engine.RuntimeFactory, error) {
	return engine.NewRuntimeFactoryBuilder().
		WithModules(engine.NativeModuleRegistrar{
			ModuleName: "hashiplugin.contract.v1",
			Loader:     contract.NewGojaBuilderFileJsmoduleProtoLoader("hashiplugin.contract.v1"),
		}).
		Build()
}
```

Use this form when protobuf builders should be available in every runtime created by the factory.

## xgoja provider integration

Generated modules can also be exposed as provider modules. The generated `TypeScriptModule` function gives the provider a DTS descriptor that `xgoja/dtsgen` can bundle for editors and agents:

```go
package host

import (
	"github.com/dop251/goja_nodejs/require"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func registerProvider(registry *providerapi.ProviderRegistry) error {
	return registry.Package("hashiplugin", providerapi.Module{
		Name:       "hashiplugin.contract.v1",
		TypeScript: contract.GojaBuilderFileJsmoduleProtoTypeScriptModule("hashiplugin.contract.v1"),
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return contract.NewGojaBuilderFileJsmoduleProtoLoader("hashiplugin.contract.v1"), nil
		},
	})
}
```

This is the preferred shape when users select modules through xgoja provider/package configuration rather than hard-coding every module into a runtime factory.

## Consuming-module demonstration: avoid JSON/protojson

A native Goja module can accept a generated protobuf object and recover the Go message directly:

```go
package host

import (
	"fmt"

	"github.com/dop251/goja"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/protogoja"
)

func consumeManifest(vm *goja.Runtime, value goja.Value) error {
	msg, ok := protogoja.MessageFromValue(value)
	if !ok {
		return fmt.Errorf("expected generated protobuf message")
	}
	manifest, ok := msg.(*contract.ModuleManifest)
	if !ok {
		return fmt.Errorf("expected ModuleManifest, got %T", msg)
	}

	// Use manifest directly. No JSON.stringify, protojson.Unmarshal, or
	// descriptor lookup is needed.
	_ = manifest.GetModuleName()
	return nil
}
```

For APIs that need a schema handle rather than a concrete payload, accept the generated namespace and recover its prototype token:

```go
func consumeMessageType(value goja.Value) error {
	prototype, ok := protogoja.MessagePrototypeFromValue(value)
	if !ok {
		return fmt.Errorf("expected generated message namespace")
	}
	_ = prototype.TypeName()
	_ = prototype.NewMessage()
	return nil
}
```

## `protoc` workflow

Install both Go plugins on `PATH`:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/go-go-golems/go-go-goja/cmd/protoc-gen-goja-builder@latest
```

Generate Go protobuf types and Goja builder companions:

```bash
protoc \
  --proto_path=. \
  --go_out=. --go_opt=paths=source_relative \
  --goja-builder_out=. \
  --goja-builder_opt=paths=source_relative,module_name=hashiplugin.contract.v1 \
  pkg/hashiplugin/contract/jsmodule.proto
```

The generated `*_goja.pb.go` file should be committed next to the corresponding `*.pb.go` file.

## Buf workflow

Add a plugin entry to `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: .
    opt:
      - paths=source_relative
  - local: protoc-gen-goja-builder
    out: .
    opt:
      - paths=source_relative
      - module_name=hashiplugin.contract.v1
```

Then run:

```bash
buf generate
```

Use a package-specific `module_name` when generating one logical JavaScript module per proto package. If a Go package contains multiple proto files, each generated companion file has file-scoped helper names to avoid symbol collisions.

## `go:generate` workflow

A package-local generator command can keep protobuf, builder, and module surfaces reproducible:

```go
//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative --goja-builder_out=. --goja-builder_opt=paths=source_relative,module_name=hashiplugin.contract.v1 jsmodule.proto
```

Then run:

```bash
go generate ./pkg/hashiplugin/contract
```

Prefer running `go test ./cmd/protoc-gen-goja-builder ./pkg/protogoja -count=1` after generator changes and `go test ./pkg/tsgen/... ./pkg/xgoja/dtsgen ./cmd/protoc-gen-goja-builder ./pkg/protogoja -count=1` after TypeScript declaration changes.

## Input rules

Generated builders delegate conversions to `pkg/protogoja`:

- ordinary message fields accept generated `ProtoMessage` values or generated builder objects;
- arbitrary plain objects are **not** accepted for ordinary message fields;
- `Struct`, `Value`, and `ListValue` accept JSON-shaped JavaScript values;
- `Timestamp` accepts RFC3339 strings and JavaScript `Date` values;
- `Duration` accepts Go duration strings such as `"1h2m3s"`;
- `Any` can wrap generated/built protobuf messages;
- wrapper types accept their scalar value;
- `FieldMask` accepts a comma-separated string or a string array;
- 64-bit integers should be passed as strings when outside JavaScript's safe integer range.
