---
Title: "Protobuf Builders for Goja"
Slug: protobuf-builders-user-guide
Short: "Generate Goja-native fluent builders for protobuf messages and recover concrete Go messages without JSON round trips."
Topics:
- goja
- protobuf
- builders
- code-generation
- modules
- typescript
Commands:
- protoc-gen-goja-builder
- goja-repl
- gen-dts
Flags:
- module_name
- paths
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Generated protobuf builders let JavaScript running inside Goja construct real Go protobuf messages through a fluent API. JavaScript gets ergonomic `pb.Task.builder().title("...").build()` calls, while Go receives a concrete `proto.Message` through `protogoja.MessageFromValue` instead of converting through `JSON.stringify`, `protojson.Unmarshal`, or schema-name lookup.

Use this guide when you own a `.proto` schema and want to expose it as a CommonJS module inside `goja-repl`, an engine-managed runtime, xgoja, or a custom host application.

## What the generator creates

The generator emits a Go companion file next to normal `protoc-gen-go` output. Normal protobuf generation gives Go structs; `protoc-gen-goja-builder` gives Goja module helpers, JavaScript builder namespaces, hidden message references, TypeScript declarations, and host integration functions.

For a file such as `task.proto`, generation produces:

```text
proto/task.proto
proto/task.pb.go        # normal protoc-gen-go output
proto/task_goja.pb.go   # protoc-gen-goja-builder output
```

The generated companion file contains file-scoped helpers like:

```go
func GojaBuilderFileTaskProtoTypeScriptModule(moduleName string) *spec.Module
func NewGojaBuilderFileTaskProtoLoader(moduleName string) require.ModuleLoader
func RegisterGojaBuilderFileTaskProtoModule(reg *require.Registry, moduleName string) error
func GojaBuilderFileTaskProtoModule(moduleName string) modules.NativeModule
func RegisterGojaBuilderFileTaskProtoMessageTypes(register func(string, proto.Message) error) error
```

Each protobuf message becomes a JavaScript namespace with `builder()`, `from(value)`, `is(value)`, and `clone(value)`. The namespace is also a hidden prototype token, so Go code can recover the message type with `protogoja.MessagePrototypeFromValue` when it needs a schema handle rather than a payload.

## Quick start with protoc

Use `protoc` when your project already has direct protobuf generation commands or `go:generate` lines. Install both generators, then invoke them in the same command so `*.pb.go` and `*_goja.pb.go` stay in sync.

Install the plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/go-go-golems/go-go-goja/cmd/protoc-gen-goja-builder@latest
```

Generate Go protobuf code and Goja builders:

```bash
protoc \
  --proto_path=. \
  --go_out=. --go_opt=paths=source_relative \
  --goja-builder_out=. \
  --goja-builder_opt=paths=source_relative,module_name=examples.tasks.v1 \
  proto/task.proto
```

Commit both generated files. The `module_name` option is the default JavaScript module name that examples, docs, and xgoja declarations should use, even though hosts can still pass a custom module name to the generated loader helpers.

## Quick start with go:generate

Use `go:generate` when the schema belongs to one Go package and developers should be able to refresh generated files with one stable command.

```go
package taskpb

//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative --goja-builder_out=. --goja-builder_opt=paths=source_relative,module_name=examples.tasks.v1 task.proto
```

Then run:

```bash
go generate ./proto
go test ./proto ./pkg/that/consumes/protos -count=1
```

A package-local generation command is often the safest workflow because the protobuf schema, normal Go output, Goja companion output, and tests live close together.

## JavaScript builder API

Generated builders are fluent, mutable construction helpers. Built messages are stable Go-backed values that carry hidden protobuf references; scripts should build new messages or clone existing messages instead of mutating built objects as plain JavaScript records.

Given this schema:

```proto
message Task {
  string id = 1;
  string title = 2;
  repeated string tags = 3;
  map<string, string> labels = 4;
  TaskPriority priority = 5;
  google.protobuf.Timestamp due_at = 6;
  google.protobuf.Struct metadata = 7;
}
```

JavaScript can construct a real protobuf message:

```javascript
const pb = require("examples.tasks.v1")

