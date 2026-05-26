---
Title: Goja context ownership and runtime package composition guide
Ticket: XGOJA-014
Status: complete
Topics:
    - xgoja
    - providers
    - command-registration
    - goja
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../loupedeck/runtime/js/module_state/module.go
      Note: Downstream state callback context pattern included in migration guidance
    - Path: ../../../../../../../loupedeck/runtime/js/module_ui/module.go
      Note: |-
        Example of how stale context capture in reactive callbacks can deadlock a composed runtime
        Downstream deadlock case used as teaching example
    - Path: engine/factory.go
      Note: |-
        Creates runtimes, event loops, owner runners, lifecycle contexts, and runtimebridge bindings
        Runtime creation path creates owner loop
    - Path: engine/runtime.go
      Note: |-
        Owns runtime shutdown, closers, runtime lifecycle cancellation, and runtimebridge cleanup
        Runtime close path owns lifecycle cancellation
    - Path: modules/timer/timer.go
      Note: |-
        Good example of capturing call and runtime lifecycle contexts for async promise settlement
        Good async promise context pattern analyzed
    - Path: pkg/gojahttp/host.go
      Note: |-
        Bridges HTTP requests into the single JS owner with request-scoped contexts
        HTTP request context enters the runtime owner and is used as a model event source
    - Path: pkg/jsverbs/runtime.go
      Note: |-
        Invokes JS verbs inside caller-owned runtimes and waits for returned promises
        InvokeInRuntime owner-call and promise-wait behavior analyzed
    - Path: pkg/runtimebridge/runtimebridge.go
      Note: |-
        Stores per-runtime bindings and current call contexts; central source of the context confusion
        Central runtime bindings/current-context API reviewed for lifecycle vs current owner context
    - Path: pkg/runtimeowner/runner.go
      Note: |-
        Serializes all VM access through Owner.Call/Post and marks owner contexts
        Owner serialization and context-marker reentrancy implementation reviewed
    - Path: pkg/xgoja/providers/http/http.go
      Note: |-
        Example package capability that starts an external server and attaches runtime cleanup
        Runtime package capability/resource cleanup model analyzed
ExternalSources: []
Summary: Deep design guide for context ownership, runtime owner scheduling, async callbacks, and package composition in go-go-goja/xgoja.
LastUpdated: 2026-05-26T07:55:00-04:00
WhatFor: Onboard an intern to go-go-goja context management and define a safer API direction before adding more mixed runtime packages such as express, discord, and loupedeck.
WhenToUse: Use before modifying runtimebridge, runtimeowner, xgoja provider initialization, async native modules, or event-source integrations.
---


# Goja context ownership and runtime package composition guide

## 1. Executive summary

`go-go-goja` is becoming a runtime composition framework, not just a JavaScript helper library. A generated xgoja binary can now combine packages that open HTTP servers, talk to Discord, listen to hardware, resolve timers, run filesystem work, and expose package-owned Glazed commands. That power changes the context problem. A runtime no longer has a single obvious caller. It can be entered by a CLI command, an HTTP request, a Discord gateway event, a Loupedeck button press, a timer goroutine, or a cleanup hook.

The current system has the right basic pieces: a single owner runner protects the `goja.Runtime`; a lifecycle context cancels runtime-owned resources; a current-call context stack lets native modules inherit request/command cancellation; and provider capabilities attach runtime-specific resources such as HTTP servers. The problem is that the API names do not clearly tell authors which context they should use in each situation. In particular, `runtimebridge.Bindings.Context` looks like “the context to pass to owner calls,” but it is actually the runtime lifecycle context. Passing it into `Owner.Call` from code that is already executing on the owner goroutine can deadlock if the runner only detects reentrancy through the context marker.

The proper design change is to make context intent explicit. We should distinguish four ideas in the API: runtime lifecycle context, current owner-entry context, captured operation context, and cleanup context. We should add safe helpers for calling/posting to the JS owner, rename or deprecate ambiguous fields, harden the runtime owner against same-goroutine reentrant calls even when the caller passes the wrong context, and document module-authoring rules for synchronous callbacks, async promises, subscriptions, and external event sources.

The most important design rule is:

> A native module should not treat `Bindings.Context` as “the context for JavaScript callbacks.” It should either use the current owner-entry context while executing inside JS, capture an operation/subscription context deliberately, or use the lifecycle context deliberately for runtime-owned background work.

## 2. What problem this document solves

This document is a map for a new intern who needs to work on `go-go-goja` without accidentally making context handling worse. It explains the current architecture, why it exists, where the current API is ambiguous, how the Loupedeck issue exposed the ambiguity, and how to redesign the API so mixed runtime packages remain understandable.

The goal is not just to fix one deadlock. The goal is to establish vocabulary and API boundaries that will still hold when a single runtime mixes packages like these:

- `express`, which receives HTTP requests from many goroutines and calls JS route handlers.
- `discord`, which receives gateway events and calls JS bot handlers.
- `loupedeck/ui`, which receives hardware events and reactive render updates.
- `timer`, which resolves promises after sleeps.
- `fs`, which performs work in goroutines and settles promises back on the JS owner.
- `jsverbs`, which invokes command functions and waits for promises.

If we do not clarify context ownership now, every new package will invent its own slightly different rule. Some of those rules will work in isolation and fail when packages are combined.

## 3. Conceptual model: one JavaScript room, many doors

Goja runtimes are not thread-safe. The simplest safe model is to imagine the runtime as a room with one chair and one notebook. Only one person is allowed to sit in the chair and write in the notebook at a time. In `go-go-goja`, the `runtimeowner.Runner` is the door attendant. Every goroutine that wants to touch the VM must ask the attendant to run a function on the owner loop.

The complication is that the room has many doors:

```text
                          ┌───────────────────────────┐
CLI command ─────────────▶│                           │
HTTP request ────────────▶│                           │
Discord event ───────────▶│      runtimeowner.Runner  │────▶ goja.Runtime
Loupedeck event ─────────▶│      Owner.Call/Post      │
Timer goroutine ─────────▶│                           │
FS goroutine ────────────▶│                           │
Cleanup hook ────────────▶│                           │
                          └───────────────────────────┘
```

