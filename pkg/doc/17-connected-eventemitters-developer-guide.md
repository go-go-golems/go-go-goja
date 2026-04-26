---
Title: Connected EventEmitter Developer Guide
Slug: connected-eventemitters-developer-guide
Short: Build opt-in Go resource helpers that deliver events to JavaScript-created EventEmitter instances safely.
Topics:
- goja
- javascript
- event-emitter
- async
- fswatch
- watermill
Commands:
- goja-repl
- jsverbs-example
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This guide explains how to build and use connected EventEmitter helpers in go-go-goja. A connected helper lets JavaScript create a normal `EventEmitter`, pass it to a Go-backed function, and receive events from a Go resource such as fsnotify or Watermill without violating goja's single-owner runtime rule.

Use this pattern when the JavaScript side should own listener registration, but the Go side owns a long-lived resource that produces events over time. Examples include filesystem watchers, message-bus subscriptions, process supervisors, or domain-specific notification streams.

## Why connected emitters exist

Go resources often emit data from background goroutines. goja runtimes are not goroutine-safe: any code that touches `goja.Runtime`, `goja.Value`, a JavaScript function, or a JavaScript object must run on the runtime owner thread. Calling JavaScript directly from a filesystem watcher goroutine or message subscription goroutine creates data races and intermittent crashes.

Connected emitters solve that boundary by separating ownership:

- JavaScript creates and owns the `EventEmitter` object.
- Go adopts that emitter on the owner thread and stores a safe `EmitterRef`.
- Background goroutines only call `EmitterRef.Emit(...)` or `EmitterRef.EmitWithBuilder(...)`.
- `EmitterRef` schedules listener delivery back onto the runtime owner.
- The returned connection object closes the Go resource without removing JavaScript listeners.

The result feels natural in JavaScript while keeping Go concurrency explicit and reviewable.

## JavaScript shape

A connected helper always starts with a JavaScript-created emitter:

```javascript
const EventEmitter = require("events");

const watcher = new EventEmitter();
const conn = fswatch.watch("/tmp/project", watcher, {
  recursive: true,
  debounceMs: 100,
  include: ["**/*.js", "**/*.ts"],
  exclude: ["**/node_modules/**", "**/.git/**"]
});

watcher.on("event", (ev) => {
  console.log(ev.relativeName, ev.op, ev.debounced, ev.count);
});

watcher.on("error", (err) => {
  console.error(err.path, err.message);
});

conn.close();
```

The helper is intentionally not a default global in every runtime. Host applications install it explicitly because filesystem watching and message subscriptions are host-resource access.

## Embedding runtime setup

Install the connected-emitter manager first, then install resource helpers. The order matters because helpers look up the manager during runtime initialization.

```go
package myruntime

import (
    "context"
    "time"

    "github.com/go-go-golems/go-go-goja/engine"
    "github.com/go-go-golems/go-go-goja/pkg/jsevents"
)

func NewRuntime(ctx context.Context) (*engine.Runtime, error) {
    factory, err := engine.NewBuilder().
        WithModules(engine.DefaultRegistryModules()).
        WithRuntimeInitializers(
            jsevents.Install(),
            jsevents.FSWatchHelper(jsevents.FSWatchOptions{
                Root:           "/tmp/my-app-sandbox",
                AllowRecursive: true,
                MaxDebounce:    time.Second,
            }),
        ).
        Build()
    if err != nil {
        return nil, err
    }
    return factory.NewRuntime(ctx)
}
```

`jsevents.Install()` does not create any emitters or start any background work. It only stores a per-runtime manager that helpers use later when JavaScript calls helper functions.

## EventEmitter module contract

The `events` and `node:events` modules are data-only defaults. go-go-goja uses Node's `node:` prefix for Node-compatible or mostly-compatible built-ins such as `node:events`, `node:path`, `node:crypto`, and opt-in host modules such as `node:fs` and `node:os`. Custom helpers such as `fswatch`, `watermill`, `time`, and `timer` intentionally do not use a `node:` prefix.

The EventEmitter module is available in fresh runtimes and implements a Go-native subset of Node's EventEmitter:

```javascript
const EventEmitter = require("events");
const emitter = new EventEmitter();

emitter.once("ready", (name) => console.log("first", name));
emitter.on("ready", (name) => console.log("always", name));

emitter.emit("ready", "goja");
emitter.emit("ready", "again");
```

