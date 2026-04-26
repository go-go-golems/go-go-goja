---
Title: Event emitter module implementation guide
Ticket: EVT-001
Status: active
Topics:
    - goja
    - javascript
    - event-emitter
    - module
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../goja_nodejs/eventloop/eventloop.go
      Note: Local RunOnLoop contract used as concurrency evidence.
    - Path: engine/factory.go
      Note: Runtime construction and owner/event-loop setup for the proposed bus.
    - Path: engine/module_specs.go
      Note: Default data-only module list and module registration APIs.
    - Path: engine/runtime.go
      Note: Module blank imports and runtime lifecycle close behavior.
    - Path: modules/common.go
      Note: NativeModule registry interface for the events module.
    - Path: modules/timer/timer.go
      Note: Existing async owner-thread scheduling pattern.
    - Path: pkg/runtimeowner/runner.go
      Note: Runner Call/Post scheduling API used by the proposed bus.
ExternalSources:
    - local:01-event-emitter.md
    - local:evidence.txt
Summary: Detailed analysis, design, and implementation guide for adding an event-emitter module to go-go-goja.
LastUpdated: 2026-04-26T10:02:00-04:00
WhatFor: Guide a new engineer through implementing a Go-owned event bus with a JavaScript EventEmitter façade in go-go-goja.
WhenToUse: Use before implementing require("events"), connected emitter factories, Watermill/fsnotify helpers, or any Go-to-JavaScript event dispatch path.
---


# Event emitter module implementation guide

## Executive summary

This ticket should implement an **event-emitter module** for `go-go-goja` in two layers:

1. A **Go-native JavaScript-compatible `events` module** exposed through the existing native-module registry so scripts can write `const EventEmitter = require("events")` and `const { EventEmitter } = require("node:events")`. The EventEmitter implementation itself must be Go code, not embedded JavaScript source.
2. A **connected-emitter helper pattern** where JavaScript can either receive a Go-created emitter from a Go-provided factory or pass a JS-created Go-native emitter into a Go function. Go can then register Go callbacks on it or safely emit events to it from Go-side resources. Watermill is one helper implementation of that general pattern, not a default runtime bus and not a preconfigured global emitter.

The critical correctness rule is:

> **No goroutine that is not the goja owner/event-loop goroutine may call JavaScript functions, touch `goja.Value`s, or use `*goja.Runtime` directly.**

The current repository already has the pieces needed to follow that rule. `engine.Factory.NewRuntime` creates a `goja.Runtime`, starts a `goja_nodejs/eventloop.EventLoop`, wraps it in a `runtimeowner.Runner`, and stores those bindings for modules through `runtimebridge` (`engine/factory.go:164-189`). The `runtimeowner.Runner` is the repository-preferred API for scheduling owner-thread work (`pkg/runtimeowner/types.go:20-25`, `pkg/runtimeowner/runner.go:62-160`). The newly added workspace checkout of `goja_nodejs` confirms that `EventLoop.RunOnLoop()` preserves ordering, passes a runtime value only for the scheduled function, is safe from inside or outside the loop, and returns `false` after termination (`../goja_nodejs/eventloop/eventloop.go:314-321`).

The recommended implementation is therefore **not** a standalone `Engine` wrapper that creates its own event loop, and it is also **not** a runtime initializer that creates default Watermill emitters. Instead, implement the feature using the existing `engine.Runtime` lifecycle:

- `modules/events`: register the Node-style `EventEmitter` constructor under `events` and `node:events`, implemented as a Go native constructor with Go-backed prototype methods.
- `pkg/jsevents`: provide reusable connected-emitter primitives for Go factory functions that create, register, emit to, and close specific JavaScript emitter instances safely.
- Watermill/fsnotify helpers: expose explicit Go-backed JavaScript factory functions such as `watermill.subscribe(...)` or `watch(path)` that return configured emitters. The application decides which helpers are installed and what Go resources they can access.

The imported source brief (`sources/local/01-event-emitter.md`) is directionally correct: it identifies the right ownership model, says to schedule every JavaScript call through the event loop, treats Node `emit()` as synchronous listener dispatch, and models Watermill settlement with explicit `ack()` / `nack()` methods. This guide adapts that brief to the actual `go-go-goja` architecture.



## Design revision: EventEmitter itself is Go-native

The EventEmitter implementation should be written in Go. There should be no embedded JavaScript source string that implements `on`, `once`, `emit`, or listener storage. JavaScript still uses the API, and JavaScript listener functions are still invoked, but the EventEmitter object, listener registry, and methods are native Go-backed goja objects.

This means the `events` module should create a native constructor using goja's constructor support:

```go
constructor := vm.ToValue(func(call goja.ConstructorCall) *goja.Object {
    emitter := events.NewEmitter(vm)       // Go struct holding listener state
    obj := vm.ToValue(emitter).(*goja.Object)
    obj.SetPrototype(call.This.Prototype())
    return obj
}).(*goja.Object)
```

The constructor's prototype should be populated with Go functions:

```go
proto.Set("on", vm.ToValue(func(call goja.FunctionCall) goja.Value {
    em := mustEmitter(vm, call.This)
    eventName := eventNameFromValue(call.Argument(0))
    listener := requireFunction(vm, call.Argument(1))
    em.On(eventName, listener)
    return call.This
}))

proto.Set("emit", vm.ToValue(func(call goja.FunctionCall) goja.Value {
    em := mustEmitter(vm, call.This)
    delivered, err := em.EmitFromCall(call)
    if err != nil {
        panic(err)
    }
    return vm.ToValue(delivered)
}))
```

The backing Go type should be discoverable from values passed back into Go. That is what makes JS-created emitters useful as handles:

```js
const EventEmitter = require("events");
const emitter = new EventEmitter();

watermill.connect("orders", emitter);

emitter.on("message", (msg) => {
  msg.ack();
});
```

When `watermill.connect(topic, emitter)` runs, it is already executing on the owner thread. The Go function can validate and unwrap the emitter:

```go
func connect(call goja.FunctionCall) goja.Value {
    topic := call.Argument(0).String()
    em, obj, ok := events.FromValue(call.Argument(1))
    if !ok {
        panic(vm.NewTypeError("second argument must be an events.EventEmitter"))
    }

    ref := manager.AdoptNativeEmitterOnOwner(em, obj)
    go runWatermillSubscription(ctx, ref, subscriber, topic)
    return vm.ToValue(ref.JSConnectionObject(vm)) // optional, or return undefined
}
```

This supersedes the earlier idea of implementing EventEmitter behavior in JavaScript. The connected-emitter manager can still exist, but its job is to hold safe references to **Go-native EventEmitter instances** and schedule owner-thread calls to their Go `Emit` method.

### Consequences of Go-native EventEmitter

- `modules/events` must not call `vm.RunString(eventEmitterSource)` for the EventEmitter implementation.
- The listener map lives in a Go struct, not in a JS `_events` object.
- Prototype methods (`on`, `once`, `off`, `emit`, `listeners`, `listenerCount`, etc.) are Go closures.
- A Go function that receives a JS-created emitter can unwrap/validate it with `events.FromValue(...)`.
- Background goroutines still cannot use the emitter directly. They must hold an `EmitterRef` and schedule owner-thread emission.
- Go code may register Go callbacks as listeners by converting a Go function to a goja function on the owner thread and adding it to the emitter's listener list.

### Revised connected-emitter shapes

Both of these are valid and should be supported:

```js
// Go creates and returns the emitter.
const watcher = fswatch.watch("/tmp/demo");
watcher.on("event", ev => console.log(ev));
```

```js
// JS creates the emitter; Go adopts/connects it.
const EventEmitter = require("events");
const orders = new EventEmitter();
watermill.connect("orders", orders);
orders.on("message", msg => msg.ack());
```

The second form is the important new requirement. It requires the Go-native EventEmitter module to expose an internal Go API for adoption, not just a JavaScript-compatible constructor.

## Design revision: connected emitters instead of default source buses

After reviewing the intended API shape, the design should use **connected emitters** as the primary abstraction:

