---
Title: "Migrating runtime context APIs"
Slug: migrating-runtime-context-api
Short: "How to update go-go-goja native modules and embedders for RuntimeServices, RuntimeOwner, and explicit runtime contexts."
Topics:
- goja
- xgoja
- migration
- runtime
- context
Commands:
- xgoja
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This migration note covers the runtime context API cleanup in `go-go-goja`. The cleanup removes ambiguous names and requires callers to choose whether work belongs to startup, runtime lifetime, the current JavaScript owner entry, or a custom request/event/operation context.

## Runtime creation

Before, embedders passed a single context to `NewRuntime`:

```go
rt, err := factory.NewRuntime(ctx)
```

After, pass explicit runtime options:

```go
rt, err := factory.NewRuntime(
    engine.WithStartupContext(ctx),
    engine.WithLifetimeContext(ctx),
)
```

Use different contexts when startup and runtime lifetime differ:

```go
rt, err := factory.NewRuntime(
    engine.WithStartupContext(startupCtx),
    engine.WithLifetimeContext(lifetimeCtx),
)
```

`WithStartupContext` controls construction and runtime initializers. `WithLifetimeContext` controls runtime-owned resources after construction. `Runtime.Close(closeCtx)` is still the explicit cleanup operation.

`Runtime.Close` cancels the lifetime context, waits briefly for active owner calls to finish, interrupts active JavaScript if needed, runs registered closers while runtime services are still available, then removes runtimebridge services and stops the owner/event loop.

## Runtime owner rename

The owner interface is now named for what it owns:

```go
runtimeowner.RuntimeOwner
```

Use `RuntimeOwner` instead of `Runner` in public API surfaces. Use `runtimeowner.NewRuntimeOwner(...)` instead of `runtimeowner.NewRunner(...)` when embedding a raw VM and scheduler. The owner still provides serialized access to the `*goja.Runtime`:

```go
ret, err := rt.Owner.Call(ctx, "my.operation", func(ctx context.Context, vm *goja.Runtime) (any, error) {
    return vm.RunString("1 + 2")
})
```

## Runtime services rename

Native modules should use `runtimebridge.RuntimeServices` instead of `runtimebridge.Bindings`.

Before:

```go
bindings, ok := runtimebridge.Lookup(vm)
ctx := runtimebridge.CurrentContext(vm)
_ = bindings.Owner.Post(ctx, "module.resolve", fn)
```

After:

```go
services, ok := runtimebridge.Lookup(vm)
ctx := runtimebridge.CurrentOwnerContext(vm)
_ = services.PostWithCustomContext(ctx, "module.resolve", fn)
```

There is no compatibility `Context` field. Use `services.Lifetime()` when work is runtime-owned.

For background work that settles a Promise created during a JS call, usually capture the owner-entry context first, then post settlement with a custom context:

```go
callCtx := runtimebridge.CurrentOwnerContext(vm)
go func() {
    value, err := slowWork()
    if err != nil {
        _ = services.PostWithCustomContext(callCtx, "module.reject", func(context.Context, *goja.Runtime) {
            _ = reject(vm.ToValue(err.Error()))
        })
        return
    }
    _ = services.PostWithCustomContext(callCtx, "module.resolve", func(context.Context, *goja.Runtime) {
        _ = resolve(vm.ToValue(value))
    })
}()
```

## Choosing the right helper

| Situation | Use |
| --- | --- |
| Synchronous callback from JS-facing native code | `services.CallWithCurrentContext(vm, op, fn)` |
| Async follow-up that belongs to the current JS owner entry | `services.PostWithCurrentContext(vm, op, fn)` |
| Runtime-owned background work | `services.CallWithLifetimeContext(op, fn)` or `services.PostWithLifetimeContext(op, fn)` |
| HTTP request, Discord event, hardware event, or other explicit operation context | `services.CallWithCustomContext(ctx, op, fn)` or `services.PostWithCustomContext(ctx, op, fn)` |
| Reading the current owner-entry context | `runtimebridge.CurrentOwnerContext(vm)` |
| Reading runtime lifetime | `runtimebridge.LifetimeContext(vm)` or `services.Lifetime()` |

## Common replacements

```go
// Old
runtimebridge.CurrentContext(vm)

// New
runtimebridge.CurrentOwnerContext(vm)
```

```go
// Old
bindings.Context

// New
services.Lifetime()
```

```go
// Old
bindings.Owner.Post(ctx, "op", fn)

// New
services.PostWithCustomContext(ctx, "op", fn)
```

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `cannot use context.Context as engine.RuntimeOption` | `NewRuntime(ctx)` was replaced. | Use `NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))`. |
| `undefined: runtimebridge.CurrentContext` | The ambiguous helper was removed. | Use `runtimebridge.CurrentOwnerContext(vm)` or `runtimebridge.LifetimeContext(vm)`. |
| `undefined: runtimebridge.Bindings` | Runtime services were renamed. | Use `runtimebridge.RuntimeServices`. |
| Native callback deadlocks during setup | The callback may be entering the owner with lifetime context from inside an owner call. | Use `CallWithCurrentContext` for JS-facing synchronous callbacks. |
| Goroutines accumulate around linked contexts | Older local code linked contexts with a goroutine per call/post. | Use current `RuntimeServices` helpers; they use lifetime cancellation hooks and unregister them after calls/callbacks complete. |

## See also

- `xgoja help tutorial-providing-package-and-modules`
- `xgoja help tutorial-providing-commands`
- `xgoja help xgoja-v2-reference`