Each door has different cancellation semantics. An HTTP request should stop if the client disconnects. A CLI command should stop if the user presses Ctrl-C. A runtime-owned background server should stop when the runtime closes. A cleanup hook should receive a close context and should not depend on an already-canceled request context.

A single `context.Context` cannot mean all of those things at once. The design must name the differences.

## 4. Vocabulary: the four contexts we need to name

### 4.1 Runtime lifecycle context

The runtime lifecycle context lives as long as the runtime. It is created when the runtime is created and canceled when `Runtime.Close` runs. In current code, `engine.Factory.NewRuntime` creates it with `context.WithCancel(context.Background())` and stores it in both `engine.Runtime` and `runtimebridge.Bindings.Context` (`engine/factory.go:202-217`). `engine.Runtime.Context()` exposes it (`engine/runtime.go:49-55`), and `Runtime.Close` cancels it before running closers and stopping the owner/event loop (`engine/runtime.go:73-113`).

This context is good for runtime-owned resources:

- an HTTP server owned by the runtime;
- a Discord session owned by the runtime;
- a Loupedeck hardware listener owned by the runtime;
- a filesystem watcher owned by the runtime;
- cancellation of goroutines that should not outlive the runtime.

It is not automatically the right context for a JS callback. It says “the runtime still exists,” not “this HTTP request is still alive” or “this CLI invocation is still active.”

### 4.2 Current owner-entry context

The current owner-entry context is the context of the currently executing entry into the VM. `runtimeowner.Runner.Call` receives a context, marks it as an owner context, and executes the callback through `runtimebridge.WithCallContext` (`runtimeowner/runner.go:63-104`, `runtimeowner/runner.go:162-188`). While that callback is running, `runtimebridge.CurrentContext(vm)` returns that active context (`runtimebridge/runtimebridge.go:73-89`).

Examples:

- `xgoja run` enters the VM with the command context (`pkg/xgoja/app/run.go:102-116`).
- `xgoja eval` enters the VM with the command context (`pkg/xgoja/app/root.go:116-145`).
- `jsverbs.InvokeInRuntime` enters the VM with the command context (`pkg/jsverbs/runtime.go:46-110`).
- `gojahttp.Host.ServeHTTP` enters the VM with `r.Context()` (`pkg/gojahttp/host.go:79-93`).

This context is the right default inside JavaScript-facing native functions. If a module function is called by JS and it needs cancellation, deadlines, trace spans, or same-owner reentrancy, it should start by asking for the current owner-entry context.

### 4.3 Captured operation or subscription context

Some native module functions begin an operation that finishes later. `timer.sleep(ms)` is a good example. The JS function is called now, but the promise resolves later from a goroutine. The timer module captures the current call context while the JS function is executing, captures the lifecycle context separately, and then waits for either cancellation or the timer (`modules/timer/timer.go:31-67`). The filesystem async helpers use the same pattern (`modules/fs/fs_async.go:11-74`).

This captured context is not “current” anymore when the goroutine runs. It is deliberately captured because it describes the operation that created the promise. If the operation belongs to an HTTP request, canceling the request should cancel the operation. If the operation belongs to a CLI command, Ctrl-C should cancel it. The lifecycle context is also checked so runtime shutdown cancels it.

Subscriptions are trickier. A hardware event handler registered during startup may be intended to survive many individual events until the runtime closes. If it captures a short HTTP request context, it may die too early. If it captures only the lifecycle context, it may ignore cancellation of the setup operation. The API should force the module author to choose.

### 4.4 Cleanup context

Cleanup context is passed into `Runtime.Close(ctx)` and runtime closers. It answers a different question: “how long should cleanup be allowed to take?” The HTTP provider registers a runtime closer that calls `server.Shutdown(ctx)` (`pkg/xgoja/providers/http/http.go:94-99`, `pkg/xgoja/providers/http/http.go:155-170`). That `ctx` should come from the close operation, not from the request that happened to start the server.

Cleanup context should not be conflated with current owner-entry context or lifecycle context. The lifecycle context is usually already canceled when cleanup begins.

## 5. Current implementation map

### 5.1 Runtime creation path

The runtime creation path starts in `engine.Factory.NewRuntime`.

```text
Factory.NewRuntime(ctx)
  ├─ create goja.Runtime
  ├─ create goja_nodejs event loop
  ├─ create runtimeowner.Runner
  ├─ create runtime lifecycle context
  ├─ store runtimebridge.Bindings{Context, Loop, Owner}
  ├─ register native modules
  ├─ enable require/console/buffer/url
  ├─ run runtime initializers
  └─ return *engine.Runtime
```

The important file references are:

- `engine/factory.go:190-199` creates the VM, event loop, and owner runner.
- `engine/factory.go:202-217` creates the lifecycle context and stores runtimebridge bindings.
- `engine/factory.go:219-239` registers modules using `RuntimeModuleContext`.
- `engine/factory.go:241-270` enables globals and runs runtime initializers.
- `engine/runtime.go:28-46` defines the owned runtime fields.
- `engine/runtime.go:73-113` defines runtime shutdown and cleanup ordering.

The important design fact is that `NewRuntime(ctx)` currently does not make `ctx` the runtime lifecycle context. It creates a new background-rooted lifecycle context. That is a defensible choice because the runtime may outlive the create call. But it should be documented. If we want runtime creation cancellation and lifecycle cancellation to be related, we need an explicit option, not an accidental reuse.

### 5.2 Owner runner path

The owner runner serializes all VM access.

```text
Owner.Call(ctx, op, fn)
  ├─ normalize ctx
  ├─ optionally apply MaxWait deadline
  ├─ if ctx says current goroutine already owns VM: invoke directly
  ├─ otherwise schedule fn on event loop
  ├─ event-loop callback wraps ctx with owner marker
  ├─ invoke fn through runtimebridge.WithCallContext
  └─ return result/error to caller
```