```js
const orders = watermill.subscribe("orders");

orders.on("message", (msg) => {
  try {
    const order = JSON.parse(msg.payload);
    console.log(order.id);
    msg.ack();
  } catch (err) {
    msg.nack();
  }
});

orders.on("error", (err) => console.error(err.message || err));
```

In this model, JavaScript does not listen on a preconfigured `globalThis.goEvents` bus for `"watermill:orders"`. Instead:

1. The Go application explicitly installs a helper module or helper global for a resource family.
2. JavaScript calls a Go-backed factory function.
3. The Go function creates a fresh JS `EventEmitter` instance on the owner thread.
4. The Go function connects that emitter to the requested Go-side resource, such as a Watermill subscription.
5. The function returns the emitter to JavaScript.
6. Background goroutines deliver events only by scheduling owner-thread callbacks that look up and emit to that specific emitter.

This changes the earlier bus-oriented design in an important way: **Watermill emitters should never be configured by default**. Watermill support should be helper code for applications that already have a Watermill subscriber and intentionally expose a specific subscription API to JavaScript.

The safety invariant does not change. The connected-emitter implementation still must not call JavaScript from Watermill/fsnotify goroutines. Those goroutines only schedule owner-thread work through `runtimeowner.Runner` or the event loop.

## What an intern needs to know first

### What is goja?

`goja` is a JavaScript engine implemented in Go. A `*goja.Runtime` is roughly analogous to one JavaScript VM instance. It owns JavaScript objects, functions, promises, globals, and module state.

The important operational fact is that `goja.Runtime` is **not goroutine-safe**. If one goroutine is running JavaScript while another goroutine calls a JS function or converts a Go value into a `goja.Value`, the runtime can race or corrupt state. The repository avoids this by treating each runtime as an owned resource and scheduling all JS work onto one owner path.

### What is `goja_nodejs/eventloop`?

`goja_nodejs/eventloop` is the Node-like event-loop companion used by goja. In this workspace, `../goja_nodejs/eventloop/eventloop.go:314-321` documents the exact contract this design relies on:

- `RunOnLoop(fn)` schedules `fn` to run in the context of the loop.
- Scheduled function order matches `RunOnLoop()` call order.
- The `*goja.Runtime` and values derived from it must not be used outside `fn`.
- It is safe to call `RunOnLoop()` from inside or outside the loop.
- It returns `false` when the loop is terminated.

`go-go-goja` wraps that lower-level API in `pkg/runtimeowner.Runner`, which adds cancellation-aware `Call` and `Post`, closed/schedule errors, panic recovery, and an owner-context fast path.

### What is an EventEmitter?

Node's `EventEmitter` is a small synchronous listener registry:

```js
const EventEmitter = require("events");
const bus = new EventEmitter();

bus.on("order", (order) => {
  console.log(order.id);
});

bus.emit("order", { id: "123" });
```

Core semantics to preserve:

- `on(name, fn)` adds a listener and returns `this`.
- `once(name, fn)` adds a listener that removes itself before calling `fn`.
- `off(name, fn)` / `removeListener(name, fn)` removes one listener.
- `emit(name, ...args)` calls listeners synchronously in registration order and returns `true` when at least one listener existed.
- Listener return values are ignored.
- Emitting `"error"` without an error listener throws.
- Listener arrays should be copied before iteration so removing a listener during `emit()` does not corrupt the active dispatch.

For this ticket, implement a practical subset rather than a full Node conformance suite. The subset should be enough for internal event bus use and common packages that expect `require("events")`.

### What is a connected emitter?

A connected emitter is a JavaScript `EventEmitter` instance returned by a Go-backed factory function. The emitter is ordinary JavaScript from the caller's point of view, but Go owns a small connection record that knows how to deliver events from a specific Go-side resource into that emitter.

For example, a Watermill helper might expose `watermill.subscribe("orders")`. When JavaScript calls it, Go subscribes to the `orders` topic, creates an EventEmitter, returns it, and starts forwarding messages to that emitter's `"message"` event.

The connection record remembers only Go-owned scheduling metadata and a safe emitter identity, such as an internal numeric ID. It should not use a JS object from a background goroutine. A robust implementation stores returned emitters in an owner-thread registry, for example `globalThis.__goEmitterRegistry[id] = emitter`, and Go goroutines store only `id`. Every later emit schedules an owner-thread callback that looks up the emitter by ID and calls `emit()` there.

An ASCII diagram of the target runtime shape:

```text
JavaScript owner thread                       Go/background side
──────────────────────                       ──────────────────

const orders = watermill.subscribe("orders")
        │
        │ calls Go-backed factory
        ▼
create EventEmitter + registry id ───────▶ start Watermill goroutine
        │                                           │
        │ return emitter to JS                      │ message arrives
        ▼                                           │
orders.on("message", listener)                     │
                                                    │ schedule owner callback
                                                    ▼
                                            lookup emitter by id
                                                    │
                                                    ▼
                                            emitter.emit("message", msg)
                                                    │
                                                    ▼
                                            JS listener calls msg.ack()/nack()
```

The boundary is the scheduled owner callback. Everything on the Go/background side can run in arbitrary goroutines. Everything that touches JavaScript values, emitters, listeners, or `goja.Runtime` must happen on the owner thread.

## Current-state architecture and evidence

### Runtime construction already centralizes ownership

`engine.Factory.NewRuntime` currently creates the runtime and all scheduling primitives in one place:

- `goja.New()` creates the VM (`engine/factory.go:164`).
- `eventloop.NewEventLoop()` creates the loop (`engine/factory.go:165`).
- `go loop.Start()` starts the loop (`engine/factory.go:166`).
- `runtimeowner.NewRunner(vm, loop, ...)` creates the owner wrapper (`engine/factory.go:168-171`).
- `runtimebridge.Store(vm, runtimebridge.Bindings{Context, Loop, Owner})` makes runtime-scoped scheduling primitives discoverable by modules (`engine/factory.go:185-189`).

After that, the factory builds a `require.Registry`, registers default data-only modules, registers explicitly requested modules, lets runtime-scoped registrars add modules, then enables `require` on the VM (`engine/factory.go:191-229`). Runtime initializers then run with `RuntimeContext`, which contains `Context`, `VM`, `Require`, `Loop`, `Owner`, and `Values` (`engine/factory.go:231-244`).

This is the correct seam for installing helper modules or helper globals that JavaScript can call. It should not be used to create default Watermill emitters. Instead, use runtime initialization or runtime-scoped registration to expose factory functions; those factories create connected emitters only when JavaScript asks for them.

### Module registration is explicit and registry-based

Every normal module under `modules/` implements `modules.NativeModule`:

```go
type NativeModule interface {
    Name() string
    Doc() string
    Loader(*goja.Runtime, *goja.Object)
}
```

That interface is defined in `modules/common.go:29-33`. A module registers itself in `init()` by calling `modules.Register(...)` (`modules/common.go:94-99`). The engine imports module packages for their registration side effects in `engine/runtime.go:15-25`, but callers still choose what becomes available through `WithModules(...)` or the automatic data-only default list.

The most relevant existing module examples are:

- `modules/path/path.go`: a simple data-only module with TypeScript declarations.
- `modules/timer/timer.go`: an async module that uses `runtimebridge.Lookup(vm)` and `bindings.Owner.Post(...)` to settle promises safely from a goroutine (`modules/timer/timer.go:31-60`).
- `modules/fs/fs_async.go`: async filesystem helpers that do background Go work and then post promise resolution/rejection back to the owner (`modules/fs/fs_async.go:11-58`).

The `events` module should look like `path` for its pure JS constructor registration, while connected-emitter helpers should follow the `timer`/`fs_async` scheduling pattern: do Go work in goroutines, then schedule JavaScript work back onto the owner thread.

### Runtime-scoped state has an established hook

`engine.RuntimeModuleRegistrar` and `engine.RuntimeInitializer` exist because some modules or integrations need per-runtime state. The registrar context exposes `AddCloser` and `Values` before the require registry is enabled (`engine/runtime_modules.go:12-27`). The initializer context exposes `Require`, `Owner`, and `Values` after require is enabled (`engine/module_specs.go:22-37`).