Go helpers validate JavaScript-created emitters with `events.FromValue(...)` indirectly through `Manager.AdoptEmitterOnOwner(...)`. Do not accept arbitrary JavaScript objects and try to call `.emit` yourself from Go; that bypasses the native emitter adoption and owner-thread scheduling model.

## Writing a connected helper

A helper function should do only setup while it is executing on the owner thread. Long-running work belongs in a goroutine that communicates through `EmitterRef`.

The typical shape is:

```go
func MyHelper(opts MyOptions) engine.RuntimeInitializer {
    return &myHelper{opts: opts}
}

func (h *myHelper) InitRuntime(ctx *engine.RuntimeContext) error {
    managerValue, ok := ctx.Value(jsevents.RuntimeValueKey)
    if !ok {
        return fmt.Errorf("my helper: manager is not installed; add jsevents.Install() first")
    }
    manager := managerValue.(*jsevents.Manager)

    obj := ctx.VM.NewObject()
    if err := obj.Set("connect", func(call goja.FunctionCall) goja.Value {
        resourceName := call.Argument(0).String()
        ref, err := manager.AdoptEmitterOnOwner(call.Argument(1))
        if err != nil {
            panic(ctx.VM.NewGoError(err))
        }

        resourceCtx, cancel := context.WithCancel(ctx.Context)
        ref.SetCancel(cancel)

        go runResource(resourceCtx, ref, resourceName)

        return connectionObject(ctx.VM, ref)
    }); err != nil {
        return err
    }

    return ctx.VM.Set("myResource", obj)
}
```

The resource goroutine never receives a `*goja.Runtime`, `goja.Value`, or JavaScript callback. It receives only plain Go data and an `EmitterRef`.

## Building typed payloads

Use typed Go structs for JavaScript-facing payloads, then build lowerCamel JavaScript objects explicitly. Do not pass free-form `map[string]any` payloads for helper events; maps make contracts drift and hide field spelling changes from review.

```go
type FileWatchEvent struct {
    Source       string
    WatchPath    string
    Name         string
    RelativeName string
    Op           string
    Create       bool
    Write        bool
    Debounced    bool
    Count        int
}

func (p FileWatchEvent) ToValue(vm *goja.Runtime) goja.Value {
    obj := vm.NewObject()
    _ = obj.Set("source", p.Source)
    _ = obj.Set("watchPath", p.WatchPath)
    _ = obj.Set("name", p.Name)
    _ = obj.Set("relativeName", p.RelativeName)
    _ = obj.Set("op", p.Op)
    _ = obj.Set("create", p.Create)
    _ = obj.Set("write", p.Write)
    _ = obj.Set("debounced", p.Debounced)
    _ = obj.Set("count", p.Count)
    return obj
}
```

Then emit through an owner-thread value builder:

```go
payload := FileWatchEvent{Source: "fsnotify", Name: event.Name, Count: 1}
_ = ref.EmitWithBuilder(ctx, "event", func(vm *goja.Runtime) ([]goja.Value, error) {
    return []goja.Value{payload.ToValue(vm)}, nil
})
```

This pattern keeps the Go side typed while giving JavaScript idiomatic lowerCamel properties.

## fswatch helper

`FSWatchHelper` installs a custom `fswatch` global. It is not a Node standard library. It is a go-go-goja host helper for connecting `github.com/fsnotify/fsnotify` to a JavaScript-created EventEmitter.

JavaScript API:

```typescript
interface FSWatchOptionsJS {
  recursive?: boolean;
  debounceMs?: number;
  include?: string[];
  exclude?: string[];
}

interface FSWatchConnection {
  id: string;
  path: string;
  recursive: boolean;
  debounceMs: number;
  include: string[];
  exclude: string[];
  close(): boolean;
}
```

Event payload:

```typescript
interface FileWatchEvent {
  source: "fsnotify";
  watchPath: string;
  name: string;
  relativeName: string;
  op: string;
  create: boolean;
  write: boolean;
  remove: boolean;
  rename: boolean;
  chmod: boolean;
  recursive: boolean;
  debounced: boolean;
  count: number;
}
```

Host options:

```go
type FSWatchOptions struct {
    GlobalName     string
    Root           string
    AllowPath      func(path string) bool
    AllowRecursive bool
    MaxDebounce    time.Duration
    IgnorePath     func(path string) bool
}
```

