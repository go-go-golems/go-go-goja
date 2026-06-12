---
Title: "Tutorial: Protobuf builder provider modules"
Slug: tutorial-protobuf-builder-provider
Short: "Expose generated protobuf fluent builders as xgoja provider modules with TypeScript declarations."
Topics:
- xgoja
- protobuf
- providers
- modules
- typescript
- goja
Commands:
- xgoja build
- xgoja gen-dts
- xgoja doctor
Flags:
- --out
- --strict
- --xgoja-replace
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Generated protobuf builder modules let xgoja scripts create real Go protobuf messages through `require()` while the host keeps concrete Go types. This is useful when JavaScript authors need a friendly construction API, but Go modules, handlers, or providers must receive `*mypb.Message` values without JSON round trips.

This tutorial shows the provider shape used by `examples/xgoja/15-protobuf-builder-provider`: a local `.proto` file is compiled with `protoc-gen-go` and `protoc-gen-goja-builder`, the generated loader is registered as a provider module, xgoja selects it through `xgoja.yaml`, and tests recover concrete protobuf messages through `protogoja.MessageFromValue`.

## What you will build

The end result is a normal xgoja provider package that exports a generated protobuf module:

```javascript
const pb = require("examples.xgoja.protobuf.v1")

const task = pb.Task.builder()
  .id("task-1")
  .title("Ship protobuf builders")
  .addTags("xgoja")
  .putLabels("component", "provider")
  .priority(pb.TaskPriority.TASK_PRIORITY_HIGH)
  .dueAt(new Date("2026-06-12T20:00:00Z"))
  .metadata({ owner: "agent", reviewed: true })
  .build()
```

That `task` value is not JSON. It is a Goja object carrying a hidden cloned protobuf message, recoverable by Go code with `protogoja.MessageFromValue`.

## 1. Define and generate the protobuf package

Start with a concrete `.proto` schema. The generated builders are most valuable when the schema exercises normal protobuf shapes such as repeated fields, maps, enums, nested messages, and well-known types.

```proto
syntax = "proto3";

package examples.xgoja.protobuf.v1;

option go_package = "github.com/example/app/proto;taskpb";

import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";

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

Generate both normal Go protobuf code and the Goja companion file:

```bash
protoc \
  -I . \
  --go_out=. --go_opt=paths=source_relative \
  --goja-builder_out=. \
  --goja-builder_opt=paths=source_relative,module_name=examples.xgoja.protobuf.v1 \
  proto/task.proto
```

A package-local `go:generate` file is usually better than asking every developer to remember this command:

```go
package taskpb

//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative --goja-builder_out=. --goja-builder_opt=paths=source_relative,module_name=examples.xgoja.protobuf.v1 task.proto
```

## 2. Register the generated loader as a provider module

The provider module is intentionally small because the generated file already knows how to create the CommonJS loader and TypeScript descriptor.

```go
package provider

import (
    "github.com/dop251/goja_nodejs/require"
    taskpb "github.com/example/app/proto"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const ModuleName = "examples.xgoja.protobuf.v1"

func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("protobuf-builder-example", providerapi.Module{
        Name:        ModuleName,
        Description: "Generated Goja protobuf builders for the example schema",
        TypeScript:  taskpb.GojaBuilderFileTaskProtoTypeScriptModule(ModuleName),
        NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
            return taskpb.NewGojaBuilderFileTaskProtoLoader(ModuleName), nil
        },
    })
}
```

The important pieces are:

| Provider field | Value | Why it matters |
| --- | --- | --- |
| `Name` | The JavaScript `require()` module name. | Scripts import this exact string. |
| `TypeScript` | Generated `GojaBuilderFile...TypeScriptModule`. | `xgoja gen-dts` can emit editor declarations. |
| `NewModuleFactory` | Generated `NewGojaBuilderFile...Loader`. | xgoja can install the module in each runtime. |

## 3. Select the module in xgoja.yaml

Once a provider package registers the module, xgoja projects select it like any other provider module.

```yaml
modules:
  - package: protobuf-builder-example
    name: examples.xgoja.protobuf.v1