The current reentrancy check is context-based. `runner.isOwnerContext(ctx)` looks for an owner marker in the context and compares the recorded goroutine id to the current goroutine (`runtimeowner/runner.go:198-207`). That works only if the caller passes the context that contains the marker. If code running on the owner goroutine passes `context.Background()` or the lifecycle context, the runner does not know it is already on the owner goroutine. It schedules behind itself and can deadlock.

That design explains the Loupedeck failure: the UI module called back into JS from within a JS invocation, but it used a context that did not carry the owner marker.

### 5.3 Runtime bridge path

`runtimebridge` stores per-VM bindings and current call context.

Current API:

```go
type Bindings struct {
    Context context.Context
    Loop    *eventloop.EventLoop
    Owner   OwnerRunner
}

func Store(vm *goja.Runtime, bindings Bindings)
func Lookup(vm *goja.Runtime) (Bindings, bool)
func CurrentContext(vm *goja.Runtime) context.Context
func WithCallContext(vm *goja.Runtime, ctx context.Context, fn func() (any, error)) (any, error)
```

The current comments say that `CurrentContext` returns the active owner-call context and falls back to the runtime lifecycle context (`runtimebridge/runtimebridge.go:73-89`). That comment is correct, but the field name `Bindings.Context` is too vague. A reader naturally assumes it is the context to pass to owner calls. It is actually the lifecycle fallback.

There is also a subtle global-stack issue. `callContextsByVM` stores a stack per VM, not per goroutine (`runtimebridge/runtimebridge.go:57-71`, `runtimebridge/runtimebridge.go:122-145`). Because only the owner goroutine should call JS-facing module functions, this works for the intended path. But a helper named `CurrentContext(vm)` is tempting to call from background goroutines. If a background goroutine calls it while an HTTP request is active on the owner, it may observe the active request context even though that goroutine is not part of the request. The API should make that misuse harder.

### 5.4 xgoja runtime factory and module sections

The xgoja app runtime factory creates engine runtimes from named runtime profiles. It turns configured provider modules into engine module specs (`pkg/xgoja/app/factory.go:50-81`). Module descriptors also carry package capabilities so built-in commands can collect Glazed config sections and run runtime initializers (`pkg/xgoja/app/module_sections.go:15-83`).

This is the layer where packages such as HTTP, Discord, and Loupedeck become runtime-relevant. A provider can say:

- “If my module is selected, add these command-line sections.”
- “After the runtime exists and values are parsed, initialize my runtime state.”
- “Register cleanup with the runtime.”

The HTTP provider is the cleanest current example. It exposes an `http` section, decodes `http-enabled` and `http-listen`, stores per-runtime settings, starts the server when `express` is required, and registers a shutdown closer (`pkg/xgoja/providers/http/http.go:62-99`, `pkg/xgoja/providers/http/http.go:102-155`).

### 5.5 JS verbs path

`jsverbs.InvokeInRuntime` is the path used by generated command-provider examples and package-owned command providers. It invokes a JavaScript function in an already-created runtime (`pkg/jsverbs/runtime.go:46-110`). If the function returns a promise, it polls promise state via repeated owner calls (`pkg/jsverbs/runtime.go:227-260`).

This means the JS verb body itself is already executing inside `Owner.Call`. Any native module called synchronously by the verb is already on the owner goroutine. If that native module calls back into JS synchronously, it must preserve the current owner context or rely on a reentrancy-safe owner.

This is not a flaw in `InvokeInRuntime`; it is the correct way to protect the VM. The flaw is that module authors have too many ways to call back into the owner and too few safe defaults.

## 6. The Loupedeck failure as a teaching example

The Loupedeck demo wants one JS file to define shared state and expose it to both web and hardware UI. The simplified shape is:

```js
const state = require("loupedeck/state");
const ui = require("loupedeck/ui");
const express = require("express");

const scene = state.signal("waiting");

ui.page("web-switcher", page => {
  page.tile(0, 0, tile => {
    tile.text(() => scene.get() === "dealt" ? "DEALT" : "WAIT");
  });
});

express.app().post("/deal", () => {
  scene.set("dealt");
});
```

The observed log showed:

```text
webSceneSwitcher starting ...
creating hardware UI page
configuring tile 0,0
```

Then the command timed out with:

```text
Error: runtimeowner jsverbs.invoke: runtime call canceled: context canceled
```

The important clue is that the failure happens even with `--deck-enabled=false`. The real deck is not the first problem. The problem appears while installing a reactive tile text callback. The UI module registers a watcher that immediately evaluates the callback. That callback needs to call JS. If it calls `Owner.Call` with the lifecycle context rather than the active owner-entry context, the owner runner does not recognize that it is already on the owner goroutine.

The deadlock sequence is:

```text
1. jsverbs.InvokeInRuntime calls Owner.Call(commandCtx, "jsverbs.invoke", ...).
2. Owner enters the JS function webSceneSwitcher(...).
3. JS calls ui.page(...), page.tile(...), tile.text(() => ...).
4. tile.text registers a reactive watcher.
5. The watcher immediately asks Owner.Call(lifecycleCtx, "ui.tile.text", ...).
6. lifecycleCtx does not contain the owner marker.
7. Owner schedules ui.tile.text behind the currently running jsverbs.invoke call.
8. jsverbs.invoke cannot finish because it is waiting for tile.text.
9. tile.text cannot start because jsverbs.invoke owns the only JS thread.
```

This is the exact “waiting for yourself” pattern the API should prevent.

## 7. What the current code already does well

Before redesigning, it is worth recognizing what is sound.

- The VM owner abstraction is the right foundation. All external event sources should enter JavaScript through a single serialized path.
- `runtimebridge.CurrentContext(vm)` is the right idea for JavaScript-facing native functions. Modules such as `database` use it to inherit the active call context for SQL operations (`modules/database/database.go:160-169`).
- Async modules already show the two-context pattern. `timer` and async `fs` capture the active call context for the operation and keep the lifecycle context as a shutdown signal (`modules/timer/timer.go:39-64`, `modules/fs/fs_async.go:13-38`).
- xgoja package capabilities are a good place to attach runtime resources. The HTTP provider registers cleanup through `RuntimeCloserRegistry` rather than leaking servers (`pkg/xgoja/providers/http/http.go:94-99`).
- HTTP request handling enters JavaScript with `r.Context()`, which is the correct request-scoped context (`pkg/gojahttp/host.go:79-93`).

