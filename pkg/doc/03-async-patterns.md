---
Title: Asynchronous Patterns with Promises
Slug: async-patterns
Short: Implementing Promise-based and callback-style async operations
Topics:
- async
- promises
- callbacks
- concurrency
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Asynchronous operations in go-go-goja bridge Go's goroutine model with JavaScript's Promise and callback patterns. The key constraint is that all JavaScript VM interactions must occur through the runtime owner. Background goroutines may do blocking Go work, but they must schedule Promise settlement, JavaScript callback invocation, and `goja.Value` access back onto the owner.

## Core async principle

A goja runtime is single-threaded from JavaScript's point of view. Any operation that touches JavaScript values, calls JavaScript functions, or resolves Promises must happen on the runtime owner.

For native modules loaded inside an `engine.Runtime`, use `pkg/runtimebridge.RuntimeServices`:

```go
runtimeServices, ok := runtimebridge.Lookup(vm)
if !ok || runtimeServices.Owner == nil {
    panic(vm.NewGoError(fmt.Errorf("module requires runtime services")))
}
```

`RuntimeServices` gives module code:

- `Owner`: serialized access to the VM;
- `Loop`: the underlying event loop when low-level integration is unavoidable;
- `Lifetime()`: the runtime-owned lifetime context;
- helper methods that make context intent explicit.

## Runtime contexts

The runtime API deliberately separates several context meanings:

| Context | Purpose | Typical API |
| --- | --- | --- |
| Startup context | Runtime construction and initializers | `engine.WithStartupContext(ctx)` |
| Lifetime context | Runtime-owned resources after construction | `engine.WithLifetimeContext(ctx)`, `RuntimeServices.Lifetime()` |
| Current owner-entry context | The context for the JavaScript/native callback currently running on owner | `runtimebridge.CurrentOwnerContext(vm)` |
| Custom operation context | HTTP request, Discord event, hardware event, or other external operation | `CallWithCustomContext`, `PostWithCustomContext` |

Create runtimes with explicit startup and lifetime contexts:

```go
rt, err := factory.NewRuntime(
    engine.WithStartupContext(ctx),
    engine.WithLifetimeContext(ctx),
)
```

Use separate contexts when construction and runtime lifetime are different:

```go
rt, err := factory.NewRuntime(
    engine.WithStartupContext(startupCtx),
    engine.WithLifetimeContext(lifetimeCtx),
)
```

## Choosing the right RuntimeServices helper

| Situation | Use |
| --- | --- |
| JS-facing native function calls another JS callback synchronously | `CallWithCurrentContext(vm, op, fn)` |
| JS-facing native function schedules a follow-up on owner | `PostWithCurrentContext(vm, op, fn)` |
| Runtime-owned background work settles a Promise | `PostWithLifetimeContext(op, fn)` or `PostWithCustomContext(callCtx, op, fn)` |
| External request/event has its own context | `CallWithCustomContext(ctx, op, fn)` / `PostWithCustomContext(ctx, op, fn)` |
| Need current callback context only | `runtimebridge.CurrentOwnerContext(vm)` |
| Need runtime lifetime only | `runtimebridge.LifetimeContext(vm)` or `services.Lifetime()` |

`CallWithCustomContext` and `PostWithCustomContext` link custom contexts to runtime lifetime cancellation. `CallWithCustomContext` cancels its linked context after the owner call returns. `PostWithCustomContext` keeps the linked context alive until the posted callback has executed, then unregisters the lifetime cancellation hook.

## Promise-based API pattern

```go
func installSleep(vm *goja.Runtime, exports *goja.Object) {
    runtimeServices, ok := runtimebridge.Lookup(vm)
    if !ok || runtimeServices.Owner == nil {
        panic(vm.NewGoError(fmt.Errorf("timer module requires runtime services")))
    }

    _ = exports.Set("sleep", func(ms int64) goja.Value {
        promise, resolve, reject := vm.NewPromise()
        callCtx := runtimebridge.CurrentOwnerContext(vm)
        runtimeCtx := runtimeServices.Lifetime()

        go func() {
            if ms < 0 {
                _ = runtimeServices.PostWithCustomContext(callCtx, "timer.sleep.reject", func(context.Context, *goja.Runtime) {
                    _ = reject(vm.ToValue("timer.sleep: duration must be >= 0"))
                })
                return
            }

            timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
            defer timer.Stop()
            select {
            case <-callCtx.Done():
                return
            case <-runtimeCtx.Done():
                return
            case <-timer.C:
            }

            _ = runtimeServices.PostWithCustomContext(callCtx, "timer.sleep.resolve", func(context.Context, *goja.Runtime) {
                _ = resolve(goja.Undefined())
            })
        }()

        return vm.ToValue(promise)
    })
}
```