For this feature, prefer runtime initializers or runtime module registrars for **factory installation**, not for default source subscription. A helper initializer can expose a JS function or module that calls back into Go, creates an EventEmitter on demand, and stores a Go connection manager in `RuntimeContext.Values` under a stable key.

### Default sandbox policy matters

The repository distinguishes data-only modules from host-access modules:

- Every runtime automatically gets `crypto`, `path`, `time`, and `timer` today (`engine/module_specs.go:165-173`, `pkg/doc/16-nodejs-primitives.md:30-40`).
- Host-access modules like `fs`, `os`, `exec`, and `database` require explicit opt-in (`pkg/doc/16-nodejs-primitives.md:41-52`, `pkg/doc/16-nodejs-primitives.md:296-306`).

A pure `events` module is data-only and safe to expose by default. The Watermill and fsnotify helpers, however, are host/integration capabilities. They should not be enabled by merely requiring `events`, and they should not create default emitters at runtime startup. Go application code should explicitly install the helper factory and decide which subscribers/watchers it exposes.

### External dependency contracts

#### goja_nodejs event loop

The local workspace checkout of `goja_nodejs` states that `RunOnLoop()` is ordered, safe from inside/outside the loop, and rejects scheduling after termination (`../goja_nodejs/eventloop/eventloop.go:314-321`). This supports the imported brief's scheduling rule and also validates `runtimeowner.Runner`, because `Runner` schedules through a `Scheduler` interface whose only method is `RunOnLoop(fn)` (`pkg/runtimeowner/types.go:9-12`).

#### Watermill subscriber settlement

Watermill's `message.Subscriber` interface says `Subscribe(ctx, topic)` returns a message channel and that `Ack()` must be called to receive the next message; `Nack()` should be called when processing fails and redelivery is desired (`watermill@v1.5.1/message/pubsub.go:25-36`). `Message.Ack()` and `Message.Nack()` are non-blocking and idempotent, and they return `false` when the opposite settlement already happened (`watermill@v1.5.1/message/message.go:96-147`).

That means Watermill messages should be delivered to JS as objects with explicit `ack()` and `nack()` functions. The adapter should not auto-ack after `emit()`, because `emit()` only proves that a listener was called, not that processing succeeded.

#### fsnotify watcher channels

`fsnotify.Watcher` exposes `Events chan Event` and `Errors chan error` (`fsnotify@v1.9.0/fsnotify.go:100-144`). `Watcher.Add(path)` monitors one path, and directory watches are not recursive (`fsnotify@v1.9.0/fsnotify.go:278-300`). This maps cleanly to a returned watcher emitter with local `"event"` and `"error"` events, but recursive watching must be added deliberately if needed.

## Problem statement and scope

The requested feature is to let JavaScript code running inside `go-go-goja` use EventEmitter-style APIs and call Go-backed factory functions that return configured emitters connected to explicit Go-side resources.

In scope:

- Add `require("events")` and `require("node:events")` with a practical Node-compatible `EventEmitter` class/function.
- Add reusable connected-emitter primitives: create an emitter on the owner thread, assign a safe internal ID, return it to JS, emit to it from Go by scheduling owner callbacks, and close it deterministically.
- Provide helper APIs for Go applications to expose JS-callable factory functions, not default emitters.
- Provide Watermill and fsnotify helper sketches or implementations that preserve lifecycle and settlement semantics when JavaScript explicitly requests a connection.
- Add tests for JavaScript EventEmitter behavior, scheduler safety, connected-emitter lifecycle, Watermill ack/nack behavior, and file watcher delivery.
- Add TypeScript declarations for the `events` module and optional helper modules if the repository's declaration generator should include them.
- Update docs that list default modules and async patterns.

Out of scope for the first implementation:

- Full Node `events` conformance, including every static helper and edge case.
- Full Node streams. Streams are a separate abstraction for backpressure, buffering, chunking, and `pipe()`.
- Auto-recursive filesystem watching.
- A JavaScript API that lets untrusted scripts subscribe directly to arbitrary host resources. Go should install only narrow helper factories for resources it intentionally exposes.
- Default Watermill emitters or default topic subscriptions created at runtime startup.
- Storing JavaScript listener functions in Go structures. Listener functions stay inside the JS `EventEmitter` instance.

## Proposed architecture

### Package layout

Use this layout:

```text
go-go-goja/
├── modules/
│   └── events/
│       ├── events.go              # NativeModule loader for events/node:events
│       ├── events_test.go         # JS EventEmitter behavior tests
│       └── event_emitter.js.go    # optional generated/embedded source constant
├── pkg/
│   └── jsevents/
│       ├── connected.go           # connected emitter registry and lifecycle
│       ├── connected_test.go      # owner-thread dispatch/lifecycle tests
│       ├── watermill.go           # optional helper that returns Watermill-connected emitters
│       ├── watermill_test.go      # ack/nack tests with fake subscriber or gochannel
│       ├── fsnotify.go            # optional helper that returns watcher-connected emitters
│       └── fsnotify_test.go       # temp-dir watcher smoke test
├── engine/
│   ├── runtime.go                 # blank import modules/events
│   ├── module_specs.go            # data-only list includes events aliases
│   └── granular_modules_test.go   # default availability expectations
└── pkg/doc/
    ├── 03-async-patterns.md       # mention event bus scheduling pattern
    └── 16-nodejs-primitives.md    # list events/node:events
```

Why split `modules/events` from `pkg/jsevents`?

- `modules/events` should be a normal data-only module. It should not import `engine`, because `engine/runtime.go` blank-imports modules. Importing `engine` from a module package would create an import cycle.
- `pkg/jsevents` may import `engine`, because it is an optional integration package that callers opt into from application code. The engine does not need to import it.
- This keeps `require("events")` simple and safe, while allowing richer connected-emitter factories without contorting the module registry.

### Layer diagram

```text
┌──────────────────────────────────────────────────────────────────┐
│ JavaScript code                                                   │
│                                                                  │
│ const EventEmitter = require("events")                           │
│ const orders = watermill.subscribe("orders")                     │
│ orders.on("message", msg => { ...; msg.ack(); })                │
└─────────────────────────────▲────────────────────────────────────┘
                              │ JS calls Go-backed factory
┌─────────────────────────────┴────────────────────────────────────┐
│ Helper module/global installed by Go application                  │
│                                                                  │
│ - validates JS request                                            │
│ - creates EventEmitter instance                                   │
│ - attaches close()/resource helpers                               │
│ - starts Go-side connection                                       │
│ - returns emitter to JS                                           │
└─────────────────────────────▲────────────────────────────────────┘
                              │ stores only safe emitter id in Go
┌─────────────────────────────┴────────────────────────────────────┐
│ pkg/jsevents connected-emitter manager                            │
│                                                                  │
│ - owner-thread emitter registry                                   │
│ - EmitterRef(id) for Go goroutines                                │
│ - Emit schedules owner callback                                   │
│ - Close removes registry entry and cancels resource context        │
└─────────────────────────────▲────────────────────────────────────┘
                              │ Go event source callbacks
┌─────────────────────────────┴────────────────────────────────────┐
│ Explicit Go helpers                                               │
│                                                                  │
│ - Watermill subscribe helper                                      │
│ - fsnotify watch helper                                           │
│ - application-specific lifecycle/event helpers                    │
└──────────────────────────────────────────────────────────────────┘
```

The important direction is JavaScript-to-Go setup, followed by Go-to-JavaScript delivery. JavaScript asks for an emitter that represents one concrete resource connection. Go then forwards events to that emitter until it is closed or the runtime shuts down.

### Module API: `require("events")`

The public JavaScript API should look like Node `events`, but the implementation behind it is Go-native. Support these imports:

```js
const EventEmitter = require("events");
const { EventEmitter: EE1 } = require("events");
const NodeEventEmitter = require("node:events");
const { EventEmitter: EE2 } = require("node:events");
```

Recommended JavaScript-visible methods for the first implementation:

| Method/property | Behavior |
|---|---|
| `new EventEmitter()` | Creates an emitter with a private listener map. |
| `on(name, fn)` / `addListener(name, fn)` | Validate `fn`, emit `newListener`, append, return `this`. |
| `once(name, fn)` | Wrap listener, remove wrapper before first call, expose `wrapped.listener = fn`. |
| `off(name, fn)` / `removeListener(name, fn)` | Remove first matching listener or once-wrapper, emit `removeListener`, return `this`. |
| `removeAllListeners(name?)` | Clear one event or all events, return `this`. |
| `listeners(name)` | Return a copy of listeners, unwrapping once wrappers if desired. |
| `rawListeners(name)` | Optional; return wrappers as registered. |
| `listenerCount(name)` | Return number of listeners for event. |
| `eventNames()` | Return known event names. |
| `emit(name, ...args)` | Copy active listener list, call each listener synchronously, return boolean. |
| `EventEmitter.EventEmitter` | Points back to constructor for destructuring compatibility. |

Go-native loader pseudocode:

```go
package events

type module struct{ name string }

var _ modules.NativeModule = (*module)(nil)
var _ modules.TypeScriptDeclarer = (*module)(nil) // optional but recommended

type EventEmitter struct {
    vm        *goja.Runtime
    listeners map[eventKey][]listenerEntry
}

func (m *module) Name() string { return m.name }

func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    registry := NewRuntimeRegistry(vm)
    RegisterRuntime(vm, registry) // package-level map keyed by *goja.Runtime, cleaned up by runtime close if possible

    constructor := vm.ToValue(func(call goja.ConstructorCall) *goja.Object {
        emitter := registry.NewEmitter()
        obj := vm.ToValue(emitter).(*goja.Object)
        obj.SetPrototype(call.This.Prototype())
        registry.TrackObject(obj, emitter)
        return obj
    }).(*goja.Object)

    proto := vm.NewObject()
    _ = proto.Set("on", registry.methodOn)
    _ = proto.Set("addListener", registry.methodOn)
    _ = proto.Set("once", registry.methodOnce)
    _ = proto.Set("off", registry.methodOff)
    _ = proto.Set("removeListener", registry.methodOff)
    _ = proto.Set("emit", registry.methodEmit)
    _ = proto.Set("listeners", registry.methodListeners)
    _ = proto.Set("listenerCount", registry.methodListenerCount)
    _ = proto.Set("removeAllListeners", registry.methodRemoveAllListeners)

    _ = constructor.Set("prototype", proto)
    proto.DefineDataProperty("constructor", constructor, goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)

    // Supports both require("events") and const { EventEmitter } = require("events").
    _ = constructor.Set("EventEmitter", constructor)
    _ = moduleObj.Set("exports", constructor)
}

func init() {
    modules.Register(&module{name: "events"})
    modules.Register(&module{name: "node:events"})
}
```

The exact registry cleanup mechanism can be refined during implementation. The important properties are that `EventEmitter` state is a Go struct, methods are Go functions, and `events.FromValue(...)` can recognize Go-native emitter objects passed back from JavaScript.

Superseded note: do not implement the following behavior with JavaScript source. Keep it only as behavioral pseudocode for the Go methods:

```js
(function () {
  function EventEmitter() {
    this._events = Object.create(null);
  }

  EventEmitter.prototype.on =
  EventEmitter.prototype.addListener = function (name, fn) {
    if (typeof fn !== "function") {
      throw new TypeError("listener must be a function");
    }

    // Node emits this before adding the listener. Avoid recursion if desired.
    if (name !== "newListener") {
      this.emit("newListener", name, fn);
    }

    const key = String(name);
    const list = this._events[key] || (this._events[key] = []);
    list.push(fn);
    return this;
  };

  EventEmitter.prototype.once = function (name, fn) {
    if (typeof fn !== "function") {
      throw new TypeError("listener must be a function");
    }
    const self = this;
    function wrapped() {
      self.removeListener(name, wrapped);
      return fn.apply(this, arguments);
    }
    wrapped.listener = fn;
    return this.on(name, wrapped);
  };

  EventEmitter.prototype.removeListener =
  EventEmitter.prototype.off = function (name, fn) {
    const key = String(name);
    const list = this._events[key];
    if (!list) return this;

    for (let i = 0; i < list.length; i++) {
      if (list[i] === fn || list[i].listener === fn) {
        const removed = list[i];
        list.splice(i, 1);
        if (list.length === 0) delete this._events[key];
        if (name !== "removeListener") {
          this.emit("removeListener", name, removed.listener || removed);
        }
        break;
      }
    }
    return this;
  };

  EventEmitter.prototype.emit = function (name) {
    const key = String(name);
    const list = this._events[key];
    if (!list || list.length === 0) {
      if (key === "error") {
        const err = arguments[1];
        throw err instanceof Error ? err : new Error(err ? String(err) : "Unhandled error event");
      }
      return false;
    }

    const args = Array.prototype.slice.call(arguments, 1);
    const snapshot = list.slice();
    for (let i = 0; i < snapshot.length; i++) {
      snapshot[i].apply(this, args);
    }
    return true;
  };

  EventEmitter.EventEmitter = EventEmitter;
  return EventEmitter;
})()
```

Implementation detail: stringifying event names is sufficient for the first pass unless the project needs Symbol event names. Node supports string and symbol names. If symbol support is needed, store events in a `Map` instead of a plain object. Goja supports modern JS well enough that a `Map`-based implementation is likely possible, but the object-based version is easier to review.

### Go API: `pkg/jsevents` connected emitters

The `pkg/jsevents` package should provide reusable infrastructure for helper modules that return connected emitters. The key type is not a global `Bus`; it is a per-runtime **Manager** plus per-emitter **EmitterRef** handles.

Suggested public API:

```go
package jsevents

const RuntimeValueKey = "jsevents.manager"

type ErrorHandler func(error)

type Manager struct {
    // unexported: runtime context, owner, registry name, id counter, onError
}

type EmitterRef struct {
    // unexported: manager pointer, emitter id, cancel/closer, closeOnce
}

type Option func(*options)

func WithRegistryName(name string) Option
func WithErrorHandler(fn ErrorHandler) Option

// Install returns a RuntimeInitializer that prepares the hidden emitter registry
// and stores *Manager in RuntimeContext.Values[RuntimeValueKey]. It does not
// create any Watermill/fsnotify emitters by itself.
func Install(opts ...Option) engine.RuntimeInitializer

// FromRuntime retrieves the manager installed by Install.
func FromRuntime(rt *engine.Runtime) (*Manager, bool)

// NewEmitterOnOwner creates an EventEmitter and returns both a Go handle and the
// JS object. It must be called on the owner thread, normally from a JS-callable
// Go factory function.
func (m *Manager) NewEmitterOnOwner(vm *goja.Runtime, opts ...EmitterOption) (*EmitterRef, *goja.Object, error)

// Emit schedules asynchronous event delivery to this one emitter.
func (r *EmitterRef) Emit(ctx context.Context, name string, args ...any) error

// EmitSync schedules delivery and waits for listener dispatch. Mostly useful for tests.
func (r *EmitterRef) EmitSync(ctx context.Context, name string, args ...any) (bool, error)

// Close cancels the Go-side resource and removes the JS emitter from the owner-thread registry.
func (r *EmitterRef) Close(ctx context.Context) error
```

Manager installation pseudocode:

```go
type initializer struct { opts options }

func (i initializer) ID() string { return "jsevents.connected-emitter-manager" }

func (i initializer) InitRuntime(ctx *engine.RuntimeContext) error {
    if ctx == nil || ctx.VM == nil || ctx.Require == nil || ctx.Owner == nil {
        return fmt.Errorf("jsevents: incomplete runtime context")
    }

    registryName := i.opts.registryName
    if registryName == "" {
        registryName = "__goConnectedEmitters"
    }

    // Owner-thread setup: hidden-ish registry object. It can be non-enumerable
    // if desired, but a plain object is enough for the first implementation.
    registry := ctx.VM.NewObject()
    if err := ctx.VM.Set(registryName, registry); err != nil {
        return err
    }

    manager := &Manager{
        ctx:          ctx.Context,
        owner:        ctx.Owner,
        registryName: registryName,
        onError:      i.opts.onError,
    }
    ctx.SetValue(RuntimeValueKey, manager)
    return nil
}
```