The redesign should preserve these strengths. We do not need a new concurrency model. We need clearer vocabulary, safer entry points, and guardrails.

## 8. Current gaps and risks

### 8.1 `Bindings.Context` is ambiguous

`Bindings.Context` is currently the lifecycle context, but the name does not say that. The field sits next to `Bindings.Owner`, so authors naturally write:

```go
bindings.Owner.Call(bindings.Context, "some.callback", fn)
```

That line looks reasonable and can be wrong. The API should not make the wrong line look idiomatic.

### 8.2 Reentrancy depends on passing the right context

`Owner.Call` can invoke directly when it sees an owner marker in the context (`runtimeowner/runner.go:75-77`). If a module passes a stale context, the runner cannot tell it is already on the owner goroutine. A core runtime primitive should be more defensive.

### 8.3 `CurrentContext(vm)` is dynamic but not clearly scoped to the owner goroutine

The current per-VM context stack is conceptually the “current owner call stack.” The API name does not say “owner.” A background goroutine can call it, and the implementation may return an active owner context from another goroutine. That could cause accidental request-context leakage into unrelated background work.

### 8.4 External event source semantics are underspecified

HTTP, Discord, and hardware events all call into JS, but they have different cancellation semantics.

| Event source | Best context for JS handler | Why |
| --- | --- | --- |
| HTTP route | `http.Request.Context()` | Cancel if client disconnects or server times out. |
| CLI command invocation | command context | Cancel if user interrupts command. |
| Timer promise settlement | captured operation context plus lifecycle guard | Cancel if original operation or runtime closes. |
| FS async promise settlement | captured operation context plus lifecycle guard | Same as timer. |
| Discord gateway event | event/session context plus lifecycle guard | Cancel if Discord session/runtime shuts down. |
| Loupedeck hardware event | event/listener context plus lifecycle guard | Cancel if hardware listener/runtime shuts down. |
| Runtime cleanup | close context | Bound cleanup time independently of event/request contexts. |

Without an explicit guide, package authors will copy whichever example is closest, even if the event semantics differ.

### 8.5 Promise waiting is currently polling

`jsverbs.waitForPromise` and `gojahttp.awaitAndFinishPromise` poll promise state with repeated owner calls (`pkg/jsverbs/runtime.go:227-260`, `pkg/gojahttp/host.go:117-145`). This is acceptable for the current prototype, and the code says so, but it matters for context design. Promise waiting should be cancellable by the entry context, and promise settlement should not require a request context after the request is gone.

### 8.6 Runtime initializer context is not the same as runtime lifecycle

`RuntimeInitializerCapability.InitRuntimeFromSections(ctx, vals, handle)` receives the command invocation context that is building/configuring the runtime (`pkg/xgoja/providerapi/capabilities.go:60-66`). The `RuntimeHandle` exposes the concrete `goja.Runtime` and cleanup. That initializer context should be used for setup work, not stored blindly as the lifecycle for future events.

## 9. Proposed API direction

The proper change is to turn context intent into names and methods.

### 9.1 Rename lifecycle context in runtimebridge bindings

Proposed shape:

```go
type Bindings struct {
    // LifecycleContext is canceled when the runtime closes. It is the right
    // context for runtime-owned goroutines and fallback cancellation, not a
    // generic context for JavaScript callbacks.
    LifecycleContext context.Context

    // Deprecated: use LifecycleContext. Context used to mean runtime lifecycle
    // context and should not be passed blindly to Owner.Call/Post.
    Context context.Context

    Loop  *eventloop.EventLoop
    Owner OwnerRunner
}

func (b Bindings) RuntimeContext() context.Context {
    if b.LifecycleContext != nil {
        return b.LifecycleContext
    }
    if b.Context != nil {
        return b.Context
    }
    return context.Background()
}
```

Migration can be soft. First write both fields in `runtimebridge.Store`, update modules to read `RuntimeContext()`, and mark `Context` deprecated. Later remove direct use.

### 9.2 Add owner-call helpers with explicit intent

The common operation “call JS with the current owner-entry context” should be one method, not three manual steps.

```go
func (b Bindings) CallCurrent(
    vm *goja.Runtime,
    op string,
    fn func(context.Context, *goja.Runtime) (any, error),
) (any, error) {
    if b.Owner == nil {
        return nil, errors.New("runtimebridge: missing owner")
    }
    return b.Owner.Call(CurrentOwnerContext(vm), op, fn)
}

func (b Bindings) PostCurrent(
    vm *goja.Runtime,
    op string,
    fn func(context.Context, *goja.Runtime),
) error {
    if b.Owner == nil {
        return errors.New("runtimebridge: missing owner")
    }
    return b.Owner.Post(CurrentOwnerContext(vm), op, fn)
}
```

But “current” should be reserved for owner-thread code. For background event sources, provide lifecycle/event helpers:

```go
func (b Bindings) CallLifecycle(op string, fn CallFunc) (any, error) {
    return b.Owner.Call(b.RuntimeContext(), op, fn)
}

func (b Bindings) PostLifecycle(op string, fn PostFunc) error {
    return b.Owner.Post(b.RuntimeContext(), op, fn)
}

func (b Bindings) PostWithContext(ctx context.Context, op string, fn PostFunc) error {
    if ctx == nil {
        ctx = b.RuntimeContext()
    }
    return b.Owner.Post(ctx, op, fn)
}
```

The naming forces the caller to decide:

- `CallCurrent`: I am inside a JS/native call and want the current owner-entry context.
- `PostWithContext`: I have a captured operation/event context and want to use it.
- `PostLifecycle`: this is runtime-owned background work.

### 9.3 Make current context owner-goroutine aware

Current `CurrentContext(vm)` should become stricter. It should only return the dynamic owner-entry context when called from the owner goroutine for that entry. From other goroutines, it should fall back to lifecycle context unless the caller passes an explicitly captured context.