JavaScript usage:

```javascript
const timer = require("timer");

async function example() {
  console.log("Starting...");
  await timer.sleep(1000);
  console.log("Done after 1 second!");
}

example();
```

A fresh runtime still does not expose global `setTimeout`/`setInterval`; the supported primitive is `require("timer").sleep(ms)`.

## Callback and retained UI pattern

For retained callbacks that may be evaluated while a JS-facing native function is already on the owner, use current-context helpers. This avoids scheduling a nested owner call that waits for itself.

```go
tile.BindText(func() string {
    ret, err := runtimeServices.CallWithCurrentContext(vm, "ui.tile.text", func(context.Context, *goja.Runtime) (any, error) {
        value, err := jsTextCallback(goja.Undefined())
        if err != nil {
            return nil, err
        }
        return value.String(), nil
    })
    if err != nil {
        panic(vm.NewGoError(err))
    }
    return ret.(string)
})
```

For hardware or network events that arrive from outside the current JS call stack, use a custom event/request context when available, or the lifetime context for runtime-owned events:

```go
_ = runtimeServices.PostWithLifetimeContext("device.button", func(context.Context, *goja.Runtime) {
    _, err := jsButtonCallback(goja.Undefined())
    if err != nil {
        panic(vm.NewGoError(err))
    }
})
```

## Deadlock safety rule

Do not execute blocking synchronous flows on the owner thread when those flows wait on background goroutines that themselves need to schedule owner-thread callbacks. That creates a circular wait.

In practice:

- keep blocking orchestration off owner when possible;
- route JavaScript value/callback boundaries through `RuntimeServices` helper methods;
- prefer `CallWithCurrentContext` for nested JS-facing callbacks;
- prefer `PostWithCustomContext` / `PostWithLifetimeContext` for background settlement.

## Runtime shutdown

`engine.Runtime.Close(ctx)` cancels the runtime lifetime context, waits briefly for active owner calls to finish, interrupts active JavaScript if necessary, runs registered closers while runtime services are still available, then removes runtimebridge services and stops the owner/event loop.

Native modules with runtime-owned resources should register closers via `RuntimeModuleRegistrationContext.AddCloser` or the xgoja runtime handle closer registry. Closers should expect the lifetime context to already be canceled, but they may still use runtime services for final owner-thread cleanup.

## Connected EventEmitter pattern

For long-lived Go resources that push events into JavaScript, prefer the connected-emitter pattern in `pkg/jsevents`:

1. JavaScript creates a Go-native `EventEmitter` with `require("events")`.
2. JavaScript passes that emitter to a Go-backed helper such as `fswatch.watch(path, emitter)`.
3. Go adopts the emitter on the owner thread and returns a small connection object.
4. Background goroutines never touch `goja.Runtime`, `goja.Value`, or JS callbacks directly; they call `EmitterRef.Emit(...)` instead.
5. The connection's `close()` cancels the Go-side resource without removing JavaScript listeners from the emitter.

Example JavaScript shape:

```javascript
const EventEmitter = require("events");
const watcher = new EventEmitter();
const conn = fswatch.watch("/tmp/demo", watcher, {
  recursive: true,
  debounceMs: 100,
  include: ["**/*.js"],
  exclude: ["**/node_modules/**"]
});

watcher.on("event", (ev) => console.log(ev.relativeName, ev.op, ev.debounced, ev.count));
watcher.on("error", (err) => console.error(err.message));

conn.close();
```

The embedding Go application must explicitly install the manager and helper:

```go
factory, err := engine.NewRuntimeFactoryBuilder().
    WithRuntimeInitializers(
        jsevents.Install(),
        jsevents.FSWatchHelper(jsevents.FSWatchOptions{
            Root: "/tmp/demo",
            AllowRecursive: true,
            MaxDebounce: time.Second,
        }),
    ).
    Build()
```

See `goja-repl help connected-eventemitters-developer-guide` for the full developer guide, including owner-thread safety, typed payload structs, fswatch, and Watermill.
