---
Title: Generated Goja protobuf fluent builders design
Ticket: GOJA-PB-001
Status: active
Topics:
    - goja
    - protobuf
    - bindings
    - typescript
    - codegen
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/common.go
      Note: NativeModule and default registry APIs that generated protobuf builder modules should implement or register through
    - Path: go-go-goja/modules/events/events.go
      Note: Example of a Go-backed JS class/module with handwritten TypeScript RawDTS and hidden Go object behavior
    - Path: go-go-goja/pkg/engine/module_specs.go
      Note: NativeModuleRegistrar and runtime module registration surface for generated loaders
    - Path: go-go-goja/pkg/hashiplugin/contract/internal/cmd/generate/main.go
      Note: Existing repo-local protoc generation helper pattern
    - Path: go-go-goja/pkg/hashiplugin/contract/jsmodule.proto
      Note: Existing protobuf schema in go-go-goja useful as fixture candidate
    - Path: go-go-goja/pkg/protogoja/builder.go
      Note: Initial implementation of the BuilderRef runtime helper proposed by the design
    - Path: go-go-goja/pkg/protogoja/builder_test.go
      Note: Executable validation of the first BuilderRef conversion slice
    - Path: go-go-goja/pkg/protogoja/ref.go
      Note: Initial implementation of the MessageRef/ProtoMessage contract proposed by the design
    - Path: go-go-goja/pkg/protogoja/ref_test.go
      Note: Executable validation of the Phase 1 ProtoMessage contract
    - Path: go-go-goja/pkg/tsgen/render/dts_renderer.go
      Note: Renderer behavior for RawDTS-backed generated declaration modules
    - Path: go-go-goja/pkg/tsgen/spec/types.go
      Note: TypeScript declaration model that generated modules can feed into
    - Path: go-go-goja/pkg/xgoja/dtsgen/dtsgen.go
      Note: xgoja provider TypeScript bundling path that builder modules should integrate with
    - Path: sessionstream/examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto
      Note: Representative external schema for acceptance tests with command/event-style payloads
ExternalSources: []
Summary: Design for a protoc plugin that generates reusable Goja fluent builders for protobuf messages, with phase-2 generated builders as the first implementation target.
LastUpdated: 2026-06-12T16:15:00-04:00
WhatFor: Planning a generic go-go-goja feature that lets any Goja-consuming code construct concrete proto.Message values through generated fluent JavaScript builders instead of JSON/protojson conversion.
WhenToUse: Use before implementing protoc-gen-goja-builder, reviewing generated APIs, or integrating generated protobuf builders with sessionstream or other Goja modules.
---



# Generated Goja protobuf fluent builders design

## Executive summary

This document designs a reusable `go-go-goja` feature: a protobuf compiler plugin that generates Goja-native fluent builders for protobuf messages. The goal is to let JavaScript code running under Goja construct concrete Go `proto.Message` values without first building plain JavaScript objects and then round-tripping through JSON/protojson.

The first implementation should target what the earlier conversation called “phase 2”: generated builders, not just a generic reflection prototype. A generic runtime helper is still useful internally, but the product users see should be generated JavaScript modules such as:

```js
const chat = require("sessionstream.examples.chatdemo.v1");

const cmd = chat.StartInferenceCommand.builder()
  .prompt("Explain ordinals")
  .build();

hub.submit("session-1", "ChatStartInference", cmd);
```

Here `cmd` is not a plain object. It is a Go-backed Goja object carrying a hidden `proto.Message` reference. A consuming module such as future `sessionstream` bindings can detect that reference, clone or validate the concrete proto message, and pass it into Go APIs directly.

The proposed implementation has three parts:

1. A small runtime support package in `go-go-goja`, tentatively `pkg/protogoja`, that defines message references, builder runtime helpers, field conversion helpers, and module registration support.
2. A protoc plugin, tentatively `cmd/protoc-gen-goja-builder`, built with `google.golang.org/protobuf/compiler/protogen`, that generates companion `.go` files next to `protoc-gen-go` output.
3. Generated module loaders and TypeScript declarations so the builders work both with raw `require.Registry` setup and with xgoja/provider DTS generation.

The first version should support scalar fields, enums, nested messages, repeated fields, maps, oneofs, optional fields, common well-known types, hidden message references, TypeScript declarations, and integration hooks for consuming Goja modules. It should not stop at a reflection-only `proto.message("pkg.Type").set("field", value)` API, although that API can exist as a debugging fallback.

## Problem statement

Goja-consuming code often sits at the boundary between JavaScript authoring ergonomics and Go's strongly typed APIs. Protobuf-heavy frameworks such as `sessionstream` want concrete `proto.Message` payloads for commands, events, UI events, and timeline entities. The simplest JavaScript binding accepts plain objects and converts them with protojson:

```js
hub.submit("session-1", "ChatStartInference", {
  prompt: "Explain ordinals",
});
```

The binding then does:

```go
raw := JSON.stringify(value)
msg := registry.NewMessage("ChatStartInference")
protojson.Unmarshal(raw, msg)
```

This is workable, but it has real costs:

- It creates extra JSON serialization and parsing overhead inside one process.
- Errors happen at runtime and often point at JSON mapping rather than the builder call that made the invalid value.
- JavaScript autocomplete cannot easily discover legal fields, enums, oneof choices, or nested message builders.
- int64/uint64 behavior is ambiguous unless every binding re-documents and re-validates it.
- Consuming modules need repeated ad-hoc code to detect plain objects, marshal JSON, instantiate prototypes, and unmarshal.