Conceptual implementation:

```go
type callContextFrame struct {
    ctx context.Context
    ownerGID uint64
}

func CurrentOwnerContext(vm *goja.Runtime) context.Context {
    if frame := topFrame(vm); frame != nil && frame.ownerGID == currentGoroutineID() {
        return frame.ctx
    }
    return LifecycleContext(vm)
}
```

This closes the accidental leak where a background goroutine observes the top owner context from an unrelated HTTP request.

The runtimebridge package currently avoids importing runtimeowner to prevent a cycle. That is fine. The current goroutine id helper can move to a small internal package, or runtimebridge can record whether the context contains a marker by calling a tiny no-cycle interface exposed by runtimeowner. The design goal matters more than the exact package boundary.

### 9.4 Harden `runtimeowner.Runner.Call` against same-goroutine reentrancy

The owner runner should not depend only on the context marker. It should also know whether the current goroutine is actively executing owner work for this runner.

Conceptual fields:

```go
type runner struct {
    vm        *goja.Runtime
    scheduler Scheduler
    opts      Options
    closed    atomic.Bool

    activeOwnerGID atomic.Uint64
}
```

Conceptual flow:

```go
func (r *runner) Call(ctx context.Context, op string, fn CallFunc) (any, error) {
    ctx = normalizeContext(ctx)

    if r.isOwnerContext(ctx) || r.isActiveOwnerGoroutine() {
        ctx = r.ensureOwnerContext(ctx)
        return r.invoke(ctx, op, fn)
    }

    return r.scheduleAndWait(ctx, op, fn)
}
```

The scheduled callback records the owner goroutine while it runs:

```go
accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
    ownerCtx := r.withOwnerContext(ctx)
    r.activeOwnerGID.Store(currentGoroutineID())
    defer r.activeOwnerGID.Store(0)
    v, err := r.invoke(ownerCtx, op, fn)
    resultCh <- callResult{value: v, err: err}
})
```

This is the belt-and-suspenders fix. Even if a native module author passes the lifecycle context from inside JS, the runner will detect that scheduling would be self-deadlock and invoke directly.

The helper API is still necessary because it preserves the correct cancellation/tracing context. The runner hardening makes mistakes non-fatal.

### 9.5 Introduce an event source contract

For runtime packages that introduce external callbacks, add a small guideline and optional helper type.

```go
type EventSourceContext struct {
    Runtime context.Context
    Event   context.Context
}

func (c EventSourceContext) Context() context.Context {
    if c.Event != nil {
        return c.Event
    }
    if c.Runtime != nil {
        return c.Runtime
    }
    return context.Background()
}
```

HTTP creates event contexts from `r.Context()`. Discord creates them from the gateway/session event. Loupedeck creates them from the hardware listener lifecycle or from per-event cancellation if available. Timers and fs operations capture the current owner-entry context at operation creation time.

The key is that each event source declares its policy instead of accidentally inheriting whatever context happens to be on a global stack.

## 10. Proposed module-authoring rules

### Rule 1: Do not call JS directly from background goroutines

Only the owner runner may touch `goja.Runtime`, `goja.Value`, `goja.Promise` settlement functions, or JS callbacks. A background goroutine must use `Owner.Post` or `Owner.Call`.

Bad:

```go
go func() {
    _, _ = jsCallback(goja.Undefined()) // unsafe
}()
```

Good:

```go
go func() {
    _ = bindings.PostWithContext(operationCtx, "module.callback", func(ctx context.Context, vm *goja.Runtime) {
        _, _ = jsCallback(goja.Undefined())
    })
}()
```

### Rule 2: Inside JS-facing functions, use current owner context

A function exported to JS is executing as part of a current owner entry. If it needs to call back into JS synchronously, it should use `CallCurrent`.

Good:

```go
_ = exports.Set("register", func(call goja.FunctionCall) goja.Value {
    fn, _ := goja.AssertFunction(call.Argument(0))
    result, err := bindings.CallCurrent(vm, "module.register.initial", func(ctx context.Context, vm *goja.Runtime) (any, error) {
        value, err := fn(goja.Undefined())
        if err != nil {
            return nil, err
        }
        return value.Export(), nil
    })
    if err != nil {
        panic(vm.NewGoError(err))
    }
    return vm.ToValue(result)
})
```

### Rule 3: For async promises, capture operation context at creation time

The `timer` pattern is the model:

```go
callCtx := runtimebridge.CurrentOwnerContext(vm)
runtimeCtx := bindings.RuntimeContext()

promise, resolve, reject := vm.NewPromise()
go func() {
    select {
    case <-callCtx.Done():
        return
    case <-runtimeCtx.Done():
        return
    case result := <-workDone:
        _ = bindings.PostWithContext(callCtx, "module.resolve", func(context.Context, *goja.Runtime) {
            _ = resolve(vm.ToValue(result))
        })
    }
}()
return vm.ToValue(promise)
```

This says: the operation belongs to the current JS call, but it must also stop when the runtime closes.

### Rule 4: For long-lived subscriptions, choose lifecycle or subscription context deliberately

A subscription registered during startup can be runtime-owned:

```go
subscriptionCtx := bindings.RuntimeContext()
sub := source.OnEvent(func(event Event) {
    _ = bindings.PostWithContext(subscriptionCtx, "source.event", func(ctx context.Context, vm *goja.Runtime) {
        _, _ = handler(goja.Undefined(), vm.ToValue(event))
    })
})
```

A subscription registered during an HTTP request should usually not outlive that request unless the API explicitly says it does. If it should outlive the request, it should not use `r.Context()` silently.

### Rule 5: Runtime initializers configure resources; they should not store command contexts as lifetimes

`InitRuntimeFromSections(ctx, vals, handle)` receives the setup command context. Use it to perform setup that should stop if the command is canceled. Do not store it as the lifetime for future server requests or hardware events. Use the runtime lifecycle context for those resources and register closers through `RuntimeCloserRegistry`.

### Rule 6: Closers use close context

A closer receives the context passed to `Runtime.Close(ctx)`. Use that for shutdown deadlines. Do not use a captured request context for cleanup.