Emitter creation pseudocode. This is called from a JS-callable Go factory, so it is already on the owner thread:

```go
func (m *Manager) NewEmitterOnOwner(vm *goja.Runtime, opts ...EmitterOption) (*EmitterRef, *goja.Object, error) {
    ctorValue, err := require.Require(vm, "events") // pseudocode; use ctx.Require if available
    if err != nil { return nil, nil, err }

    ctor, ok := goja.AssertConstructor(ctorValue)
    if !ok { return nil, nil, fmt.Errorf("events export is not a constructor") }

    emitter, err := ctor(nil)
    if err != nil { return nil, nil, err }

    id := m.nextID()
    registry := vm.Get(m.registryName).ToObject(vm)
    _ = registry.Set(id, emitter)

    ref := &EmitterRef{manager: m, id: id}

    // Attach close() so JS can release the Go-side resource.
    _ = emitter.Set("close", func() bool {
        return ref.Close(m.ctx) == nil
    })

    return ref, emitter, nil
}
```

Dispatch pseudocode:

```go
func (r *EmitterRef) Emit(ctx context.Context, name string, args ...any) error {
    if r == nil || r.manager == nil {
        return fmt.Errorf("jsevents: nil emitter ref")
    }
    copiedArgs := append([]any(nil), args...)
    return r.manager.owner.Post(ctx, "jsevents.emit."+r.id+"."+name, func(_ context.Context, vm *goja.Runtime) {
        _, err := r.emitOnOwner(vm, name, copiedArgs...)
        if err != nil { r.manager.report(err) }
    })
}

func (r *EmitterRef) emitOnOwner(vm *goja.Runtime, name string, args ...any) (bool, error) {
    registry := vm.Get(r.manager.registryName).ToObject(vm)
    emitterValue := registry.Get(r.id)
    if emitterValue == nil || goja.IsUndefined(emitterValue) || goja.IsNull(emitterValue) {
        return false, fmt.Errorf("jsevents: emitter %s is closed", r.id)
    }

    emitter := emitterValue.ToObject(vm)
    emit, ok := goja.AssertFunction(emitter.Get("emit"))
    if !ok {
        return false, fmt.Errorf("jsevents: emitter %s has no emit function", r.id)
    }

    argv := []goja.Value{vm.ToValue(name)}
    for _, arg := range args {
        argv = append(argv, vm.ToValue(arg))
    }

    ret, err := emit(emitter, argv...)
    if err != nil { return false, err }
    return ret.ToBoolean(), nil
}

func (r *EmitterRef) Close(ctx context.Context) error {
    var err error
    r.closeOnce.Do(func() {
        if r.cancel != nil { r.cancel() }
        err = r.manager.owner.Post(ctx, "jsevents.close."+r.id, func(_ context.Context, vm *goja.Runtime) {
            registry := vm.Get(r.manager.registryName).ToObject(vm)
            _ = registry.Delete(r.id)
        })
    })
    return err
}
```

This registry-by-ID pattern avoids using a JavaScript object from background goroutines. Go goroutines only hold an `EmitterRef` with an ID. The JS object is looked up and used inside owner-thread callbacks.

### Watermill helper API

Watermill support should be an opt-in helper that installs a JS-callable factory. It should not create topic subscriptions by default.

Suggested Go API:

```go
type WatermillOptions struct {
    ModuleName string        // default "watermill" or app-specific name
    Subscriber message.Subscriber
    AllowTopic func(topic string) bool
}

func WatermillHelper(opts WatermillOptions) engine.RuntimeInitializer
```

Suggested JavaScript API:

```js
const orders = watermill.subscribe("orders");

orders.on("message", (msg) => {
  try {
    const order = JSON.parse(msg.payload);
    console.log("order", order.id);
    msg.ack();
  } catch (err) {
    msg.nack();
  }
});

orders.on("error", (err) => {
  console.error("watermill error", err.message || err);
});

// Later, if the script owns the lifecycle:
orders.close();
```

Factory pseudocode:

```go
func installWatermill(ctx *engine.RuntimeContext, manager *jsevents.Manager, sub message.Subscriber) error {
    watermillObj := ctx.VM.NewObject()

    _ = watermillObj.Set("subscribe", func(call goja.FunctionCall) goja.Value {
        topic := call.Argument(0).String()
        if !allowed(topic) {
            panic(ctx.VM.NewGoError(fmt.Errorf("watermill topic %q is not allowed", topic)))
        }

        ref, emitter, err := manager.NewEmitterOnOwner(ctx.VM)
        if err != nil {
            panic(ctx.VM.NewGoError(err))
        }

        subCtx, cancel := context.WithCancel(ctx.Context)
        ref.SetCancel(cancel) // pseudocode

        go runWatermillSubscription(subCtx, ref, sub, topic)

        return emitter
    })

    return ctx.VM.Set("watermill", watermillObj)
}
```

Subscription goroutine pseudocode:

```go
func runWatermillSubscription(ctx context.Context, ref *jsevents.EmitterRef, sub message.Subscriber, topic string) {
    messages, err := sub.Subscribe(ctx, topic)
    if err != nil {
        _ = ref.Emit(ctx, "error", map[string]any{
            "source": "watermill",
            "topic": topic,
            "message": err.Error(),
        })
        _ = ref.Close(ctx)
        return
    }

    for {
        select {
        case <-ctx.Done():
            return
        case msg, ok := <-messages:
            if !ok {
                _ = ref.Emit(ctx, "close")
                return
            }
            dispatchWatermillMessage(ctx, ref, msg)
        }
    }
}
```

Message dispatch still uses explicit settlement:

```go
func dispatchWatermillMessage(ctx context.Context, ref *jsevents.EmitterRef, msg *message.Message) {
    // This helper likely needs a lower-level EmitWithOwnerBuilder API because
    // ack()/nack() functions should be created as JS functions on the owner thread.
    _ = ref.EmitWithBuilder(ctx, "message", func(vm *goja.Runtime) (goja.Value, error) {
        jsMsg := vm.NewObject()
        var settleOnce sync.Once

        _ = jsMsg.Set("uuid", msg.UUID)
        _ = jsMsg.Set("payload", string(msg.Payload))
        _ = jsMsg.Set("metadata", copyMetadata(msg.Metadata))
        _ = jsMsg.Set("ack", func() bool {
            called := false
            settleOnce.Do(func() { called = msg.Ack() })
            return called
        })
        _ = jsMsg.Set("nack", func() bool {
            called := false
            settleOnce.Do(func() { called = msg.Nack() })
            return called
        })
        return jsMsg, nil
    })
}
```

The helper should nack if scheduling fails, if no listener handles the message and that is the chosen policy, or if listener dispatch throws. It should not ack merely because `emit("message", msg)` was called.

### fsnotify helper API

The fsnotify helper follows the same connected-emitter pattern. It should expose a factory such as `fswatch.watch(path)` only when the Go application intentionally installs it.

Suggested JavaScript API:

```js
const watcher = fswatch.watch("/tmp/demo");

watcher.on("event", (ev) => {
  console.log(ev.name, ev.op);
});

watcher.on("error", (err) => {
  console.error(err.message || err);
});

watcher.close();
```

Go helper pseudocode:

```go
func installFSWatch(ctx *engine.RuntimeContext, manager *jsevents.Manager) error {
    obj := ctx.VM.NewObject()
    _ = obj.Set("watch", func(path string) goja.Value {
        ref, emitter, err := manager.NewEmitterOnOwner(ctx.VM)
        if err != nil { panic(ctx.VM.NewGoError(err)) }

        watchCtx, cancel := context.WithCancel(ctx.Context)
        ref.SetCancel(cancel)
        go runWatcher(watchCtx, ref, path)

        return emitter
    })
    return ctx.VM.Set("fswatch", obj)
}

func runWatcher(ctx context.Context, ref *jsevents.EmitterRef, path string) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        _ = ref.Emit(ctx, "error", map[string]any{"source": "fsnotify", "message": err.Error()})
        _ = ref.Close(ctx)
        return
    }
    defer watcher.Close()

    if err := watcher.Add(path); err != nil {
        _ = ref.Emit(ctx, "error", map[string]any{"source": "fsnotify", "message": err.Error()})
        _ = ref.Close(ctx)
        return
    }

    for {
        select {
        case <-ctx.Done():
            return
        case ev, ok := <-watcher.Events:
            if !ok { return }
            _ = ref.Emit(ctx, "event", map[string]any{"name": ev.Name, "op": ev.Op.String()})
        case err, ok := <-watcher.Errors:
            if !ok { return }
            _ = ref.Emit(ctx, "error", map[string]any{"source": "fsnotify", "message": err.Error()})
        }
    }
}
```