```

Use an `as:` alias only when JavaScript should import a different name than the generated default:

```yaml
modules:
  - package: protobuf-builder-example
    name: examples.xgoja.protobuf.v1
    as: pb:tasks
```

If you use an alias, scripts should call `require("pb:tasks")`, and generated TypeScript declarations should also use the alias.

## 4. Write JavaScript against the generated API

Generated builders expose fluent setters and shape-specific helpers. The most common pattern is to build messages in JavaScript and return or export them for a Go module to consume.

```javascript
const pb = require("examples.xgoja.protobuf.v1")

const task = pb.Task.builder()
  .id("task-1")
  .title("Review generated protobuf builders")
  .addTags("protobuf")
  .addTags("xgoja")
  .putLabels("component", "goja")
  .priority(pb.TaskPriority.TASK_PRIORITY_HIGH)
  .dueAt(new Date("2026-06-12T20:00:00Z"))
  .metadata({ owner: "agent", reviewed: true })
  .build()

exports.task = task
exports.envelope = pb.TaskEnvelope.builder().task(task).build()
```

Use generated builder values or built message values for nested ordinary messages. Plain objects are accepted for JSON-shaped well-known types such as `Struct`, but they are intentionally rejected for arbitrary message fields.

## 5. Recover concrete messages in Go

A consuming provider, command, or host can recover the concrete protobuf message directly from the JavaScript value.

```go
msg, ok := protogoja.MessageFromValue(exports.Get("task"))
if !ok {
    return fmt.Errorf("expected generated protobuf message")
}

task, ok := msg.(*taskpb.Task)
if !ok {
    return fmt.Errorf("expected *taskpb.Task, got %T", msg)
}

_ = task.GetLabels()["component"]
```

This is the main reason to use generated protobuf builders instead of plain JavaScript objects. The value crosses the JavaScript/Go boundary as a typed protobuf payload, not as a string that has to be parsed and validated again.

## 6. Generate declarations for editors and agents

Provider modules that set the `TypeScript` field participate in xgoja declaration generation.

```bash
xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts --strict
```

The declaration contains message namespaces, builder interfaces, enum exports, repeated helpers, map helpers, optional-presence helpers, and oneof helpers. Put the generated `.d.ts` file under a source root that your editor or `tsconfig.json` includes.

## 7. Run the compiled example

The repository includes a complete working example:

```bash
cd examples/xgoja/15-protobuf-builder-provider
make smoke
```

The smoke target regenerates code, runs provider tests, validates xgoja configuration, lists selected modules, generates declarations, and builds a generated xgoja runtime. Start there when copying the pattern into another project.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `xgoja gen-dts` omits the protobuf module | The provider module did not set `TypeScript`, or `xgoja.yaml` did not select the module. | Use the generated `GojaBuilderFile...TypeScriptModule(ModuleName)` helper and verify `xgoja doctor -f xgoja.yaml`. |
| `require("examples.xgoja.protobuf.v1")` fails | The provider package is not registered or the module name differs from `xgoja.yaml`. | Check the provider `Name`, package id, and any `as:` alias. |
| Go receives the wrong message type | JavaScript exported a different built message or the Go side asserted the wrong concrete type. | Check `protogoja.TypeNameFromValue(value)` before the concrete type assertion. |
| Nested message setter rejects `{ ... }` | Ordinary messages do not accept arbitrary plain objects. | Build the nested message with its generated namespace and pass the built message or builder. |
| Forcing dependency upgrades breaks the generated runtime build | xgoja generated binaries depend on the selected provider/toolchain versions. | Keep protobuf-builder changes separate from broad JavaScript dependency upgrades and validate with `make smoke`. |

## See Also

- `goja-repl help protobuf-builders-user-guide` — shared guide for generator, runtime, and host integration.
- `goja-repl help typescript-declaration-generator` — declaration generation concepts and drift checks.
- `xgoja help tutorial-typescript-declarations` — xgoja-specific `.d.ts` workflow.
- `examples/xgoja/15-protobuf-builder-provider/` — complete compiled provider example.
- `cmd/protoc-gen-goja-builder/README.md` — generator reference and raw host integration examples.