## 11. Proposed API reference for the intern to implement

This is an implementation sketch, not final code.

### 11.1 `runtimebridge.Bindings`

```go
package runtimebridge

type Bindings struct {
    LifecycleContext context.Context

    // Deprecated: use LifecycleContext or RuntimeContext().
    Context context.Context

    Loop  *eventloop.EventLoop
    Owner OwnerRunner
}

func (b Bindings) RuntimeContext() context.Context {
    switch {
    case b.LifecycleContext != nil:
        return b.LifecycleContext
    case b.Context != nil:
        return b.Context
    default:
        return context.Background()
    }
}

func (b Bindings) CallCurrent(vm *goja.Runtime, op string, fn CallFunc) (any, error) {
    if b.Owner == nil {
        return nil, errors.New("runtimebridge: missing owner")
    }
    return b.Owner.Call(CurrentOwnerContext(vm), op, fn)
}

func (b Bindings) PostCurrent(vm *goja.Runtime, op string, fn PostFunc) error {
    if b.Owner == nil {
        return errors.New("runtimebridge: missing owner")
    }
    return b.Owner.Post(CurrentOwnerContext(vm), op, fn)
}

func (b Bindings) CallWithContext(ctx context.Context, op string, fn CallFunc) (any, error) {
    if b.Owner == nil {
        return nil, errors.New("runtimebridge: missing owner")
    }
    if ctx == nil {
        ctx = b.RuntimeContext()
    }
    return b.Owner.Call(ctx, op, fn)
}

func (b Bindings) PostWithContext(ctx context.Context, op string, fn PostFunc) error {
    if b.Owner == nil {
        return errors.New("runtimebridge: missing owner")
    }
    if ctx == nil {
        ctx = b.RuntimeContext()
    }
    return b.Owner.Post(ctx, op, fn)
}
```

### 11.2 Context accessors

```go
func LifecycleContext(vm *goja.Runtime) context.Context
func CurrentOwnerContext(vm *goja.Runtime) context.Context

// Deprecated: use CurrentOwnerContext for JS-facing code or LifecycleContext
// for runtime-owned background work.
func CurrentContext(vm *goja.Runtime) context.Context
```

`CurrentContext` can remain as an alias during migration, but docs should steer new code to the explicit names.

### 11.3 Runner introspection/hardening

```go
type Runner interface {
    Call(ctx context.Context, op string, fn CallFunc) (any, error)
    Post(ctx context.Context, op string, fn PostFunc) error
    Shutdown(context.Context) error
    IsClosed() bool

    // Maybe internal only.
    IsOwnerGoroutine() bool
}
```

We may not want to expose `IsOwnerGoroutine` publicly. The main point is that `Call` and `Post` should use it internally.

## 12. Implementation plan

### Phase 1: Add safe names without behavior changes

Files:

- `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- `go-go-goja/pkg/runtimebridge/runtimebridge_test.go`
- `go-go-goja/engine/factory.go`

Steps:

1. Add `LifecycleContext` to `runtimebridge.Bindings`.
2. Add `RuntimeContext()` method.
3. Update `engine.Factory.NewRuntime` to store both `LifecycleContext: runtimeCtx` and `Context: runtimeCtx` for compatibility.
4. Add `CallCurrent`, `PostCurrent`, `CallWithContext`, and `PostWithContext` helpers.
5. Add tests proving:
   - `RuntimeContext` prefers `LifecycleContext` over deprecated `Context`.
   - `CallCurrent` uses active owner context.
   - `PostWithContext(nil, ...)` falls back to lifecycle context.

### Phase 2: Rename concepts in documentation and comments

Files:

- `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- `go-go-goja/pkg/runtimeowner/types.go`
- `go-go-goja/pkg/jsverbs/runtime.go`
- `go-go-goja/pkg/gojahttp/host.go`

Steps:

1. Update comments to distinguish lifecycle, owner-entry, operation, and cleanup contexts.
2. Add a warning to `InvokeInRuntime`: modules invoked from it are already on the owner and should use current owner context for nested JS callbacks.
3. Add a native module authoring guide in docs or help pages.

### Phase 3: Migrate modules away from direct `Bindings.Context`

Files to start with:

- `go-go-goja/modules/timer/timer.go`
- `go-go-goja/modules/fs/fs_async.go`
- `go-go-goja/modules/database/database.go`
- `go-go-goja/pkg/gojahttp/host.go`
- downstream `loupedeck/runtime/js/module_ui/module.go`
- downstream `loupedeck/runtime/js/module_state/module.go`

Steps:

1. Replace lifecycle reads with `bindings.RuntimeContext()`.
2. Replace nested owner callbacks with `bindings.CallCurrent(vm, ...)` when called synchronously from JS-facing code.
3. Replace promise settlements with `bindings.PostWithContext(capturedCtx, ...)`.
4. Replace runtime-owned external event callbacks with `bindings.PostWithContext(subscriptionCtx, ...)` or `PostLifecycle`.

### Phase 4: Harden runner reentrancy

Files:

- `go-go-goja/pkg/runtimeowner/runner.go`
- `go-go-goja/pkg/runtimeowner/runner_test.go`

Steps:

1. Track active owner goroutine during scheduled callbacks.
2. If `Call` or `Post` is invoked from that goroutine, invoke directly even when the passed context lacks the owner marker.
3. Preserve the best available context:
   - if the caller passed a marked owner context, use it;
   - otherwise wrap the passed context with owner marker;
   - if there is an active runtimebridge current context, consider using it for call stack continuity.
4. Add a regression test that fails under current behavior:

```go
func TestRunnerCallFromOwnerWithLifecycleContextDoesNotDeadlock(t *testing.T) {
    // Start outer Owner.Call(commandCtx, ...).
    // Inside it, call Owner.Call(lifecycleCtx, ...).
    // It should run inline and return, not schedule behind itself.
}
```

### Phase 5: Make `CurrentOwnerContext` goroutine-aware

Files:

- `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- possibly a small internal goroutine-id helper package

Steps:

1. Store goroutine id in call-context frames.
2. Return dynamic context only to the goroutine that owns the current frame.
3. Return lifecycle context from background goroutines.
4. Add tests with two goroutines: one owner call holds a context; a background goroutine calling `CurrentOwnerContext(vm)` should see lifecycle context, not the owner entry.

### Phase 6: Add mixed-package integration tests

Target tests:

1. `jsverbs + timer`: a verb awaits `timer.sleep`; cancellation cancels the wait.
2. `jsverbs + express`: a verb starts HTTP server; request handlers use request contexts.
3. `express + timer`: an HTTP handler awaits a timer; request cancellation cancels the timer.
4. `loupedeck/ui-style reactive callback`: a native module synchronously registers a callback that calls back into JS during an owner call; no deadlock.
5. `external event + runtime close`: a fake hardware/Discord event source posts to JS until runtime close; close cancels future delivery.

## 13. Migration examples

### 13.1 Loupedeck UI reactive binding

Before:

```go
ownerCtx := bindings.Context

tile.BindText(func() string {
    result, err := bindings.Owner.Call(ownerCtx, "ui.tile.text", func(_ context.Context, vm *goja.Runtime) (any, error) {
        value, err := fn(goja.Undefined())
        if err != nil {
            return nil, err
        }
        return stringify(value), nil
    })
    if err != nil {
        panic(runtime.NewGoError(err))
    }
    return result.(string)
})
```

After:

```go
tile.BindText(func() string {
    result, err := bindings.CallCurrent(runtime, "ui.tile.text", func(_ context.Context, vm *goja.Runtime) (any, error) {
        value, err := fn(goja.Undefined())
        if err != nil {
            return nil, err
        }
        return stringify(value), nil
    })
    if err != nil {
        panic(runtime.NewGoError(err))
    }
    return result.(string)
})
```

The after version says exactly what is happening: this callback is evaluating JS in the current owner-entry context if it is already inside one, or with the lifecycle fallback if it is later called from outside.

### 13.2 Timer promise

Before-current code is already close:

```go
callCtx := runtimebridge.CurrentContext(vm)
runtimeCtx := bindings.Context
```

After:

```go
operationCtx := runtimebridge.CurrentOwnerContext(vm)
runtimeCtx := bindings.RuntimeContext()
```

The names tell the story: the operation belongs to the current owner entry, and the runtime context cancels on runtime shutdown.

### 13.3 HTTP route handler

Current code is conceptually right:

```go
ret, err := h.owner.Call(r.Context(), "http-handler", func(ctx context.Context, vm *goja.Runtime) (any, error) {
    result, err := route.Handler(goja.Undefined(), vm.ToValue(req.Map()), res.JSObject(vm))
    ...
})
```

The route handler should run under the HTTP request context. If JavaScript calls `timer.sleep` inside the route, the timer should capture that request context and stop if the client disconnects.

### 13.4 Discord/Loupedeck event source

A Discord or hardware event callback should not call `CurrentContext(vm)` from the event goroutine. It should choose an event context explicitly:

```go
listenerCtx := bindings.RuntimeContext()

session.OnMessage(func(msg Message) {
    eventCtx := listenerCtx // or a per-event context if the source provides one
    _ = bindings.PostWithContext(eventCtx, "discord.message", func(ctx context.Context, vm *goja.Runtime) {
        _, err := handler(goja.Undefined(), vm.ToValue(msg))
        if err != nil {
            // report/log error; do not panic uncontrolled from event goroutine
        }
    })
})
```

## 14. Diagrams

### 14.1 Runtime creation and lifecycle

```text
xgoja command / provider command
          │
          ▼
RuntimeFactory.NewRuntime(profile)
          │
          ▼
engine.Factory.NewRuntime(ctx)
          │
          ├─ VM: *goja.Runtime
          ├─ Loop: eventloop.EventLoop
          ├─ Owner: runtimeowner.Runner
          ├─ LifecycleContext: context.WithCancel(background)
          ├─ runtimebridge.Store(VM, Bindings{LifecycleContext, Loop, Owner})
          ├─ require registry + selected modules
          └─ runtime initializers
          │
          ▼
*engine.Runtime
          │
          └─ Close(closeCtx)
               ├─ cancel LifecycleContext
               ├─ delete runtimebridge bindings
               ├─ run closers in reverse order
               ├─ shutdown Owner
               └─ stop Loop
```

### 14.2 Owner entry and current context

```text
HTTP request context r.Context()
          │
          ▼
Owner.Call(r.Context(), "http-handler", fn)
          │
          ├─ wrap ctx with owner marker
          ├─ push current call context for VM
          ├─ run JS route handler
          │    └─ native module calls runtimebridge.CurrentOwnerContext(vm)
          │         └─ returns r.Context() with owner marker
          └─ pop current call context
```

### 14.3 The deadlock we want to make impossible

```text
Owner goroutine is running jsverbs.invoke
          │
          ▼
JS calls tile.text(() => ...)
          │
          ▼
tile.text calls Owner.Call(lifecycleCtx, "ui.tile.text", ...)
          │
          ├─ lifecycleCtx has no owner marker
          ├─ runner schedules ui.tile.text on owner loop
          └─ caller waits

But owner loop cannot process ui.tile.text because it is still running jsverbs.invoke.
```

The safe helper or runner hardening changes the middle step:

```text
tile.text calls bindings.CallCurrent(vm, "ui.tile.text", ...)
          │
          ├─ current context has owner marker
          └─ runner invokes directly