Repository guideline note: the project guidance says to use `errgroup` when starting goroutines. For helper implementations, make goroutine ownership explicit with a cancelable per-emitter context and runtime closer. If a helper starts long-running goroutines, tests should prove `close()` and runtime shutdown stop them.

## Implementation phases

### Phase 1: Add the Go-native `events` module

Files to add/update:

- Add `modules/events/events.go`.
- Add `modules/events/events_test.go`.
- Add optional internal files such as `modules/events/emitter.go` and `modules/events/registry.go` if that keeps the implementation readable.
- Update `engine/runtime.go` with `_ "github.com/go-go-golems/go-go-goja/modules/events"`.
- Decide whether to update `engine/module_specs.go` `dataOnlyDefaultRegistryModuleNames` to include `events` and `node:events`.
- Update `engine/granular_modules_test.go` if `events` is enabled by default.

Steps:

1. Create `modules/events`.
2. Implement a Go backing type such as `type EventEmitter struct { ... }` with listener storage.
3. Implement a native constructor with `func(goja.ConstructorCall) *goja.Object`.
4. Create a prototype object whose methods are Go functions.
5. Return Go-backed objects from the constructor with `vm.ToValue(emitter).(*goja.Object)` and set their prototype to `call.This.Prototype()`.
6. Register both `events` and `node:events` aliases.
7. Attach `EventEmitter` to the exported constructor object so `const { EventEmitter } = require("events")` works.
8. Export a Go helper such as `FromValue(value goja.Value) (*EventEmitter, *goja.Object, bool)` so Go-backed helper functions can adopt JS-created emitters.
9. Add tests that execute JavaScript through `engine.NewBuilder().Build()`.

Core tests:

```go
func TestEventsModuleExportsGoNativeConstructor(t *testing.T) {
    rt := newRuntime(t)
    ret := runJS(t, rt, `
      const EventEmitter = require("events");
      const ee = new EventEmitter();
      JSON.stringify({
        functionExport: typeof EventEmitter,
        sameNamed: EventEmitter === require("events").EventEmitter,
        hasOn: typeof ee.on,
        hasEmit: typeof ee.emit,
        instance: ee instanceof EventEmitter
      });
    `)
    require.JSONEq(t, `{"functionExport":"function","sameNamed":true,"hasOn":"function","hasEmit":"function","instance":true}`, ret)
}

func TestEventEmitterSynchronousEmitAndOnce(t *testing.T) {
    ret := runJS(t, rt, `
      const EventEmitter = require("events");
      const ee = new EventEmitter();
      const calls = [];
      ee.on("x", v => calls.push("on:" + v));
      ee.once("x", v => calls.push("once:" + v));
      const first = ee.emit("x", 1);
      const second = ee.emit("x", 2);
      JSON.stringify({ first, second, calls, count: ee.listenerCount("x") });
    `)
    require.JSONEq(t, `{"first":true,"second":true,"calls":["on:1","once:1","on:2"],"count":1}`, ret)
}

func TestGoCanAdoptJSCreatedEmitter(t *testing.T) {
    // Install a Go function that receives an emitter created in JS, unwraps it
    // with events.FromValue, stores an EmitterRef, and later emits to it from Go.
}
```

### Phase 2: Add TypeScript declarations

Files to update:

- `modules/events/events.go` implements `modules.TypeScriptDeclarer`.
- `cmd/gen-dts` should not need changes if the descriptor validates.
- Generated declaration output used by the Bun demo may need regeneration if checked into the repo.

Descriptor sketch:

```go
func (m *module) TypeScriptModule() *spec.Module {
    return &spec.Module{
        Name: m.name,
        RawDTS: []string{
            "type EventName = string | symbol;",
            "type Listener = (...args: any[]) => void;",
            "class EventEmitter {",
            "  constructor();",
            "  on(name: EventName, listener: Listener): this;",
            "  addListener(name: EventName, listener: Listener): this;",
            "  once(name: EventName, listener: Listener): this;",
            "  off(name: EventName, listener: Listener): this;",
            "  removeListener(name: EventName, listener: Listener): this;",
            "  removeAllListeners(name?: EventName): this;",
            "  emit(name: EventName, ...args: any[]): boolean;",
            "  listeners(name: EventName): Listener[];",
            "  listenerCount(name: EventName): number;",
            "  eventNames(): EventName[];",
            "}",
            "export = EventEmitter;",
            "export { EventEmitter };",
        },
    }
}
```

If the renderer does not support CommonJS `export =` well through structured fields, use `RawDTS` as shown. Validate with:

```bash
go run ./cmd/gen-dts --out /tmp/goja-modules.d.ts --module events,node:events --strict
```

### Phase 3: Add `pkg/jsevents` connected-emitter manager

Files to add:

- `pkg/jsevents/connected.go`
- `pkg/jsevents/connected_test.go`

Implementation guidance:

1. Define `RuntimeValueKey = "jsevents.manager"`.
2. Define `Manager` with runtime context, owner runner, registry name, ID generator, and error handler.
3. Define `EmitterRef` with manager pointer, emitter ID, optional cancel function, and `sync.Once` close guard.
4. Define `Install(opts...) engine.RuntimeInitializer` that creates the hidden owner-thread emitter registry and stores the manager in `ctx.Values`.
5. Implement `FromRuntime(rt)`.
6. Implement `NewEmitterOnOwner(vm)` for JS-callable Go factories.
7. Implement `Emit`, `EmitSync`, `EmitWithBuilder`, and `Close`.
8. Ensure Go goroutines hold only `EmitterRef`/ID data and never touch JS objects directly.

Test scenarios:

- Installing the initializer creates a manager but does not create any default event-source emitters.
- A JS-callable test factory returns a fresh emitter.
- JS can register listeners on the returned emitter.
- A Go goroutine can emit to that specific emitter through `EmitterRef.EmitSync`.
- Closing the emitter cancels the resource context and removes the registry entry.
- Emitting after close returns an error or reports an error deterministically.
- Many emitted events preserve order for a single emitter.

Test pseudocode:

```go
func TestConnectedEmitterFactoryReturnsEmitter(t *testing.T) {
    rt, manager := newRuntimeWithManager(t)

    var ref *jsevents.EmitterRef
    _, err := rt.Owner.Call(context.Background(), "test.installFactory", func(_ context.Context, vm *goja.Runtime) (any, error) {
        factory := func() goja.Value {
            var err error
            var emitter *goja.Object
            ref, emitter, err = manager.NewEmitterOnOwner(vm)
            if err != nil {
                panic(vm.NewGoError(err))
            }
            return emitter
        }
        return nil, vm.Set("makeEmitter", factory)
    })
    require.NoError(t, err)

    runJS(t, rt, `
      globalThis.seen = [];
      const emitter = makeEmitter();
      emitter.on("tick", n => seen.push(n));
    `)

    for i := 0; i < 3; i++ {
        _, err := ref.EmitSync(context.Background(), "tick", i)
        require.NoError(t, err)
    }

    got := runJS(t, rt, `JSON.stringify(globalThis.seen)`)
    require.Equal(t, `[0,1,2]`, got)
}
```

### Phase 4: Add Watermill connected-emitter helper

Files to add:

- `pkg/jsevents/watermill.go`
- `pkg/jsevents/watermill_test.go`

Implementation guidance:

1. Provide an opt-in helper initializer or registrar that installs a JS factory such as `watermill.subscribe(topic)`.
2. Do not create any Watermill subscriptions during runtime startup.
3. When JS calls the factory, create a connected emitter and return it immediately.
4. Start a per-emitter goroutine with a per-emitter cancel context.
5. Call `message.Subscriber.Subscribe(ctx, topic)` inside that goroutine unless the concrete subscriber is known to be cheap/non-blocking.
6. For each message, schedule owner-thread construction of the JS message object.
7. Expose explicit `ack()` and `nack()` methods on the message payload.
8. Attach `close()` to the emitter so JS can cancel the subscription.
9. If scheduling fails, call `msg.Nack()` immediately.
10. If JS emit throws or no listener exists, call `msg.Nack()` unless the application chooses a different documented policy.
11. Do not auto-ack after successful delivery. Let JS call `ack()`.

Test scenarios:

- `watermill.subscribe("orders")` returns an emitter and does not subscribe until called.
- A listener that calls `msg.ack()` closes `msg.Acked()`.
- A listener that calls `msg.nack()` closes `msg.Nacked()`.
- No listener nacks the message or emits a helper-level error according to the chosen policy.
- Listener throw nacks the message and reports the error.
- Double settlement only settles once.
- `emitter.close()` cancels the subscription goroutine.
- Runtime close cancels all open connected emitters.

### Phase 5: Add fsnotify connected-emitter helper

Files to add:

- `pkg/jsevents/fsnotify.go`
- `pkg/jsevents/fsnotify_test.go`

Implementation guidance:

1. Provide an opt-in helper factory such as `fswatch.watch(path)`.
2. Do not start any watchers during runtime startup.
3. When JS calls `watch(path)`, create a connected emitter, start one watcher goroutine, and return the emitter.
4. Emit file events on `"event"`, not on a global `"fs:event"` namespace.
5. Emit watcher errors on that emitter's `"error"` event.
6. Attach `close()` to cancel the watcher and remove the emitter registry entry.
7. Document non-recursive behavior clearly.

Test scenarios:

- `fswatch.watch(tempDir)` returns an emitter.
- Writing a file eventually emits an `"event"` payload to that returned emitter.
- `watcher.close()` stops delivery.
- Runtime close stops open watchers.
- If an error can be injected through a helper loop, assert it emits an `"error"` payload on the returned emitter.

### Phase 6: Documentation and examples

Files to update:

- `README.md`: add `events` to the quick list of Node-style primitives and show a connected-emitter factory example.
- `pkg/doc/03-async-patterns.md`: add a section for JS-called Go factories that return connected emitters.
- `pkg/doc/16-nodejs-primitives.md`: add `events` and `node:events` to available primitives and implementation map.
- Optional: add an example under `cmd/` or `examples/` that installs a helper and lets JS call `watermill.subscribe(...)` or a simpler fake source factory.

Example application setup:

```go
factory, err := engine.NewBuilder().
    WithRuntimeInitializers(
        jsevents.Install(jsevents.WithErrorHandler(func(err error) {
            log.Error().Err(err).Msg("javascript connected emitter failed")
        })),
        jsevents.WatermillHelper(jsevents.WatermillOptions{
            ModuleName: "watermill",
            Subscriber: ordersSubscriber,
            AllowTopic: func(topic string) bool { return topic == "orders" },
        }),
    ).
    Build()
if err != nil { return err }

rt, err := factory.NewRuntime(ctx)
if err != nil { return err }
defer rt.Close(ctx)

_, err = rt.Owner.Call(ctx, "app.install-listeners", func(_ context.Context, vm *goja.Runtime) (any, error) {
    _, err := vm.RunString(`
      const orders = watermill.subscribe("orders");
      orders.on("message", (msg) => {
        try {
          console.log(JSON.parse(msg.payload));
          msg.ack();
        } catch (err) {
          msg.nack();
        }
      });
    `)
    return nil, err
})
```

## Design decisions

### Decision 1: Use the existing runtime owner instead of creating a new engine wrapper

The imported brief includes a standalone `Engine` type that creates its own event loop. That is useful as a minimal sketch, but it duplicates `go-go-goja`'s existing `engine.Runtime`. The repository already exposes `rt.VM`, `rt.Require`, `rt.Loop`, `rt.Owner`, lifecycle context, closers, and values (`engine/runtime.go:28-36`, `engine/runtime.go:82-115`).

Use `engine.Runtime` and `runtimeowner.Runner` so the feature composes with existing modules, plugin registrars, lifecycle shutdown, and tests.

### Decision 2: Make `events` default data-only, but keep connected-source helpers opt-in

`events` is a pure listener registry. It does not expose host filesystem, process environment, command execution, or database access. It belongs with the default data-only primitives.

Watermill and fsnotify helpers are different: they connect JavaScript to host/infrastructure event sources. They should be wired by Go application code, not activated by `require("events")`, and not created as default emitters during runtime startup.

### Decision 3: JavaScript calls Go factories to obtain connected emitters

The primary integration pattern is:

```js
const source = helper.openOrSubscribe(...);
source.on("event", listener);
source.close();
```

This is clearer than a global event namespace because the returned emitter is the resource handle. It also makes lifecycle explicit: closing the emitter closes the Go-side resource connection.

### Decision 4: Store emitter IDs, not JS object handles, in background goroutines

A tempting optimization is to cache the JS emitter object or its `emit` function in Go and call it later. Do not do this from background goroutines. The goja_nodejs contract explicitly says the runtime passed to `RunOnLoop()` and values derived from it must not be used outside the scheduled function (`../goja_nodejs/eventloop/eventloop.go:314-321`).

Use an owner-thread registry keyed by emitter ID. Background goroutines hold `EmitterRef{id}` and schedule callbacks. The callback looks up the JS emitter by ID and calls `emit()` on the owner thread.

### Decision 5: Watermill messages require explicit settlement

Node `emit()` ignores listener return values, and Watermill requires message settlement for progress (`watermill@v1.5.1/message/pubsub.go:25-36`). Therefore listener return values must not drive ack/nack. The JS payload should expose explicit `ack()` and `nack()` methods.

### Decision 6: Start with EventEmitter, not streams

Streams solve backpressure, buffering, chunking, and `pipe()` composition. EventEmitter solves listener dispatch. Watermill messages are message-lifecycle objects with ack/nack, not byte chunks. File watcher events are discrete notifications. Start with EventEmitter; add readable-like wrappers only when a specific high-volume or stream-composition requirement appears.

## Alternatives considered

### Alternative: Implement only a JS `events` module and no connected emitters

This would allow JavaScript-to-JavaScript events but would not solve the integration requirement: JS needs a way to request resource-backed emitters from Go, and Go needs a reusable safe dispatch pattern.

### Alternative: Expose Go methods directly on `require("events")`

Mixing host integration APIs into the Node-compatible `events` module would make compatibility murky and could expose host capabilities accidentally. Keep the compatibility module pure and put Go integration in explicit helper modules/globals installed by the application.

### Alternative: Configure global `goEvents` and emit namespaced events

The earlier draft used `globalThis.goEvents` with event names like `"watermill:orders"`. This works technically, but it hides lifecycle and makes resource ownership less obvious. A returned emitter is a better handle: it can have `close()`, can emit local `"message"`/`"error"` events, and maps directly to one Go-side connection.

### Alternative: Let Watermill listener return values control ack/nack

This would look convenient:

```js
orders.on("message", msg => true); // auto-ack?
```

Reject this because Node `emit()` ignores listener returns, multiple listeners make return interpretation ambiguous, and async listener promises would be especially confusing. Explicit `ack()` / `nack()` is clearer and matches Watermill's lifecycle.

### Alternative: Full Node EventEmitter implementation immediately

Full conformance includes max listener warnings, `prependListener`, `prependOnceListener`, static helpers, async resource variants, symbols, `captureRejections`, and more. That is too much for the first implementation. The guide recommends a focused subset with tests, leaving a compatibility expansion path.

## Validation strategy

Run these commands during implementation:

```bash
# Fast package tests while developing modules/events
go test ./modules/events -count=1

# Connected-emitter helper tests
go test ./pkg/jsevents -count=1

# Engine composition tests
go test ./engine -count=1

# Declaration generation if TypeScript descriptor is added
go run ./cmd/gen-dts --out /tmp/goja-modules.d.ts --module events,node:events --strict

# Full repository test before handoff
go test ./... -count=1
```