Important behavior:

- `Root` and `AllowPath` constrain host path access.
- `AllowRecursive` defaults to false because recursive watching can allocate one OS watch per directory.
- `MaxDebounce` caps script-provided `debounceMs`.
- `IgnorePath` lets hosts skip paths before traversal or event delivery.
- Recursive traversal skips symlink directories.
- Include and exclude globs match slash-separated paths relative to the watch root.
- `**` as a full path segment matches zero or more path segments.

## Watermill helper

`WatermillHelper` follows the same adoption pattern for message subscriptions. JavaScript passes an emitter into `watermill.connect(topic, emitter)`, and Go emits `message`, `error`, and `close` events on that emitter.

A message event receives a typed JavaScript object with settlement methods:

```javascript
orders.on("message", (msg) => {
  console.log(msg.uuid, msg.payload, msg.metadata.source);
  msg.ack();
});
```

Use the Watermill helper only when the embedding application has an explicit `message.Subscriber` and topic policy. Do not install default subscriptions at runtime startup.

## jsverbs examples

The fixture directory contains examples for the EventEmitter and fswatch APIs:

```text
testdata/jsverbs/events.js
testdata/jsverbs/fswatch.js
```

EventEmitter examples run through the default `jsverbs-example` runtime:

```bash
go run ./cmd/jsverbs-example --dir testdata/jsverbs events event-timeline evt --count 2
go run ./cmd/jsverbs-example --dir testdata/jsverbs events listener-summary demo
go run ./cmd/jsverbs-example --dir testdata/jsverbs events handled-error boom
```

The fswatch example demonstrates the JavaScript shape but requires an embedding runtime that installs `jsevents.Install()` and `FSWatchHelper(...)`. The regression test `TestFSWatchJsverbUsesInstalledHelper` shows that embedding path and invokes the jsverb with recursive watching, debouncing, and glob filters.

```bash
go test ./pkg/jsverbs -run TestFSWatchJsverbUsesInstalledHelper -count=1
```

## Review checklist

Use this checklist when reviewing a new connected helper:

- The helper requires explicit runtime installation.
- `jsevents.Install()` runs before the helper initializer.
- JavaScript creates the EventEmitter and passes it into Go.
- Go adopts the emitter on the owner thread.
- Background goroutines do not touch goja runtime values directly.
- Event, error, and connection payloads are typed Go structs with explicit JS builders.
- The connection `close()` is idempotent and cancels Go resources.
- Host access has policy hooks such as `Root`, `AllowPath`, `AllowTopic`, or equivalent.
- Tests cover invalid emitters, denied resources, listener delivery, no-listener behavior where relevant, and close cleanup.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `fswatch is not defined` | The runtime did not install `FSWatchHelper` | Add `jsevents.Install()` and `jsevents.FSWatchHelper(...)` to `WithRuntimeInitializers(...)`. |
| `manager is not installed` | Helper initializer ran before `jsevents.Install()` | Put `jsevents.Install()` before helper initializers. |
| Recursive watch request fails | Host did not set `AllowRecursive: true` | Enable `AllowRecursive` only for trusted/sandboxed roots. |
| Debounce request fails | `debounceMs` is negative, non-finite, or above `MaxDebounce` | Validate script options or raise `MaxDebounce` intentionally. |
| No event for the first file in a newly-created directory | Recursive directory registration happens after fsnotify reports directory creation | Wait for directory registration, write again, or add a future `directory-added`/`ready` event if the script needs a guarantee. |
| Listener throws but the goroutine keeps running | Async listener errors are reported through the manager error handler | Install `jsevents.WithErrorHandler(...)` and decide whether the host should close the resource on listener errors. |
| Go code wants to pass `map[string]any` as event payload | The helper contract becomes hard to review and document | Define a struct and a `ToValue(vm)` method instead. |

## See Also

- `glaze help nodejs-primitives` for built-in module availability and EventEmitter reference.
- `glaze help async-patterns` for owner-thread scheduling and Promise/callback patterns.
- `glaze help creating-modules` for native module adapter structure.
- `glaze help jsverbs-example-developer-guide` for embedding and invoking JavaScript-backed Glazed commands.
- `glaze help jsverbs-example-reference` for fixture and runner details.