```

## 15. Test strategy

### 15.1 Unit tests for context names

- `runtimebridge.CurrentOwnerContext` returns lifecycle outside owner calls.
- `runtimebridge.CurrentOwnerContext` returns call context inside owner calls.
- `Bindings.RuntimeContext` prefers `LifecycleContext` and falls back to deprecated `Context`.
- Background goroutines do not accidentally observe another goroutine's owner-entry context.

### 15.2 Unit tests for reentrancy

- Nested `Owner.Call` with current owner context invokes directly.
- Nested `Owner.Call` with lifecycle context invokes directly after hardening.
- Nested `Owner.Post` with lifecycle context does not schedule behind itself.
- Reentrant calls preserve panic recovery behavior.

### 15.3 Native module pattern tests

Create a small test module that exposes:

```js
const m = require("test/reentrant");
m.bind(() => "ok");
```

The Go implementation should synchronously evaluate the callback during binding. Run it through `jsverbs.InvokeInRuntime`. The test should fail on the old self-deadlock path and pass after `CallCurrent`/runner hardening.

### 15.4 Mixed package tests

Use generated or app-level tests for these paths:

- `jsverbs + express + timer`: verb starts Express; HTTP handler awaits timer.
- `jsverbs + express + fs`: route writes a file; close cancels server.
- fake `discord` + express: Discord event updates runtime state, HTTP reads it.
- fake `loupedeck` + express: hardware event updates runtime state, HTTP reads it.

The goal is not to test every package in full. The goal is to test every class of context transition.

## 16. Implementation risks and tradeoffs

### Risk: same-goroutine detection uses goroutine IDs

The current code already uses goroutine IDs in owner context markers (`runtimeowner/runner.go:198-220`). Hardening reentrancy would extend that pattern. Goroutine ID parsing is not a public Go API. The project already accepted this tradeoff for owner context checks. If we want to avoid it, the event loop scheduler would need to expose an owner-thread token or reentrant execution facility.

### Risk: making `CurrentContext` goroutine-aware can change behavior

If background code currently calls `CurrentContext(vm)` and relies on seeing an active request context, that code is probably relying on accidental behavior. Still, this is a behavior change. The migration should add explicit helper names first, then tighten behavior with tests and release notes.

### Risk: too many helper methods can confuse authors

The API should not expose ten nearly identical ways to call the owner. The minimum useful set is:

- `RuntimeContext()` for lifecycle.
- `CurrentOwnerContext(vm)` for JS-facing functions.
- `CallCurrent` / `PostCurrent` for sync code already in JS.
- `CallWithContext` / `PostWithContext` for explicit external event or operation contexts.

Avoid adding more until real use cases prove they are needed.

### Risk: context cannot solve resource ownership by itself

Context tells goroutines when to stop; it does not define who owns a resource. Runtime packages still need explicit closers, subscription handles, and cleanup order. The xgoja provider capability pattern should remain the resource ownership layer.

## 17. Recommended naming decisions

Use these names consistently:

| Concept | Recommended API name | Avoid |
| --- | --- | --- |
| Runtime lifetime | `LifecycleContext`, `RuntimeContext()` | `Context` |
| Current JS owner entry | `CurrentOwnerContext(vm)` | `CurrentContext(vm)` as the only name |
| Captured async work | `operationCtx` | `ctx` without qualifier |
| External event callback | `eventCtx` or `listenerCtx` | `CurrentContext(vm)` from event goroutine |
| Cleanup deadline | `closeCtx` | lifecycle context |
| Serialized VM access | `Owner.Call` / `Owner.Post` behind helpers | direct `goja.Runtime` access |

The point of the table is not aesthetics. Good names keep code review honest. A line like this should raise an eyebrow:

```go
bindings.Owner.Call(bindings.Context, "callback", fn)
```

A line like this explains itself:

```go
bindings.CallCurrent(vm, "ui.tile.text", fn)
```

## 18. Suggested intern task list

1. Read this document end-to-end.
2. Read `pkg/runtimebridge/runtimebridge.go` and identify every public symbol.
3. Read `pkg/runtimeowner/runner.go` and trace `Call`, `Post`, `invoke`, and `withOwnerContext`.
4. Read `engine/factory.go` and `engine/runtime.go` to understand runtime lifecycle creation and close order.
5. Read `modules/timer/timer.go` and `modules/fs/fs_async.go` as good async context examples.
6. Read `pkg/gojahttp/host.go` as the HTTP event-source example.
7. Reproduce the Loupedeck hang or review the ELI5 doc in `reference/02-eli5-loupedeck-web-hardware-issue.md`.
8. Implement Phase 1 safe names and helper methods.
9. Migrate one small module, preferably `timer`, to the new names without behavior change.
10. Add reentrancy tests for `runtimeowner`.
11. Harden `runtimeowner.Call` and `Post`.
12. Migrate Loupedeck UI/state to the helper API.
13. Run focused tests in `go-go-goja` and `loupedeck`.
14. Add a generated xgoja integration test if possible.

## 19. Open questions

1. Should `engine.Factory.NewRuntime(ctx)` optionally derive the runtime lifecycle context from `ctx`, or should lifecycle always be independent? The current implementation uses an independent background-rooted lifecycle. That is often correct for long-lived runtimes, but a short-lived CLI runtime may reasonably want lifecycle cancellation when the command context is canceled.
2. Should `CurrentContext(vm)` be kept as compatibility alias forever, or should it be deprecated in favor of `CurrentOwnerContext(vm)` and `LifecycleContext(vm)`?
3. Should owner reentrancy hardening be implemented before helper migration, or after? Doing it first makes the system safer immediately. Doing helpers first makes tests easier to express and documents desired behavior.
4. How should errors from external event callbacks be reported? HTTP has a response path. Hardware and Discord events need logging or event-emitter error channels.
5. Should promise waiting move from polling to a central async job/settlement abstraction? Polling works now, but mixed event packages may benefit from a clearer promise bridge.

## 20. Bottom line

The system is not fundamentally out of control. It has the right primitive: a single owner runner around the Goja VM. What is getting out of hand is the vocabulary. `context.Context` is carrying too many meanings without enough names.

The design direction is:

- name runtime lifecycle context explicitly;
- name current owner-entry context explicitly;
- capture operation/subscription contexts deliberately;
- keep cleanup context separate;
- provide helper APIs that make correct owner calls easy;
- harden the owner runner so a wrong context does not deadlock the VM;
- test mixed event-source packages as first-class runtime compositions.

Once those rules are encoded in API names and tests, combining `express`, `discord`, `loupedeck`, `timer`, `fs`, and package-owned command providers becomes much less mysterious. Each package can be reviewed by asking one question at every boundary: “Which context is this, and who owns the work it controls?”