Use the race detector for the connected-emitter helpers if test runtime is acceptable:

```bash
go test -race ./pkg/jsevents ./modules/events -count=1
```

Manual smoke test shape:

```go
factory, _ := engine.NewBuilder().
    WithRuntimeInitializers(jsevents.Install()).
    Build()
rt, _ := factory.NewRuntime(context.Background())
defer rt.Close(context.Background())

manager, _ := jsevents.FromRuntime(rt)

var ref *jsevents.EmitterRef
_, _ = rt.Owner.Call(context.Background(), "smoke.factory", func(_ context.Context, vm *goja.Runtime) (any, error) {
    return nil, vm.Set("makeEmitter", func() goja.Value {
        var emitter *goja.Object
        var err error
        ref, emitter, err = manager.NewEmitterOnOwner(vm)
        if err != nil { panic(vm.NewGoError(err)) }
        return emitter
    })
})

_, _ = rt.Owner.Call(context.Background(), "smoke.listeners", func(_ context.Context, vm *goja.Runtime) (any, error) {
    _, err := vm.RunString(`
      globalThis.seen = [];
      const source = makeEmitter();
      source.on("hello", ev => seen.push(ev.name));
    `)
    return nil, err
})

_, _ = ref.EmitSync(context.Background(), "hello", map[string]any{"name": "world"})
```

Then read `JSON.stringify(globalThis.seen)` through `rt.Owner.Call` and expect `["world"]`.

## Risks and review checklist

### Concurrency risks

- **Risk:** A helper accidentally calls `vm.ToValue`, `vm.RunString`, or JS functions outside owner callbacks.
  - **Review:** Search for `goja.Runtime` usage in `pkg/jsevents`; all direct VM usage should be inside JS-callable owner-thread functions or callbacks passed to `Owner.Call`/`Owner.Post`, except initializer setup before runtime exposure.

- **Risk:** A background goroutine captures a `*goja.Object` and uses it directly.
  - **Review:** Goroutines should hold `EmitterRef`/ID and schedule work. Owner callbacks should perform registry lookup and JS calls.

- **Risk:** Asynchronous `EmitterRef.Emit` reports listener errors only via an error handler.
  - **Review:** Document this clearly. Tests that need errors should use `EmitSync` or a helper-specific error channel.

- **Risk:** Watermill messages can stall if JS forgets to settle.
  - **Review:** This is expected first-pass behavior. Add timeout/auto-nack only with explicit product requirements.

### Lifecycle risks

- **Risk:** Watcher/subscription goroutines leak after emitter close or runtime close.
  - **Review:** Each connected emitter needs a cancel function. Runtime shutdown should close all open emitters or cancel their parent context. `engine.Runtime.Close` cancels the runtime context before closers run (`engine/runtime.go:96-107`).

- **Risk:** Emitting after emitter close silently drops events.
  - **Review:** `EmitterRef.Emit` should return or report a deterministic closed-emitter error after the registry entry is removed.

- **Risk:** A helper subscribes to Watermill during runtime startup.
  - **Review:** Startup should install only the factory. The subscription starts only when JS calls the factory.

### Compatibility risks

- **Risk:** Some npm code expects methods beyond the initial EventEmitter subset.
  - **Review:** Add missing methods in small tested increments. Do not add streams as a compatibility band-aid unless the package truly needs streams.

- **Risk:** TypeScript `export =` rendering may not fit the current descriptor renderer.
  - **Review:** Use `RawDTS` and run `cmd/gen-dts --strict` for `events,node:events`.

## File reference map

### Repository files

| File | Why it matters |
|---|---|
| `engine/factory.go:156-246` | Runtime construction, event loop start, owner creation, module registration, runtime initializers. |
| `engine/runtime.go:15-25` | Blank imports that make module `init()` registration happen. Add `modules/events` here. |
| `engine/runtime.go:82-115` | Runtime close order and context cancellation. Important for connected-emitter lifecycle. |
| `engine/module_specs.go:16-20` | `ModuleSpec` interface for static module registration. |
| `engine/module_specs.go:96-180` | Default registry registration and data-only default module list. Add `events` aliases here if `events` is default. |
| `engine/runtime_modules.go:12-27` | Runtime-scoped registrar context; useful for helper modules that install JS factories before require is enabled. |
| `modules/common.go:29-33` | `NativeModule` interface that `modules/events` must implement. |
| `modules/common.go:94-117` | Default registry and registration helpers. |
| `modules/timer/timer.go:31-60` | Existing async pattern: background goroutine posts back to owner before resolving JS promise. |
| `modules/fs/fs_async.go:11-58` | Existing async helpers that settle promises on owner thread. |
| `pkg/runtimeowner/types.go:9-25` | Scheduler and Runner interfaces. |
| `pkg/runtimeowner/runner.go:62-160` | `Call` and `Post` scheduling behavior and errors. |
| `pkg/runtimebridge/runtimebridge.go:12-18` | Runtime-owned bindings exposed to modules. |
| `pkg/doc/03-async-patterns.md:20-64` | Existing documentation of owner-thread async rules. |
| `pkg/doc/16-nodejs-primitives.md:28-52` | Existing documentation of default primitives and sandbox policy. |
| `cmd/gen-dts/main.go` | TypeScript declaration generation from `modules.TypeScriptDeclarer`. |

### Workspace and dependency files

| File | Why it matters |
|---|---|
| `../go.work` | The workspace now includes `./goja_nodejs`, so local event loop source is available while implementing this ticket. |
| `../goja_nodejs/eventloop/eventloop.go:314-321` | Authoritative local `RunOnLoop()` contract. |
| `$GOMODCACHE/github.com/ThreeDotsLabs/watermill@v1.5.1/message/pubsub.go:25-36` | Subscriber contract: ack/nack required for progress. |
| `$GOMODCACHE/github.com/ThreeDotsLabs/watermill@v1.5.1/message/message.go:96-147` | Ack/nack idempotency and return behavior. |
| `$GOMODCACHE/github.com/fsnotify/fsnotify@v1.9.0/fsnotify.go:100-144` | Watcher event/error channels. |
| `$GOMODCACHE/github.com/fsnotify/fsnotify@v1.9.0/fsnotify.go:278-300` | Watch add semantics and non-recursive directory watching. |

## Appendix: implementation checklist for the intern

1. Read `modules/path/path.go` and `modules/timer/timer.go` before coding.
2. Add `modules/events` and tests for `require("events")` behavior.
3. Add the blank import in `engine/runtime.go`.
4. Decide whether `events` and `node:events` should be in the automatic data-only default list or only selected explicitly.
5. Run `go test ./modules/events ./engine -count=1`.
6. Add TypeScript declaration support and validate with `cmd/gen-dts`.
7. Add `pkg/jsevents.Manager`, `EmitterRef`, and the connected-emitter registry.
8. Test a JS-callable Go factory that returns an emitter before implementing Watermill.
9. Implement Watermill helper as a JS-called factory with explicit `ack()` / `nack()`.
10. Implement fsnotify helper as a JS-called factory only after connected-emitter tests pass.
11. Update docs and examples.
12. Run `go test ./... -count=1` before code review.

## Appendix: imported brief deltas

The imported brief proposed a minimal `jsevents.Engine` that creates a new `eventloop.EventLoop` and global `goEvents`. The first draft of this guide adapted that into a global `Bus`; this revision changes the shape again based on feedback: use `engine.Runtime`, but expose **JS-called factory functions that return connected emitters**.

Keep these ideas from the brief exactly:

- Go owns cross-goroutine event delivery.
- JavaScript receives EventEmitter façades.
- Do not touch goja runtime values from Watermill or filewatcher goroutines.
- Watermill messages expose explicit `ack()` and `nack()`.
- Start with EventEmitter; postpone streams until backpressure or chunking is a concrete requirement.

Change these ideas from the brief/draft:

- Do not create default Watermill emitters.
- Do not require JavaScript to listen on a global namespaced `goEvents` bus.
- Do return configured emitter objects from Go-backed JS factory calls.
- Do attach lifecycle (`close`) to each returned emitter.