A generated builder module solves the problem at the right layer: the `.proto` file remains the schema source of truth, Go codegen still produces the canonical Go message types, and `go-go-goja` adds a Goja-specific construction API that returns real protobuf messages.

## Current-state analysis

### Native module shape in go-go-goja

`go-go-goja/modules/common.go` defines the core `modules.NativeModule` interface with `Name`, `Doc`, and `Loader(*goja.Runtime, *goja.Object)` methods. The module registry can enable all registered modules by calling `RegisterNativeModule` on a `goja_nodejs/require.Registry` (`modules/common.go:28-31`, `85-89`).

Generated protobuf builder modules should fit this shape. They can expose a `Module(requireName string, opts ...Option) modules.NativeModule` function, or they can expose `NewLoader` and `Register` helpers like Geppetto does. The key point is that a generated module should feel like any other Goja native module.

### Existing modules already use Go-backed JS objects and RawDTS

`go-go-goja/modules/events/events.go` is an important precedent. It exposes a Go-backed `EventEmitter` class to JavaScript and returns a `spec.Module` with handwritten `RawDTS` lines (`events.go:92-115`). Its loader creates a constructor, a prototype object, methods, and exports the constructor as both CommonJS default and named export (`events.go:118-195`).

The builder generator should use the same approach:

- construct Go-backed builder/message objects;
- attach methods to prototypes;
- export message namespaces or constructors;
- emit `RawDTS` for rich TypeScript declarations that are more expressive than the current structural `spec.Function` model.

### Runtime module registration is already available

`go-go-goja/pkg/engine/module_specs.go` provides `NativeModuleRegistrar`, which registers one native module loader with a runtime module registration context (`module_specs.go:51-77`). This is useful for generated builders because a caller may not want global `modules.Register` side effects. A generated package can expose:

```go
func RuntimeModule(requireName string) engine.NativeModuleRegistrar
```

or simply return a loader that the host registers through existing xgoja or engine configuration.

### TypeScript declaration generation supports raw module text

`go-go-goja/pkg/tsgen/spec/types.go` defines `spec.Module` with `RawDTS` (`types.go:26-31`). `go-go-goja/pkg/tsgen/render/dts_renderer.go` writes a `declare module "name"` block and appends each non-empty RawDTS line (`dts_renderer.go:49-79`). `go-go-goja/pkg/xgoja/dtsgen/dtsgen.go` clones provider module TypeScript descriptors, rewrites the module name to the selected alias, validates them, and bundles them (`dtsgen.go:52-93`).

Generated protobuf modules should therefore produce a `*spec.Module` descriptor with rich `RawDTS`. That allows xgoja-generated runtimes to include builder declarations without building a second TypeScript generation pipeline.

### Protobuf tooling is already present in go-go-goja

`go-go-goja/go.mod` already depends on `google.golang.org/protobuf v1.36.11`. The repository also has existing protobuf generation for the hashiplugin contract and a local generator helper that installs fixed `protoc-gen-go` and `protoc-gen-go-grpc` versions before running `protoc` (`pkg/hashiplugin/contract/internal/cmd/generate/main.go:12-60`).

This means a new plugin can use official protobuf Go APIs without introducing an unfamiliar dependency family:

```go
import "google.golang.org/protobuf/compiler/protogen"
```

### Good fixture schemas are available

Two useful fixtures already exist in the workspace:

- `go-go-goja/pkg/hashiplugin/contract/jsmodule.proto` has messages, repeated fields, enums, and `google.protobuf.Value`.
- `sessionstream/examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto` has command/event-style payloads and mirrors the likely first consumer of generated builders.

## Design goals

The generator should satisfy these goals:

1. **Concrete proto output:** `.build()` returns a Go-backed value from which Go can recover a `proto.Message` without JSON.
2. **Fluent JavaScript API:** builders should be discoverable and chainable.
3. **TypeScript-first authoring:** generated `.d.ts` should give useful autocomplete for fields, enums, oneofs, maps, and nested messages.
4. **Reusable across Goja consumers:** no `sessionstream` dependency in the core generator or runtime support package.
5. **Protoc/Bbuf friendly:** plugin should work with standard `protoc` and Buf plugin invocation patterns.
6. **Safe Goja ownership assumptions:** generated builder objects are ordinary Goja values and must not be used concurrently outside the owning runtime unless first unwrapped/cloned by Go.
7. **Low friction for Go hosts:** generated packages should provide `Register`, `NewLoader`, `Module`, and `TypeScriptModule` helpers.
8. **Precise errors:** invalid field values should fail at the builder call site with field names and expected types.
9. **No top-level Struct escape hatch:** generated builders should strengthen protobuf contracts, not recreate arbitrary JSON payloads.

## Non-goals for the first version

The first version should not attempt to solve every protobuf and packaging problem:

- Do not generate TypeScript npm packages. It only generates Go code that serves TypeScript declarations through go-go-goja's existing DTS mechanisms.
- Do not replace `protoc-gen-go` or generate Go message types.
- Do not require sessionstream or any other consuming framework.
- Do not support dynamic descriptors as the primary authoring path. Generated builders are the primary first target.
- Do not optimize every setter into direct field assignment initially if a common protoreflect setter reduces correctness risk.
- Do not support untrusted sandbox security claims; hidden Go references are an embedding convenience, not a security boundary.

## Proposed user-facing API

### Generated JavaScript module

Given a proto package such as:

```proto
syntax = "proto3";
package sessionstream.examples.chatdemo.v1;

message StartInferenceCommand {
  string prompt = 1;
}

message UserMessageAcceptedEvent {
  string message_id = 1;
  string role = 2;
  string content = 3;
  bool streaming = 4;
}
```

The generated module should be usable as:

```js
const chat = require("sessionstream.examples.chatdemo.v1");

const cmd = chat.StartInferenceCommand.builder()
  .prompt("Explain ordinals")
  .build();

const ev = chat.UserMessageAcceptedEvent.builder()
  .messageId("m1-user")
  .role("user")
  .content("Explain ordinals")
  .streaming(false)
  .build();
```

### Message namespace shape

Each message export is a namespace-like object:

```ts
interface MessageType<TMessage, TBuilder> {
  readonly typeName: string;
  builder(): TBuilder;
  from(message: TMessage): TBuilder;
  is(value: unknown): value is TMessage;
  clone(value: TMessage): TMessage;
}
```

For JavaScript this means:

```js
chat.StartInferenceCommand.typeName;
chat.StartInferenceCommand.builder();
chat.StartInferenceCommand.is(value);
chat.StartInferenceCommand.clone(value);
```

### Builder shape

A generated builder should be mutable and chainable:

```ts
interface StartInferenceCommandBuilder {
  prompt(value: string): this;
  clearPrompt(): this;
  build(): StartInferenceCommand;
  clone(): StartInferenceCommandBuilder;
}
```

`build()` should clone before returning, so subsequent mutations to the builder do not mutate the already-built message reference.

### Message reference shape

A built message should be a Go-backed JS object with a hidden proto reference and a small public API:

```ts
interface ProtoMessage<TTypeName extends string = string> {
  readonly typeName: TTypeName;
  toJSON(): unknown;
  clone(): ProtoMessage<TTypeName>;
  equals(other: unknown): boolean;
}
```

It should not expose raw Go struct fields as enumerable JavaScript properties. That avoids surprising `Export()` behavior and keeps mutation controlled through builders.

## Generated Go API

The generated Go file should be a companion to `protoc-gen-go` output. If `protoc-gen-go` emits `chat.pb.go`, this plugin emits something like `chat_goja.pb.go` in the same Go package.

Generated package-level API:

```go
// Code generated by protoc-gen-goja-builder. DO NOT EDIT.

func NewGojaLoader(opts ...gojapb.Option) require.ModuleLoader
func RegisterGojaModule(reg *require.Registry, opts ...gojapb.Option)
func GojaModule(opts ...gojapb.Option) modules.NativeModule
func TypeScriptModule(moduleName string) *spec.Module
func RegisterMessageTypes(reg *gojapb.Registry)
```

A host can choose the integration style:

```go
// Direct require registry.
chatv1.RegisterGojaModule(requireRegistry, gojapb.WithModuleName("chat.v1"))

// Engine runtime module.
runtimeModule := engine.NativeModuleRegistrar{
  ModuleName: "chat.v1",
  Loader: chatv1.NewGojaLoader(gojapb.WithModuleName("chat.v1")),
}

// xgoja provider descriptor.
providerapi.Module{
  Name: "chat.v1",
  DefaultAs: "chat.v1",
  TypeScript: chatv1.TypeScriptModule("chat.v1"),
  NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
    return chatv1.NewGojaLoader(gojapb.WithModuleName(ctx.As)), nil
  },
}
```

## Runtime support package

Add a package such as:

```text
go-go-goja/pkg/protogoja/
├── ref.go          # MessageRef, FromValue, ToValue, hidden ref attach/extract
├── builder.go      # generic builder object helpers
├── fields.go       # protoreflect field conversion and setter helpers
├── module.go       # generated module helper types
├── typescript.go   # shared DTS snippets and type naming helpers
├── wellknown.go    # Timestamp/Duration/Any/Struct/ListValue/Value helpers
└── errors.go       # field-path-rich errors
```

### MessageRef

`MessageRef` is the core bridge type:

```go
type MessageRef struct {
    msg proto.Message
    typeName protoreflect.FullName
}

func NewMessageRef(msg proto.Message) (*MessageRef, error)
func FromValue(value goja.Value) (proto.Message, bool)
func MustFromValue(vm *goja.Runtime, value goja.Value) proto.Message
func ToValue(vm *goja.Runtime, msg proto.Message) (*goja.Object, error)
```

`ToValue` should create a JS object with methods like `toJSON`, `clone`, and `equals`, then attach the Go reference as a non-enumerable hidden property.

### BuilderRef

`BuilderRef` owns a mutable message while fields are being set:

```go
type BuilderRef struct {
    msg proto.Message
    desc protoreflect.MessageDescriptor
}

func NewBuilder(msg proto.Message) *BuilderRef
func (b *BuilderRef) Set(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) error
func (b *BuilderRef) Add(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) error
func (b *BuilderRef) Put(vm *goja.Runtime, field protoreflect.FieldDescriptor, key, value goja.Value) error
func (b *BuilderRef) Clear(field protoreflect.FieldDescriptor)
func (b *BuilderRef) Build() proto.Message
func (b *BuilderRef) Clone() *BuilderRef
```

Generated builder methods can call these helpers rather than duplicating conversion code in every generated method.

### Why generated methods can still use reflection internally

The generated JS API is the important value. It gives users typed methods and TypeScript declarations. Internally, the first implementation can call a common protoreflect setter for correctness:

```go
_ = builderObj.Set("prompt", func(call goja.FunctionCall) goja.Value {
    if err := ref.Set(vm, fieldStartInferenceCommandPrompt, call.Argument(0)); err != nil {
        panic(vm.NewGoError(err))
    }
    return call.This
})
```