const task = pb.Task.builder()
  .id("task-1")
  .title("Ship protobuf builders")
  .addTags("protobuf")
  .addTags("goja")
  .putLabels("component", "runtime")
  .priority(pb.TaskPriority.TASK_PRIORITY_HIGH)
  .dueAt(new Date("2026-06-12T20:00:00Z"))
  .metadata({ owner: "agent", reviewed: true })
  .build()
```

The generated API includes these field-shape helpers:

| Field shape | Generated methods | Notes |
| --- | --- | --- |
| Scalar / enum / message | `<field>(value)`, `clear<Field>()` | Replaces the field value. |
| Repeated | `<field>(values)`, `add<Field>(value)`, `clear<Field>()` | `add<Field>` appends one element. |
| Map | `<field>(objectOrMap)`, `put<Field>(key, value)`, `delete<Field>(key)`, `clear<Field>()` | Plain objects and JavaScript `Map` are accepted for map replacement. |
| Explicit presence | `has<Field>()`, `clear<Field>()` | Works for proto2 fields, `optional`, messages, and oneofs with protobuf presence. |
| Real oneof | `which<Oneof>()`, `clear<Oneof>()` | Synthetic proto3 optional backing oneofs are hidden. |

Enums are exported as JavaScript objects. Use generated enum constants rather than raw integers when readability matters:

```javascript
.priority(pb.TaskPriority.TASK_PRIORITY_HIGH)
```

## Recover concrete protobuf messages in Go

The important host-side rule is: do not ask JavaScript to serialize the message back to JSON just so Go can parse it again. Generated `build()` returns a Goja value with a hidden cloned protobuf reference, and `protogoja.MessageFromValue` recovers a fresh Go clone.

```go
value := exports.Get("task")

msg, ok := protogoja.MessageFromValue(value)
if !ok {
    return fmt.Errorf("expected generated protobuf message")
}

task, ok := msg.(*taskpb.Task)
if !ok {
    return fmt.Errorf("expected *taskpb.Task, got %T", msg)
}

fmt.Println(task.GetTitle())
```

Use `protogoja.MustMessageFromValue` in tests when a mismatch should fail immediately. Use `protogoja.TypeNameFromValue` or `protogoja.IsMessageValueOf` when a generic module needs to inspect the message type before dispatching.

## Install generated modules in a raw require registry

A raw Goja host can install a generated module with the generated registry helper. This is the smallest integration path and is useful in tests, CLIs, and custom embedders.

```go
vm := goja.New()
reg := require.NewRegistry()

if err := taskpb.RegisterGojaBuilderFileTaskProtoModule(reg, "examples.tasks.v1"); err != nil {
    return err
}
reg.Enable(vm)

_, err := vm.RunString(`
  const pb = require("examples.tasks.v1")
  globalThis.task = pb.Task.builder().title("demo").build()
`)
if err != nil {
    return err
}
```

This path does not require xgoja or a runtime factory. It is the direct CommonJS module loader path.

## Install generated modules in engine-managed runtimes

Use `engine.NativeModuleRegistrar` when every runtime created by an `engine.RuntimeFactory` should have the protobuf module available.

```go
factory, err := engine.NewRuntimeFactoryBuilder().
    WithModules(engine.NativeModuleRegistrar{
        ModuleName: "examples.tasks.v1",
        Loader:     taskpb.NewGojaBuilderFileTaskProtoLoader("examples.tasks.v1"),
    }).
    Build()
if err != nil {
    return err
}
```

This is the right integration point for applications that already centralize runtime construction through `pkg/engine`.

## Expose generated modules through xgoja providers

Use provider registration when a module should be selected from `xgoja.yaml`, built into generated xgoja binaries, and included in generated TypeScript declarations.

```go
func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("task-protos", providerapi.Module{
        Name:        "examples.tasks.v1",
        Description: "Generated builders for task protobuf messages",
        TypeScript:  taskpb.GojaBuilderFileTaskProtoTypeScriptModule("examples.tasks.v1"),
        NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
            return taskpb.NewGojaBuilderFileTaskProtoLoader("examples.tasks.v1"), nil
        },
    })
}
```

Then select it in `xgoja.yaml`:

```yaml
modules:
  - package: task-protos
    name: examples.tasks.v1