This still avoids JSON/protojson. It also centralizes tricky behavior such as `int64`, maps, oneofs, optional fields, bytes, and well-known types. Later, direct generated field assignment can optimize hot paths.

## Compiler plugin architecture

### Command layout

Add:

```text
go-go-goja/cmd/protoc-gen-goja-builder/main.go
```

The entry point should use `protogen`:

```go
func main() {
    opts := protogen.Options{
        ParamFunc: flags.Set,
    }
    opts.Run(func(plugin *protogen.Plugin) error {
        for _, file := range plugin.Files {
            if !file.Generate {
                continue
            }
            generateFile(plugin, file, config)
        }
        return nil
    })
}
```

### Plugin options

Support these options in version 1:

```text
module_name=<require-name>          # default derived from proto package
paths=source_relative|import        # mirror protoc-gen-go convention where practical
emit_dts=true|false                 # default true
emit_provider=true|false            # default false, see provider section
register_global=true|false          # default false; avoid surprise init registration
builder_suffix=Builder              # default Builder
message_ref_name=ProtoMessage       # TypeScript helper name
```

Example `protoc` invocation:

```bash
protoc \
  -I proto \
  --go_out=gen --go_opt=paths=source_relative \
  --goja-builder_out=gen --goja-builder_opt=paths=source_relative,module_name=sessionstream.examples.chatdemo.v1 \
  proto/sessionstream/examples/chatdemo/v1/chat.proto
```

Example Buf v2 config:

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt:
      - paths=source_relative
  - local: protoc-gen-goja-builder
    out: gen
    opt:
      - paths=source_relative
      - module_name=sessionstream.examples.chatdemo.v1
```

### Generated file content

Each generated file should include:

1. message descriptor table;
2. loader constructor;
3. per-message export installer;
4. per-message builder prototype installer;
5. enum exports;
6. TypeScript descriptor function;
7. optional provider/module registration helpers.

Pseudo-generated shape:

```go
func NewGojaLoader(opts ...gojapb.Option) require.ModuleLoader {
    cfg := gojapb.NewConfig("sessionstream.examples.chatdemo.v1", opts...)
    return func(vm *goja.Runtime, moduleObj *goja.Object) {
        exports := moduleObj.Get("exports").(*goja.Object)
        installEnums(vm, exports)
        installStartInferenceCommand(vm, exports, cfg)
        installUserMessageAcceptedEvent(vm, exports, cfg)
    }
}

func installStartInferenceCommand(vm *goja.Runtime, exports *goja.Object, cfg gojapb.Config) {
    messageObj := vm.NewObject()
    _ = messageObj.Set("typeName", "sessionstream.examples.chatdemo.v1.StartInferenceCommand")
    _ = messageObj.Set("builder", func(call goja.FunctionCall) goja.Value {
        return newStartInferenceCommandBuilder(vm, &StartInferenceCommand{})
    })
    _ = messageObj.Set("is", func(value goja.Value) bool {
        msg, ok := gojapb.FromValue(value)
        if !ok { return false }
        _, ok = msg.(*StartInferenceCommand)
        return ok
    })
    _ = exports.Set("StartInferenceCommand", messageObj)
}
```

## Field mapping rules

### Scalar fields

| Proto kind | JS setter input | Go behavior | DTS type |
|---|---|---|---|
| string | string | assign string | `string` |
| bool | boolean | assign bool | `boolean` |
| double/float | number | reject NaN/Inf unless option allows | `number` |
| int32/sint32/sfixed32 | number | require integer and range | `number` |
| uint32/fixed32 | number | require integer, range 0..max | `number` |
| int64/sint64/sfixed64 | string, bigint, safe integer number | parse/check int64 | `string | bigint | number` |
| uint64/fixed64 | string, bigint, safe integer number | parse/check uint64 | `string | bigint | number` |
| bytes | Uint8Array, ArrayBuffer, base64 string | copy bytes | `Uint8Array | ArrayBuffer | string` |
| enum | enum number or name string | validate against enum descriptor | `EnumType | keyof typeof EnumType` |

For 64-bit integers, accepting JS `number` is convenient but dangerous. The setter should reject non-safe integers:

```js
builder.sequenceId(9007199254740993); // error
builder.sequenceId("9007199254740993"); // ok
builder.sequenceId(9007199254740993n); // ok if BigInt supported by current Goja build
```

### Message fields

Message fields should accept:

1. a built `ProtoMessage` reference of the exact expected type;
2. a builder for the exact expected type, in which case it builds/clones;
3. for well-known wrapper ergonomics only, a supported native JS value such as `Date` for `Timestamp`.

Do not accept arbitrary plain objects by default in generated builder setters. If a `fromObject` escape hatch is added later, make it explicit:

```js
builder.metadata(MyMessage.fromObject({ x: 1 })); // explicit JSON-like path
```

### Repeated fields

Generate methods:

```ts
items(values: ItemInput[]): this;      // replace entire list
addItem(value: ItemInput): this;       // append one
clearItems(): this;
```

For scalar repeated fields:

```js
builder.tags(["a", "b"]);
builder.addTag("c");
```

For message repeated fields:

```js
builder.addTurn(turns.Turn.builder().role("user").content("hi").build());
```

### Map fields

Generate methods:

```ts
labels(values: Record<string, string> | Map<string, string>): this;
putLabel(key: string, value: string): this;
deleteLabel(key: string): this;
clearLabels(): this;
```

Map key types in protobuf are constrained. Support string, bool, int32/uint32/int64/uint64 key conversions according to protobuf rules. Avoid accepting arbitrary objects for non-string keys unless conversion is deterministic and documented.

### Oneof fields

Generate one setter per oneof member plus introspection and clear methods:

```ts
text(value: string): this;
toolCall(value: ToolCall): this;
whichContent(): "text" | "toolCall" | undefined;
clearContent(): this;
```

Calling one oneof setter must clear previous alternatives. This is naturally handled by `protoreflect.Message.Set` for a oneof field if the runtime helper uses descriptors correctly, or by direct assignment to generated oneof wrapper structs if optimized later.

### Optional fields

Generate presence helpers:

```ts
nickname(value: string): this;
hasNickname(): boolean;
clearNickname(): this;
```

Presence must use protobuf presence, not a zero-value comparison. For proto3 optional scalars, setting `""` is different from not setting the field.

### Enums

Export enums as frozen-ish JS objects:

```js
chat.Role = {
  ROLE_UNSPECIFIED: 0,
  ROLE_USER: 1,
  ROLE_ASSISTANT: 2,
};
```

DTS:

```ts
export enum Role {
  ROLE_UNSPECIFIED = 0,
  ROLE_USER = 1,
  ROLE_ASSISTANT = 2,
}
```

Setters should accept either numeric enum values or string names:

```js
builder.role(chat.Role.ROLE_USER);
builder.role("ROLE_USER");
```

Reject unknown enum values by default, with an option to allow open enum numbers if a consumer needs protobuf's forward-compat behavior.

## Well-known type policy

The first version should support common well-known types well enough to be usable:

| Type | Builder input | Output `toJSON()` |
|---|---|---|
| `google.protobuf.Timestamp` | `Date`, RFC3339 string, Timestamp message ref | RFC3339 string or protojson-compatible string |
| `google.protobuf.Duration` | string duration, `{seconds,nanos}`, Duration ref | protojson duration string |
| `google.protobuf.Any` | `ProtoMessage`, `{typeUrl, value}` explicit object | protojson Any |
| `google.protobuf.Struct` | plain JS object | plain JS object |
| `google.protobuf.Value` | JS primitive/object/array/null | JS value |
| `google.protobuf.ListValue` | JS array | JS array |
| wrapper types | primitive or wrapper message ref | primitive |
| `FieldMask` | string or string[] | protojson field mask string |

This is the one place where plain JS objects are acceptable because `Struct` and `Value` are explicitly open JSON-shaped protobuf types.

## TypeScript declaration generation

Generated DTS should be RawDTS-centered because builder interfaces, enum declarations, generic message refs, and namespace-like exports exceed the simple function model in `pkg/tsgen/spec`.

Example DTS:

```ts
type Int64Like = string | bigint | number;
type BytesLike = Uint8Array | ArrayBuffer | string;

interface ProtoMessage<TTypeName extends string = string> {
  readonly typeName: TTypeName;
  toJSON(): unknown;
  clone(): ProtoMessage<TTypeName>;
  equals(other: unknown): boolean;
}

interface StartInferenceCommand extends ProtoMessage<"sessionstream.examples.chatdemo.v1.StartInferenceCommand"> {}

interface StartInferenceCommandBuilder {
  prompt(value: string): this;
  clearPrompt(): this;
  build(): StartInferenceCommand;
  clone(): StartInferenceCommandBuilder;
}

export const StartInferenceCommand: {
  readonly typeName: "sessionstream.examples.chatdemo.v1.StartInferenceCommand";
  builder(): StartInferenceCommandBuilder;
  is(value: unknown): value is StartInferenceCommand;
  clone(value: StartInferenceCommand): StartInferenceCommand;
};
```

The generated Go function:

```go
func TypeScriptModule(moduleName string) *spec.Module {
    if moduleName == "" { moduleName = defaultModuleName }
    return &spec.Module{Name: moduleName, RawDTS: generatedRawDTS()}
}
```

This descriptor can be consumed directly by xgoja provider DTS generation.

## Integration with consuming modules

### The unwrapping contract

Add a stable unwrapping function in `pkg/protogoja`:

```go
func MessageFromValue(value goja.Value) (proto.Message, bool)
func MessageFromValueAs[T proto.Message](value goja.Value) (T, bool)
```

Consuming modules should use this before any JSON fallback:

```go
func decodePayload(value goja.Value, fallback func(goja.Value) (proto.Message, error)) (proto.Message, error) {
    if msg, ok := protogoja.MessageFromValue(value); ok {
        return proto.Clone(msg), nil
    }
    return fallback(value)
}
```

For future `sessionstream` bindings, this lets `hub.submit` and `publisher.publish` accept generated message refs:

```js
hub.submit("s1", "ChatStartInference", chat.StartInferenceCommand.builder().prompt("hi").build());
```

No JSON marshal/unmarshal is needed on this path.

### Schema token contract

Generated modules should also expose prototype/schema tokens so consuming schema registries can register expected types without asking callers for string full names:

```js
schemas.registerCommand("ChatStartInference", chat.StartInferenceCommand);
schemas.registerEvent("ChatUserMessageAccepted", chat.UserMessageAcceptedEvent);
```

Go side:

```go
func PrototypeFromValue(value goja.Value) (proto.Message, bool)
```

The message namespace object can carry a hidden zero-value prototype reference in addition to public `typeName`.

## Build and generation workflows

### Protoc workflow

```bash
go install github.com/go-go-golems/go-go-goja/cmd/protoc-gen-goja-builder@latest
protoc -I proto \
  --go_out=gen --go_opt=paths=source_relative \
  --goja-builder_out=gen --goja-builder_opt=paths=source_relative,module_name=my.pkg.v1 \
  proto/my/pkg/v1/*.proto
```

### Buf workflow

Buf users should use a local plugin initially:

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt:
      - paths=source_relative
  - local: protoc-gen-goja-builder
    out: gen
    opt:
      - paths=source_relative
      - module_name=my.pkg.v1
```

A remote Buf plugin can come later after the CLI stabilizes.

### go:generate workflow

For repo-local generation, mirror the existing hashiplugin generator pattern. Example:

```go
//go:generate protoc -I proto --go_out=gen --go_opt=paths=source_relative --goja-builder_out=gen --goja-builder_opt=paths=source_relative,module_name=sessionstream.examples.chatdemo.v1 proto/sessionstream/examples/chatdemo/v1/chat.proto
```

If a project wants pinned tool installation, create an internal generator command that installs both `protoc-gen-go` and `protoc-gen-goja-builder` to a temp `GOBIN`, similar to `pkg/hashiplugin/contract/internal/cmd/generate/main.go`.

## Implementation plan

### Milestone A: runtime support package

Files:

```text
pkg/protogoja/ref.go
pkg/protogoja/builder.go
pkg/protogoja/fields.go
pkg/protogoja/wellknown.go
pkg/protogoja/typescript.go
pkg/protogoja/ref_test.go
pkg/protogoja/fields_test.go
```

Build:

- `MessageRef` with hidden non-enumerable Go reference.
- `MessageFromValue` and `PrototypeFromValue` extraction.
- `ToValue(vm, msg)` wrapping with `typeName`, `toJSON`, `clone`, and `equals`.
- `BuilderRef` with descriptor-based `Set`, `Add`, `Put`, `Clear`, `Build`, and `Clone`.
- Conversion for scalar, enum, message ref, repeated, map, oneof, optional, and common WKTs.

Acceptance tests:

- Built message unwraps as the same concrete Go type.
- `build()` returns a clone, not mutable builder state.
- invalid field values return field-specific errors.
- uint64/int64 reject unsafe numbers and accept strings/bigints.
- Struct/Value accept plain JS objects only for those WKTs.

### Milestone B: protoc plugin skeleton

Files:

```text
cmd/protoc-gen-goja-builder/main.go
pkg/protogoja/genconfig/config.go
pkg/protogoja/generator/*.go  # optional internal package if main becomes large
```

Build:

- Parse plugin parameters.
- Iterate `plugin.Files` and skip non-generated files.
- Generate one companion Go file per proto file.
- Derive module names from proto package unless overridden.
- Emit loader with message exports and enum exports.
- Emit TypeScript descriptor function.

Acceptance tests:

- Golden test for a tiny proto file.
- Generated code compiles with `go test`.
- Generated module can be required in a Goja runtime.

### Milestone C: generated fluent builders for phase-2 first version

This is the first real user-facing version. It must include generated builder methods, not only generic `.set()`.

Build support for:

- scalars;
- enums;
- nested messages;
- repeated fields;
- maps;
- oneofs;
- optional fields;
- common well-known types;
- TypeScript declarations for all generated message builders;
- schema/prototype tokens for consuming modules.

Generated method naming:

- Use protojson camelCase by default: `message_id` → `messageId`.
- Generate `clear<Field>()`, `add<Singular>()`, `put<Singular>()`, `delete<Singular>()`, `has<Field>()`, and `which<Oneof>()` as appropriate.
- Avoid generating both snake_case and camelCase in version 1; add aliases later if needed.

Acceptance tests:

- Use a fixture proto that includes every supported field kind.
- In JS, build messages through fluent methods and unwrap them in Go.
- Compare against manually constructed Go messages with `proto.Equal`.
- Render DTS and check key interface/method lines.

### Milestone D: module/provider integration helpers

Build:

- `GojaModule(opts...) modules.NativeModule`.
- `RegisterGojaModule(reg, opts...)`.
- Optional `ProviderModule(packageID, moduleName string)` helper if importing `providerapi` is acceptable.
- Examples for raw require, engine runtime modules, and xgoja provider integration.

Acceptance tests:

- A generated builder module registers through raw `require.Registry`.
- A generated builder module registers through `engine.NativeModuleRegistrar`.
- TypeScript descriptor is accepted by `validate.Module` and `dtsgen`.

### Milestone E: consuming-module demonstration

Use sessionstream or a small local fake consumer to prove the integration value:

```go
if msg, ok := protogoja.MessageFromValue(value); ok {
    // no JSON/protojson conversion
    return proto.Clone(msg), nil
}
```

Test from JS:

```js
const pb = require("fixture.v1");
consumer.accept(pb.Example.builder().name("x").build());
```

Acceptance test:

- Consumer receives concrete `*fixturev1.Example`.
- Plain object fallback still works if the consumer supports it.
- Type mismatch errors are clear.

## Testing strategy

### Unit tests for runtime conversion

Create table-driven tests for each field kind. Each test should build a JS value in a runtime, call the generated/builder helper, unwrap the proto, and compare with expected Go message.

### Golden tests for generated code

Use a small test harness that feeds a `CodeGeneratorRequest` to the plugin and compares generated output to golden files. Keep golden files stable by sorting message/enum output and avoiding timestamps.

### Compile tests for generated code

Generate into a temporary fixture package and run:

```bash
go test ./...
```

This catches missing imports, bad oneof wrapper names, and package naming errors that golden text tests can miss.

### Goja runtime tests

For each fixture package:

```go
vm := goja.New()
reg := require.NewRegistry()
fixturev1.RegisterGojaModule(reg)
reg.Enable(vm)
_, err := vm.RunString(`
  const pb = require("fixture.v1");
  const msg = pb.Example.builder().name("abc").count(3).build();
`)
```

Then retrieve and unwrap `msg` with `protogoja.MessageFromValue`.

### TypeScript rendering tests

Feed generated `TypeScriptModule` into `tsgen/render` or `xgoja/dtsgen`. Assert that output contains:

- `declare module "fixture.v1"`;
- `interface ExampleBuilder`;
- `build(): Example`;
- enum declarations;
- oneof methods;
- repeated/map helper methods.

### Compatibility tests

Use both fixture schemas:

- `go-go-goja/pkg/hashiplugin/contract/jsmodule.proto` for in-repo protobuf coverage.
- `sessionstream/examples/chatdemo/proto/.../chat.proto` as an external-workspace acceptance fixture, if tests can refer to the workspace module without creating a hard dependency. If not, copy a small fixture into `pkg/protogoja/testdata`.

## Error model

Builder setter errors should be precise:

```text
Example.sequenceId: expected int64 as string, bigint, or safe integer number; got number 9007199254740993 outside safe integer range
Example.labels[key]: expected string key; got object
Example.status: unknown enum name "DONEE" for fixture.v1.Status
Example.payload: expected fixture.v1.Payload ProtoMessage; got fixture.v1.OtherPayload
```

In JavaScript, generated setters should throw Goja `GoError` values. This matches existing Go-backed module patterns.

## Performance considerations

The primary performance win is avoiding JSON/protojson round-trips. Even if setters use protoreflect internally, the path is still:

```text
JS setter input → Go conversion → proto.Message field set → hidden MessageRef
```

rather than:

```text
JS object → JSON.stringify/export → protojson.Unmarshal → proto.Message
```

Later optimization can generate direct assignment for hot fields:

```go
b.msg.Prompt = value.String()
```

But this should be deferred until correctness and coverage are strong.

## Security and safety notes

- Hidden Go references are not a security boundary. They are an embedding convention.
- Generated builder modules should assume JavaScript can call methods in any order and with wrong types.
- Built messages returned to JS should be immutable from JavaScript's perspective. Mutation should require `MessageType.from(msg)` or `msg.toBuilder()`.
- Always clone when passing messages across API boundaries unless ownership is clearly documented.
- Do not let JS mutate the same Go message concurrently with Go consumers.

## Decision records

### Decision: first public version is generated fluent builders, not reflection-only

- **Context:** The user explicitly wants at least phase 2 for the first version. A reflection-only builder proves mechanics but does not deliver the desired ergonomics.
- **Options considered:** Reflection-only runtime builder; generated fluent builders; generated direct Go structs with no runtime helpers.
- **Decision:** Build generated fluent builders as the first user-facing version, backed by shared runtime conversion helpers.
- **Rationale:** This gives autocomplete, clear methods, and reusable modules while keeping tricky conversion logic centralized.
- **Consequences:** The initial implementation is larger than a prototype, but it avoids designing an API that will be immediately replaced.
- **Status:** proposed

### Decision: generate companion Go files in the protobuf Go package

- **Context:** Builders need to instantiate concrete generated Go message types.
- **Options considered:** Generate separate package importing the pb package; generate into the same Go package; generate a dynamic descriptor-only module.
- **Decision:** Generate companion files into the same Go package as `protoc-gen-go` output.
- **Rationale:** This matches common protobuf plugin behavior and avoids awkward import/path configuration for concrete message construction.
- **Consequences:** The plugin must follow `go_package` and `paths` conventions carefully. Separate package generation can be added later if needed.
- **Status:** proposed

### Decision: built message values carry hidden proto refs

- **Context:** Consuming Goja modules need to recover concrete `proto.Message` values without JSON.
- **Options considered:** Return plain objects; return Go struct wrappers directly; attach hidden refs to JS objects; store refs in a runtime side table.
- **Decision:** Return JS objects with hidden non-enumerable Go refs, plus `protogoja.MessageFromValue` extraction.
- **Rationale:** This matches patterns already used by local Goja bindings and keeps a stable object API visible to JS.
- **Consequences:** This is not a security boundary. Careful cloning is required at API boundaries.
- **Status:** proposed

### Decision: TypeScript output uses RawDTS

- **Context:** Builder declarations need classes/interfaces/enums/generics and namespace-like exports.
- **Options considered:** Extend `spec.TypeRef`; use RawDTS; emit standalone `.d.ts` files only.
- **Decision:** Generate `spec.Module{RawDTS: ...}` in Go code.
- **Rationale:** Existing go-go-goja rendering and xgoja DTS bundling already support RawDTS and module alias rewriting.
- **Consequences:** RawDTS generation must be tested carefully because it bypasses typed AST validation for declarations.
- **Status:** proposed

### Decision: plain object input is only allowed for explicit JSON-shaped protobuf types

- **Context:** The purpose is to avoid JSON object marshalling for typed schemas.
- **Options considered:** Accept arbitrary plain objects for any message field; reject all plain objects; allow plain objects only for Struct/Value and explicit fromObject helpers.
- **Decision:** Allow plain objects only for protobuf types whose semantics are JSON-shaped, such as `Struct` and `Value`, plus explicit `fromObject` escape hatches later.
- **Rationale:** This preserves protobuf type boundaries and prevents the builder API from becoming a hidden protojson path.
- **Consequences:** Nested message values require builders or message refs, which is stricter but clearer.
- **Status:** proposed

## Risks and open questions

### Oneof code generation details

The protoreflect helper approach should handle oneofs, but direct assignment optimization must know generated wrapper type names. Keep direct assignment out of version 1 unless necessary.

### Proto2 support

The design should not intentionally break proto2, but first tests should focus on proto3. Proto2 required fields and presence semantics need dedicated tests before claiming full support.

### Goja BigInt behavior

Goja supports modern JavaScript features in current versions, but tests must verify BigInt behavior in the pinned dependency. If BigInt interop is awkward, string input remains the stable 64-bit path.

### Module naming defaults

Proto package names often work as require names, but some hosts may prefer shorter aliases. The plugin should allow `module_name` override and xgoja aliasing should still work.

### Provider helper imports

Generated code that imports xgoja `providerapi` may be too heavy for all protobuf packages. Keep provider helpers optional or in a separate generated file controlled by `emit_provider=true`.

## Suggested first PR sequence

1. Add `pkg/protogoja` runtime support with tests.
2. Add `cmd/protoc-gen-goja-builder` skeleton and golden tests.
3. Generate fluent builders for scalar, enum, message, repeated, map, optional, oneof, and common WKT fields.
4. Generate TypeScript RawDTS and test with `tsgen/render`/`xgoja/dtsgen`.
5. Add fixture generation and Goja runtime acceptance tests.
6. Add documentation and examples showing integration with a consuming module.

## Acceptance criteria for the first usable release

The feature is ready for first use when this script works in a test:

```js
const pb = require("fixture.v1");

const nested = pb.Nested.builder()
  .label("child")
  .build();

const msg = pb.Example.builder()
  .name("demo")
  .sequenceId("9007199254740993")
  .enabled(true)
  .status(pb.Status.STATUS_READY)
  .nested(nested)
  .addTag("alpha")
  .putLabel("owner", "goja")
  .text("oneof text")
  .build();

consumer.accept(msg);
```

And Go verifies:

```go
unwrapped, ok := protogoja.MessageFromValue(value)
require.True(t, ok)
require.IsType(t, &fixturev1.Example{}, unwrapped)
require.True(t, proto.Equal(expected, unwrapped))
```

No JSON/protojson conversion should be used on that path.

## Final recommendation

Build the full reusable feature in `go-go-goja`, not in `sessionstream`. The first public version should be a generated fluent builder system backed by a small runtime helper package. Use protoreflect internally for correctness, emit rich TypeScript declarations through existing RawDTS support, and provide a stable `protogoja.MessageFromValue` unwrapping contract so any Goja-consuming module can accept generated protobuf values.

For `sessionstream`, this later becomes an optimization and ergonomics layer: schema bindings can accept generated message type tokens, command/event submissions can accept built message refs, and JSON/protojson becomes a fallback rather than the primary path.

## Phase 6 generated TypeScript declaration examples

The first generated DTS slice exposes the runtime objects produced by Phase 5 through `spec.Module{RawDTS: ...}`. The generator emits a file-local function such as:

```go
func GojaBuilderFileJsmoduleProtoTypeScriptModule(moduleName string) *spec.Module
```

Hosts can pass the returned descriptor through `pkg/tsgen/render` directly, or register it on an xgoja provider module so `pkg/xgoja/dtsgen` can bundle it with other selected runtime modules.

A generated module declaration has this shape:

```ts
declare module "hashiplugin.contract.v1" {
  export interface ProtoMessage<TTypeName extends string = string> {
    readonly typeName: TTypeName;
    toJSON(): unknown;
    clone(): ProtoMessage<TTypeName>;
    equals(other: unknown): boolean;
  }

  export interface ModuleManifest
    extends ProtoMessage<"hashiplugin.contract.v1.ModuleManifest"> {}

  export interface ModuleManifestBuilder {
    moduleName(value: string): this;
    clearModuleName(): this;
    version(value: string): this;
    clearVersion(): this;
    exports(value: ExportSpec[]): this;
    clearExports(): this;
    capabilities(value: string[]): this;
    clearCapabilities(): this;
    doc(value: string): this;
    clearDoc(): this;
    build(): ModuleManifest;
    clone(): ModuleManifestBuilder;
  }

  export const ModuleManifest: MessageNamespace<
    ModuleManifest,
    ModuleManifestBuilder
  >;

  export const ExportKind: {
    readonly typeName: "hashiplugin.contract.v1.ExportKind";
    readonly EXPORT_KIND_UNSPECIFIED: 0;
    readonly EXPORT_KIND_FUNCTION: 1;
    readonly EXPORT_KIND_OBJECT: 2;
  };

  export type ExportKindValue = 0 | 1 | 2;
}
```

A JavaScript author then gets a typed fluent construction path:

```ts
import { ModuleManifest, ExportKind, ExportSpec } from "hashiplugin.contract.v1";

const exportSpec = ExportSpec.builder()
  .name("run")
  .kind(ExportKind.EXPORT_KIND_FUNCTION)
  .build();

const manifest = ModuleManifest.builder()
  .moduleName("demo")
  .version("v1")
  .exports([exportSpec])
  .capabilities(["tools"])
  .build();
```

The generated declarations deliberately model the currently implemented runtime surface. They include `ProtoMessage`, message namespace objects, message interfaces, builder interfaces, enum export objects, enum value unions, repeated fields as arrays, and maps as `Record<string, T>`. Oneof-specific declarations are still deferred until oneof runtime helpers are implemented.