```

This gives JavaScript authors a typed `require("examples.tasks.v1")` module and lets `xgoja gen-dts` include the generated declarations.

## TypeScript declarations

Generated modules expose a `RawDTS` descriptor because protobuf builders need rich object, enum, map, repeated, and oneof types. Hosts do not have to hand-write declarations; use the generated `TypeScriptModule` helper.

```go
mod := taskpb.GojaBuilderFileTaskProtoTypeScriptModule("examples.tasks.v1")
```

When the module is registered with xgoja, declaration generation can be run from the xgoja project:

```bash
xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts --strict
```

The resulting declarations include message interfaces, builder interfaces, enum objects, repeated helpers, map helpers, optional-presence helpers, and real-oneof helpers.

## Runtime conversion rules

The runtime intentionally accepts only the inputs that preserve type meaning. This keeps protobuf construction predictable and avoids silently converting arbitrary JavaScript objects into arbitrary message graphs.

| Input target | Accepted JavaScript values |
| --- | --- |
| Ordinary message field | Generated protobuf message value or generated builder value. |
| `google.protobuf.Timestamp` | JavaScript `Date` or RFC3339 string. |
| `google.protobuf.Duration` | Go duration string such as `"1h2m3s"`. |
| `google.protobuf.Struct` | Plain JavaScript object. |
| `google.protobuf.Value` | JSON-shaped JavaScript value. |
| `google.protobuf.ListValue` | JavaScript array. |
| Wrapper types | Wrapped scalar value. |
| `google.protobuf.FieldMask` | Comma-separated string or string array. |
| `google.protobuf.Any` | Generated protobuf message value or builder value. |
| 64-bit integers | Safe JavaScript number or string; prefer string outside the safe integer range. |

Plain objects are not accepted for ordinary message fields. Build nested messages explicitly:

```javascript
const task = pb.Task.builder().title("nested").build()
const envelope = pb.TaskEnvelope.builder().task(task).build()
```

or pass the builder directly when you want the parent setter to build it:

```javascript
const envelope = pb.TaskEnvelope.builder()
  .task(pb.Task.builder().title("nested"))
  .build()
```

## Complete xgoja example

The repository contains a compiled provider example that exercises the full path from `.proto` to JavaScript and back to Go:

```text
examples/xgoja/15-protobuf-builder-provider/
```

Run it with:

```bash
cd examples/xgoja/15-protobuf-builder-provider
make smoke
```

The example generates protobuf code, registers a provider module, runs JavaScript from `scripts/build-task.js`, and verifies concrete `*taskpb.Task` and `*taskpb.TaskEnvelope` extraction in Go tests.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `protoc-gen-goja-builder: program not found` | The plugin is not on `PATH`. | Run `go install github.com/go-go-golems/go-go-goja/cmd/protoc-gen-goja-builder@latest` or install from your local checkout. |
| `require("...")` fails | The generated module loader was not registered with the runtime's require registry. | Call the generated `RegisterGojaBuilderFile...Module`, use `engine.NativeModuleRegistrar`, or register an xgoja provider module. |
| Go receives a plain object instead of a protobuf message | JavaScript code serialized or copied the object instead of passing the built value. | Pass the result of `.build()` directly and recover it with `protogoja.MessageFromValue`. |
| Ordinary nested message setter rejects a plain object | Plain objects are intentionally accepted only for JSON-shaped WKTs. | Build the nested message with its generated builder and pass the message or builder. |
| `Task_LabelsEntry` or another map-entry symbol appears in generated API | The generator is too old and is emitting synthetic map-entry messages. | Regenerate with a version that skips `Descriptor.IsMapEntry()` messages. |
| TypeScript output misses the module | The provider did not expose `TypeScript` or `xgoja.yaml` did not select the module. | Use the generated `GojaBuilderFile...TypeScriptModule(moduleName)` helper in provider registration. |

## See Also

- `cmd/protoc-gen-goja-builder/README.md` — generator-specific command and host integration reference.
- `examples/xgoja/15-protobuf-builder-provider/` — compiled end-to-end provider example.
- `pkg/protogoja` — runtime package for message refs, prototype refs, and builder refs.
- `xgoja help tutorial-protobuf-builder-provider` — xgoja-focused provider tutorial once the xgoja CLI docs are available.
- `goja-repl help typescript-declaration-generator` — shared TypeScript declaration generation guide.
