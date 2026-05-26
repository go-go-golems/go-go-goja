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
LastUpdated: 2026-05-26T09:15:00-04:00
WhatFor: Onboard an intern to go-go-goja context management and define a safer API direction before adding more mixed runtime packages such as express, discord, and loupedeck.
WhenToUse: Use before modifying runtimebridge, runtimeowner, xgoja provider initialization, async native modules, or event-source integrations.
---


# Goja context ownership and runtime package composition guide

## 1. Executive summary

`go-go-goja` is becoming a runtime composition framework, not just a JavaScript helper library. A generated xgoja binary can now combine packages that open HTTP servers, talk to Discord, listen to hardware, resolve timers, run filesystem work, and expose package-owned Glazed commands. That power changes the context problem. A runtime no longer has a single obvious caller. It can be entered by a CLI command, an HTTP request, a Discord gateway event, a Loupedeck button press, a timer goroutine, or a cleanup hook.

The current system has the right basic pieces: a single owner abstraction protects the `goja.Runtime`; a lifetime context cancels runtime-owned resources; a current-call context stack lets native modules inherit request/command cancellation; and provider capabilities attach runtime-specific resources such as HTTP servers. The problem is that the API names do not clearly tell authors which context they should use in each situation. In particular, the current `runtimebridge.Bindings` type is too generic and the old `Context` field is actively misleading. The proposed replacement is `runtimebridge.RuntimeServices` with an explicit `LifetimeContext` field and no backwards-compatible `Context` alias.

The proper design change is to make context intent explicit. We should distinguish four ideas in the API: runtime lifecycle context, current owner-entry context, captured operation context, and cleanup context. We should add safe helpers for calling/posting to the JS owner, rename or deprecate ambiguous fields, link lifecycle cancellation into every active owner call, harden the runtime owner against same-goroutine reentrant calls even when the caller passes the wrong context, and document module-authoring rules for synchronous callbacks, async promises, subscriptions, and external event sources.

The most important design rule is:

> A native module should not treat `RuntimeServices.LifetimeContext` as “the context for JavaScript callbacks.” It should call with the current owner-entry context while executing inside JS, pass an explicit custom operation/subscription context for external work, or use the lifetime context deliberately for runtime-owned background work.

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

## 3. Conceptual model: serialized VM ownership with multiple entry sources

Goja runtimes are not thread-safe. A `*goja.Runtime` must be accessed by one goroutine at a time. In `go-go-goja`, `runtimeowner.Runner` is the component that serializes VM access. Code outside the owner path does not call methods on the VM directly; it submits work through `Owner.Call` when it needs a result or `Owner.Post` when it only needs to enqueue work.

The complication is that a composed xgoja runtime can be entered by several sources, each with different lifetime and cancellation semantics:

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

Each entry source has a different cancellation source. An HTTP request should stop if the client disconnects. A CLI command should stop if the command context is canceled. A runtime-owned background server should stop when the runtime closes. A cleanup hook should receive a close context and should not depend on an already-canceled request context.

A single `context.Context` cannot represent all of those lifetimes at once. The design must name the differences and specify which context is passed at each boundary.

## 4. Vocabulary: the four contexts we need to name

### 4.1 Runtime lifecycle context

The runtime lifecycle context lives as long as the runtime. It is created when the runtime is created and canceled when `Runtime.Close` runs. In current code, `engine.Factory.NewRuntime` creates it with `context.WithCancel(context.Background())` and stores it in both `engine.Runtime` and `runtimebridge.RuntimeServices.LifetimeContext` (`engine/factory.go:202-217`). `engine.Runtime.Context()` exposes it (`engine/runtime.go:49-55`), and `Runtime.Close` cancels it before running closers and stopping the owner/event loop (`engine/runtime.go:73-113`).

This context is good for runtime-owned resources:

- an HTTP server owned by the runtime;
- a Discord session owned by the runtime;
- a Loupedeck hardware listener owned by the runtime;
- a filesystem watcher owned by the runtime;
- cancellation of goroutines that should not outlive the runtime.

It is not automatically the right context for a JS callback. It says “the runtime still exists,” not “this HTTP request is still alive” or “this CLI invocation is still active.”

### 4.2 Current owner-entry context

The current owner-entry context is the context of the currently executing entry into the VM. `runtimeowner.Runner.Call` receives a context, marks it as an owner context, and executes the callback through `runtimebridge.WithCallContext` (`runtimeowner/runner.go:63-104`, `runtimeowner/runner.go:162-188`). In the current implementation, `runtimebridge.CurrentContext(vm)` returns that active context (`runtimebridge/runtimebridge.go:73-89`). In the proposed cleanup, the public name becomes `runtimebridge.CurrentOwnerContext(vm)`.

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
  ├─ store runtimebridge.RuntimeServices{LifetimeContext, Loop, Owner}
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

#### 5.2.1 What a Runner is

A `Runner` is the runtime-owned serialized execution API for the VM. It is not a general goroutine runner. It is specifically the component that controls when code may touch the `*goja.Runtime`.

Current shape:

```go
type Runner interface {
    Call(ctx context.Context, op string, fn CallFunc) (any, error)
    Post(ctx context.Context, op string, fn PostFunc) error
    Shutdown(context.Context) error
    IsClosed() bool
}
```

The methods have distinct semantics:

- `Call` schedules VM work and waits for a returned value or error. It is used by `xgoja eval`, `xgoja run`, `jsverbs.InvokeInRuntime`, HTTP route dispatch, and REPL evaluation.
- `Post` schedules VM work without waiting for a returned value. It is used by background goroutines that need to settle promises or deliver external events.
- `Shutdown` marks the runner closed so new calls should not be accepted.
- `IsClosed` reports whether new calls are expected to fail.

The runner should eventually also expose shutdown coordination, either directly or through the owning `engine.Runtime`:

```go
type Runner interface {
    Call(ctx context.Context, op string, fn CallFunc) (any, error)
    Post(ctx context.Context, op string, fn PostFunc) error
    Shutdown(context.Context) error
    IsClosed() bool

    // Proposed, or implemented on a private concrete type used by engine.Runtime.
    WaitIdle(ctx context.Context) error
    Interrupt(reason any)
}
```

`WaitIdle` would let `Runtime.Close` wait for active owner entries to unwind after lifecycle cancellation. `Interrupt` would be a last-resort call to `goja.Runtime.Interrupt` when active JavaScript does not cooperate with cancellation.

### 5.3 Runtime bridge path

`runtimebridge` stores per-VM runtime services and current owner-call context.

Current API before cleanup:

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

The cleanup should replace that with `RuntimeServices`, remove `Context`, and expose explicit `LifetimeContext` plus explicit current-context helpers. The problem is not that lifetime context exists; the problem is that the old `Context` name made lifetime context look like a generic callback context.

There is also a subtle global-stack issue. `callContextsByVM` stores a stack per VM, not per goroutine (`runtimebridge/runtimebridge.go:57-71`, `runtimebridge/runtimebridge.go:122-145`). Because only the owner goroutine should call JS-facing module functions, this works for the intended path. But a helper named `CurrentContext(vm)` is tempting to call from background goroutines. If a background goroutine calls it while an HTTP request is active on the owner, it may observe the active request context even though that goroutine is not part of the request. The cleanup removes `CurrentContext` and replaces it with `CurrentOwnerContext` so the name describes the owner-entry scope.

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
- `runtimebridge.CurrentOwnerContext(vm)` is the right concept for JavaScript-facing native functions. Modules such as `database` currently use `CurrentContext` to inherit the active call context for SQL operations; after cleanup that should become `CurrentOwnerContext` (`modules/database/database.go:160-169`).
- Async modules already show the two-context pattern. `timer` and async `fs` capture the active call context for the operation and keep the lifetime context as a shutdown signal (`modules/timer/timer.go:39-64`, `modules/fs/fs_async.go:13-38`).
- xgoja package capabilities are a good place to attach runtime resources. The HTTP provider registers cleanup through `RuntimeCloserRegistry` rather than leaking servers (`pkg/xgoja/providers/http/http.go:94-99`).
- HTTP request handling enters JavaScript with `r.Context()`, which is the correct request-scoped context (`pkg/gojahttp/host.go:79-93`).

The redesign should preserve these strengths. We do not need a new concurrency model. We need clearer vocabulary, safer entry points, and guardrails.

## 8. Current gaps and risks

### 8.1 The old `Bindings.Context` name is ambiguous

The old `Bindings.Context` field is the ambiguous part. It is a runtime lifetime context, but the name does not say that. Because the field sits next to `Owner`, authors naturally write:

```go
bindings.Owner.Call(bindings.Context, "some.callback", fn)
```

That line looks reasonable and can be wrong. The cleanup should remove the field entirely and replace the type with `RuntimeServices{LifetimeContext, Loop, Owner}`.

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

### 9.1 Replace `Bindings` with explicit `RuntimeServices`

`Bindings` is not a good name for the object stored in `runtimebridge`. In this codebase, “bindings” already suggests JavaScript globals, command bindings, REPL bindings, or schema bindings. The object actually contains runtime services that native modules need in order to cooperate with the runtime owner and lifetime.

The proposed replacement is `RuntimeServices`:

```go
type RuntimeServices struct {
    // LifetimeContext is canceled when the runtime lifetime ends. It is not a
    // generic context for callbacks into JavaScript.
    LifetimeContext context.Context

    Loop  *eventloop.EventLoop
    Owner RuntimeOwner
}

func (svc RuntimeServices) Lifetime() context.Context {
    if svc.LifetimeContext != nil {
        return svc.LifetimeContext
    }
    return context.Background()
}
```

There should be no backwards-compatible `Context` field. The old field is a footgun because it makes this line look normal:

```go
services.Owner.Call(services.Context, "callback", fn)
```

The new code should force intent into the name:

```go
services.Owner.Call(services.LifetimeContext, "runtime-owned-callback", fn)
services.CallWithCurrentContext(vm, "js-facing-callback", fn)
services.PostWithCustomContext(eventCtx, "event-callback", fn)
```

The storage helpers can keep their simple names or become explicit:

```go
runtimebridge.Store(vm, runtimebridge.RuntimeServices{...})
runtimebridge.Lookup(vm) (runtimebridge.RuntimeServices, bool)
```

If we want stricter names, use `StoreServices` and `LookupServices`, but that is optional. The important change is the type and field names.

### 9.2 Add owner-call helpers with explicit context names

The common operation “call JS with the current owner-entry context” should be one method, not three manual steps. The method name should say that the context is the important part.

```go
func (svc RuntimeServices) CallWithCurrentContext(
    vm *goja.Runtime,
    op string,
    fn func(context.Context, *goja.Runtime) (any, error),
) (any, error) {
    if svc.Owner == nil {
        return nil, errors.New("runtimebridge: missing owner")
    }
    return svc.Owner.Call(CurrentOwnerContext(vm), op, fn)
}

func (svc RuntimeServices) PostWithCurrentContext(
    vm *goja.Runtime,
    op string,
    fn func(context.Context, *goja.Runtime),
) error {
    if svc.Owner == nil {
        return errors.New("runtimebridge: missing owner")
    }
    return svc.Owner.Post(CurrentOwnerContext(vm), op, fn)
}
```

For runtime-owned work and external event sources, provide names that state the policy:

```go
func (svc RuntimeServices) CallWithLifetimeContext(op string, fn CallFunc) (any, error) {
    return svc.Owner.Call(svc.Lifetime(), op, fn)
}

func (svc RuntimeServices) PostWithLifetimeContext(op string, fn PostFunc) error {
    return svc.Owner.Post(svc.Lifetime(), op, fn)
}

func (svc RuntimeServices) CallWithCustomContext(ctx context.Context, op string, fn CallFunc) (any, error) {
    if ctx == nil {
        ctx = svc.Lifetime()
    }
    return svc.Owner.Call(ctx, op, fn)
}

func (svc RuntimeServices) PostWithCustomContext(ctx context.Context, op string, fn PostFunc) error {
    if ctx == nil {
        ctx = svc.Lifetime()
    }
    return svc.Owner.Post(ctx, op, fn)
}
```

The names are intentionally longer. They make call sites reviewable:

- `CallWithCurrentContext`: code is already inside a JS/native owner entry and wants the active owner-entry context.
- `PostWithCurrentContext`: code is already inside a JS/native owner entry and wants to enqueue follow-up work under that same active context.
- `CallWithLifetimeContext`: work is owned by the runtime lifetime, not by a request or command.
- `PostWithLifetimeContext`: asynchronous work is owned by the runtime lifetime.
- `CallWithCustomContext`: the caller has an explicit operation, request, subscription, or event context.
- `PostWithCustomContext`: same, but fire-and-forget.

### 9.2.1 Make startup and runtime lifetime explicit in `NewRuntime`

Runtime creation should distinguish startup from lifetime. Startup is the bounded operation that creates the VM, registers modules, enables `require`, and runs initializers. Lifetime is the period during which runtime-owned goroutines, servers, subscriptions, and active owner calls should remain valid.

Use runtime options for both:

```go
rt, err := factory.NewRuntime(
    engine.WithStartupContext(startupCtx),
    engine.WithLifetimeContext(lifetimeCtx),
)
```

`WithStartupContext` controls construction and initializer execution. If it is canceled during setup, `NewRuntime` should fail and close partially-created resources.

`WithLifetimeContext` controls runtime-owned work after construction. It should become `Runtime.Context()`, `RuntimeServices.LifetimeContext`, and the lifetime side of linked owner-call contexts. Canceling it should make active calls and runtime-owned goroutines observe shutdown, but the caller should still call `Runtime.Close(closeCtx)` to run cleanup deterministically.

This keeps runtime creation atomic. It avoids a separate `Start(ctx)` phase and therefore avoids a half-created, not-yet-started runtime state.

### 9.3 Make current context owner-goroutine aware

`CurrentOwnerContext(vm)` should be strict. It should only return the dynamic owner-entry context when called from the owner goroutine for that entry. From other goroutines, it should fall back to lifetime context unless the caller passes an explicitly captured context.

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
    return LifetimeContext(vm)
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

This is the defensive runtime fix. Even if a native module author passes the lifecycle context from inside JS, the runner will detect that scheduling would be self-deadlock and invoke directly.

The helper API is still necessary because it preserves the correct cancellation/tracing context. The runner hardening makes mistakes non-fatal.

### 9.5 Defer a formal `EventSourceContext` API

A formal `EventSourceContext` helper is not necessary for the first cleanup pass. The immediate problem is not that event sources lack a struct. The immediate problem is that call sites do not state whether they are using the current owner-entry context, the runtime lifetime context, or a custom context.

For now, event sources should use the explicit helper names:

```go
// HTTP request handler: request-owned work.
services.CallWithCustomContext(r.Context(), "http-handler", fn)

// Discord or hardware event with an explicit event context.
services.PostWithCustomContext(eventCtx, "discord.message", fn)

// Runtime-owned listener with no per-event cancellation.
services.PostWithLifetimeContext("loupedeck.button", fn)
```

A later `EventSourceContext` type may still be useful if Discord, Loupedeck, filesystem watchers, and HTTP sessions converge on shared event metadata. It should be introduced only when real call sites prove that a struct reduces duplication. It is not required to fix the current context semantics.

### 9.6 Link runtime lifecycle cancellation into every owner call

The next design improvement is to make every owner entry observe runtime shutdown automatically. Today, `Owner.Call(ctx, op, fn)` uses the caller's `ctx` as the active owner-entry context. If an HTTP request is active and `Runtime.Close` cancels the runtime lifecycle context, the active call does not necessarily see that cancellation unless the request context is also canceled or the native module explicitly checks both contexts.

The desired invariant is stronger:

> Every owner entry runs under an effective context that is canceled when either the caller context is canceled or the runtime lifecycle context is canceled.

In Go terms, this requires a linked context. Go's standard `context` package has parent-child cancellation but no built-in context that is canceled when either of two unrelated parents is canceled. We can implement that small primitive in `runtimebridge` or `runtimeowner`.

API sketch:

```go
func LinkContexts(a, b context.Context) (context.Context, context.CancelFunc) {
    if a == nil {
        a = context.Background()
    }
    if b == nil {
        b = context.Background()
    }
    ctx, cancel := context.WithCancelCause(context.Background())
    go func() {
        select {
        case <-a.Done():
            cancel(context.Cause(a))
        case <-b.Done():
            cancel(context.Cause(b))
        case <-ctx.Done():
        }
    }()
    return ctx, cancel
}
```

The runner would use it at the owner-entry boundary:

```go
func (r *runner) Call(ctx context.Context, op string, fn CallFunc) (any, error) {
    effectiveCtx, cancel := runtimebridge.LinkContexts(ctx, r.lifecycleCtx)
    defer cancel()
    return r.callWithEffectiveContext(effectiveCtx, op, fn)
}
```

The same applies to `Post`, but with a different cancellation tradeoff. `Post` may return before the scheduled function runs. If the effective context is canceled immediately by `defer cancel()`, the posted work may never run. Therefore `Post` should not create a short-lived linked context that is canceled when `Post` returns. It should either:

- link the supplied context with lifecycle and let the scheduled callback own the cancel function; or
- reject posts when lifecycle is already canceled and otherwise pass a context whose cancellation is driven by the supplied context or lifecycle, not by `Post` returning.

A concrete `Post` shape:

```go
func (r *runner) Post(ctx context.Context, op string, fn PostFunc) error {
    effectiveCtx, cancel := runtimebridge.LinkContexts(ctx, r.lifecycleCtx)
    accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
        defer cancel()
        if effectiveCtx.Err() != nil {
            return
        }
        r.invokePost(r.withOwnerContext(effectiveCtx), op, fn)
    })
    if !accepted {
        cancel()
        return ErrScheduleRejected
    }
    return nil
}
```

This changes the module-authoring model in a useful way. A module that captures `runtimebridge.CurrentOwnerContext(vm)` during an HTTP handler gets a context that is canceled by either request cancellation or runtime close. The module no longer needs to manually combine the request context and lifecycle context for every operation.

Sequence diagram:

```text
HTTP client      gojahttp.Host        runtimeowner.Runner       JS/native code        Runtime.Close
    |                 |                       |                       |                    |
    | GET /route      |                       |                       |                    |
    |---------------->|                       |                       |                    |
    |                 | Owner.Call(r.ctx)     |                       |                    |
    |                 |---------------------->|                       |                    |
    |                 |                       | effective ctx =       |                    |
    |                 |                       | r.ctx OR lifecycle    |                    |
    |                 |                       |                       |                    |
    |                 |                       | run fn(effective ctx) |                    |
    |                 |                       |---------------------->|                    |
    |                 |                       |                       | JS/native running  |
    |                 |                       |                       |                    |
    |                 |                       |                       | Close(closeCtx)    |
    |                 |                       |                       |<-------------------|
    |                 |                       | lifecycle canceled    |                    |
    |                 |                       |---------------------->|                    |
    |                 |                       |                       | ctx.Done closes    |
    |                 |                       |                       | cooperative unwind |
    |                 |                       |<----------------------|                    |
    |                 | response/error        |                       |                    |
    |<----------------|                       |                       |                    |
```

This is cooperative cancellation. It affects code that checks context or waits on operations that check context. It does not preempt a pure JavaScript loop by itself.

### 9.7 Add bounded runtime shutdown with active-call waiting and optional interrupt

Runtime shutdown should become a two-stage process: graceful cancellation first, forced interruption only if graceful shutdown does not complete in time.

The current code cancels the lifecycle context, deletes runtimebridge bindings, runs closers, shuts down the owner, and stops the loop (`engine/runtime.go:73-113`). That ordering can stop the event loop while an active call or package closer might still need owner access. The safer shutdown protocol is:

```text
Runtime.Close(closeCtx)
  1. mark runtime closing so new Owner.Call/Post entries are rejected
  2. cancel runtime lifecycle context
  3. linked active owner contexts observe cancellation
  4. wait for active owner calls to finish, bounded by closeCtx
  5. if calls are still active, call goja.Runtime.Interrupt(reason)
  6. wait briefly again for interrupted JS to unwind
  7. run runtime closers while owner/loop are still available if possible
  8. shutdown owner and stop event loop
  9. delete runtimebridge bindings after no more owner work can run
```

There are two ordering choices to settle during implementation.

First, closers may need owner access. For example, a runtime package might need to unregister JS-visible resources, flush a final event, or resolve/reject pending promises. If the event loop is already stopped, that cleanup cannot run. Therefore closers should run before final loop stop. However, lifecycle cancellation should happen before closers so background goroutines begin exiting.

Second, `runtimebridge.Delete(vm)` should happen only after active owner work and owner-dependent closers are done. Deleting bindings too early means cleanup code cannot look up owner/lifecycle bindings for finalization. This is a change from the current ordering in `engine.Runtime.Close`.

Proposed close pseudocode:

```go
func (r *Runtime) Close(closeCtx context.Context) error {
    r.closeOnce.Do(func() {
        r.markClosing()

        // Cause linked active calls and runtime-owned goroutines to see shutdown.
        r.runtimeCtxCancel()

        // Give cooperative code a chance to leave the VM.
        if err := r.Owner.WaitIdle(closeCtx); err != nil {
            r.VM.Interrupt("runtime closing")
            _ = r.Owner.WaitIdle(shortGraceContext(closeCtx))
        }

        // Run provider/module cleanup while owner and loop still exist.
        retErr = errors.Join(retErr, r.runClosers(closeCtx))

        retErr = errors.Join(retErr, r.Owner.Shutdown(closeCtx))
        r.Loop.Stop()
        runtimebridge.Delete(r.VM)
    })
    return retErr
}
```

`goja.Runtime.Interrupt` should not be the first shutdown mechanism. It should be a fallback for JavaScript that does not cooperatively observe cancellation. The normal path should be context cancellation plus `WaitIdle`.

The shutdown behavior should be documented as:

- Runtime close cancels the lifecycle context.
- Active owner calls see cancellation through their effective current owner context.
- Native modules and promise waiters should return promptly when that context is canceled.
- Pure JavaScript CPU loops require `goja.Runtime.Interrupt` to stop.
- Shutdown must be bounded by the close context.

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
    _ = services.PostWithCustomContext(operationCtx, "module.callback", func(ctx context.Context, vm *goja.Runtime) {
        _, _ = jsCallback(goja.Undefined())
    })
}()
```

### Rule 2: Inside JS-facing functions, use current owner context

A function exported to JS is executing as part of a current owner entry. If it needs to call back into JS synchronously, it should use `CallWithCurrentContext`.

Good:

```go
_ = exports.Set("register", func(call goja.FunctionCall) goja.Value {
    fn, _ := goja.AssertFunction(call.Argument(0))
    result, err := services.CallWithCurrentContext(vm, "module.register.initial", func(ctx context.Context, vm *goja.Runtime) (any, error) {
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
runtimeCtx := services.Lifetime()

promise, resolve, reject := vm.NewPromise()
go func() {
    select {
    case <-callCtx.Done():
        return
    case <-runtimeCtx.Done():
        return
    case result := <-workDone:
        _ = services.PostWithCustomContext(callCtx, "module.resolve", func(context.Context, *goja.Runtime) {
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
subscriptionCtx := services.Lifetime()
sub := source.OnEvent(func(event Event) {
    _ = services.PostWithCustomContext(subscriptionCtx, "source.event", func(ctx context.Context, vm *goja.Runtime) {
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

### 11.1 `runtimebridge.RuntimeServices`

```go
package runtimebridge

type RuntimeServices struct {
    LifetimeContext context.Context
    Loop            *eventloop.EventLoop
    Owner           RuntimeOwner
}

func (svc RuntimeServices) Lifetime() context.Context {
    if svc.LifetimeContext != nil {
        return svc.LifetimeContext
    }
    return context.Background()
}

func (svc RuntimeServices) CallWithCurrentContext(vm *goja.Runtime, op string, fn CallFunc) (any, error) {
    if svc.Owner == nil {
        return nil, errors.New("runtimebridge: missing owner")
    }
    return svc.Owner.Call(CurrentOwnerContext(vm), op, fn)
}

func (svc RuntimeServices) PostWithCurrentContext(vm *goja.Runtime, op string, fn PostFunc) error {
    if svc.Owner == nil {
        return errors.New("runtimebridge: missing owner")
    }
    return svc.Owner.Post(CurrentOwnerContext(vm), op, fn)
}

func (svc RuntimeServices) CallWithLifetimeContext(op string, fn CallFunc) (any, error) {
    return svc.CallWithCustomContext(svc.Lifetime(), op, fn)
}

func (svc RuntimeServices) PostWithLifetimeContext(op string, fn PostFunc) error {
    return svc.PostWithCustomContext(svc.Lifetime(), op, fn)
}

func (svc RuntimeServices) CallWithCustomContext(ctx context.Context, op string, fn CallFunc) (any, error) {
    if svc.Owner == nil {
        return nil, errors.New("runtimebridge: missing owner")
    }
    if ctx == nil {
        ctx = svc.Lifetime()
    }
    return svc.Owner.Call(ctx, op, fn)
}

func (svc RuntimeServices) PostWithCustomContext(ctx context.Context, op string, fn PostFunc) error {
    if svc.Owner == nil {
        return errors.New("runtimebridge: missing owner")
    }
    if ctx == nil {
        ctx = svc.Lifetime()
    }
    return svc.Owner.Post(ctx, op, fn)
}
```

There is no `Context` compatibility field. Aggressive simplification is preferable here because the old name encouraged incorrect owner calls.

### 11.2 Context accessors

```go
func LifetimeContext(vm *goja.Runtime) context.Context
func CurrentOwnerContext(vm *goja.Runtime) context.Context
```

Remove `CurrentContext` rather than keeping it as a compatibility alias. The purpose of this change is to force callers to choose a specific context concept.

### 11.3 Runtime creation options

```go
type RuntimeOption func(*runtimeOptions)

func WithStartupContext(ctx context.Context) RuntimeOption
func WithLifetimeContext(ctx context.Context) RuntimeOption

func (f *Factory) NewRuntime(opts ...RuntimeOption) (*Runtime, error)
```

Default behavior should be explicit in code and docs:

- If `WithStartupContext` is omitted, startup uses `context.Background()`.
- If `WithLifetimeContext` is omitted, lifetime uses `context.Background()`.
- `NewRuntime` derives an internal cancelable lifetime context from the supplied lifetime parent.
- `Runtime.Close(closeCtx)` cancels the internal lifetime context even if the supplied lifetime parent is still active.

Pseudocode:

```go
func (f *Factory) NewRuntime(opts ...RuntimeOption) (*Runtime, error) {
    cfg := runtimeOptions{
        StartupContext:  context.Background(),
        LifetimeContext: context.Background(),
    }
    for _, opt := range opts {
        opt(&cfg)
    }

    runtimeCtx, runtimeCancel := context.WithCancel(cfg.LifetimeContext)

    // Use cfg.StartupContext for construction and initializers.
    // Store runtimeCtx in Runtime and RuntimeServices.
}
```

### 11.4 Runtime owner lifecycle and shutdown coordination

```go
type Runner interface {
    Call(ctx context.Context, op string, fn CallFunc) (any, error)
    Post(ctx context.Context, op string, fn PostFunc) error
    Shutdown(context.Context) error
    IsClosed() bool

    // Proposed. This may stay on the concrete runner type if we do not want to
    // expose it to all package authors.
    WaitIdle(ctx context.Context) error
    Interrupt(reason any)
}
```

`Call` and `Post` should internally link the supplied context with the runtime lifecycle context. `WaitIdle` should let `engine.Runtime.Close` wait for active calls to leave before stopping the event loop. `Interrupt` should call `goja.Runtime.Interrupt` as a last resort after cooperative cancellation and waiting fail.

We may not want to expose owner-goroutine introspection publicly. Same-goroutine reentrancy detection can stay internal to the concrete runner. The public concept should be simpler: use `Call`, `Post`, and the runtime close protocol; do not ask arbitrary code to reason about goroutine identity.

## 12. Implementation plan

### Phase 1: Replace ambiguous context/runtimebridge names

Files:

- `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- `go-go-goja/pkg/runtimebridge/runtimebridge_test.go`
- `go-go-goja/engine/factory.go`

Steps:

1. Rename `runtimebridge.Bindings` to `runtimebridge.RuntimeServices`.
2. Replace `Context` with `LifetimeContext`; do not keep a compatibility field.
3. Add `Lifetime()` method.
4. Update `engine.Factory.NewRuntime` to store `RuntimeServices{LifetimeContext: runtimeCtx, Loop: loop, Owner: owner}`.
5. Add `CallWithCurrentContext`, `PostWithCurrentContext`, `CallWithLifetimeContext`, `PostWithLifetimeContext`, `CallWithCustomContext`, and `PostWithCustomContext` helpers.
6. Add tests proving:
   - `Lifetime()` falls back to `context.Background()` when no lifetime is set.
   - `CallWithCurrentContext` uses the active owner context.
   - `PostWithCustomContext(nil, ...)` falls back to lifetime context.

### Phase 2: Add explicit startup and lifetime runtime options

Files:

- `go-go-goja/engine/factory.go`
- `go-go-goja/engine/runtime.go`
- `go-go-goja/engine/options.go`
- xgoja runtime factory callers

Steps:

1. Add `RuntimeOption`, `WithStartupContext`, and `WithLifetimeContext`.
2. Change runtime creation to use startup context for setup and initializers.
3. Derive runtime lifetime from the supplied lifetime context instead of always using `context.Background()`.
4. Update short-lived command paths to pass command context as both startup and lifetime where appropriate.
5. Update long-lived host paths to pass a separately owned lifetime context when the runtime should outlive a single command/request.
6. Add tests for canceled startup, canceled lifetime, and explicit close.

### Phase 3: Rename concepts in documentation and comments

Files:

- `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- `go-go-goja/pkg/runtimeowner/types.go`
- `go-go-goja/pkg/jsverbs/runtime.go`
- `go-go-goja/pkg/gojahttp/host.go`

Steps:

1. Update comments to distinguish lifecycle, owner-entry, operation, and cleanup contexts.
2. Add a warning to `InvokeInRuntime`: modules invoked from it are already on the owner and should use current owner context for nested JS callbacks.
3. Add a native module authoring guide in docs or help pages.

### Phase 4: Migrate modules away from direct `RuntimeServices.LifetimeContext`

Files to start with:

- `go-go-goja/modules/timer/timer.go`
- `go-go-goja/modules/fs/fs_async.go`
- `go-go-goja/modules/database/database.go`
- `go-go-goja/pkg/gojahttp/host.go`
- downstream `loupedeck/runtime/js/module_ui/module.go`
- downstream `loupedeck/runtime/js/module_state/module.go`

Steps:

1. Replace lifecycle reads with `services.Lifetime()`.
2. Replace nested owner callbacks with `services.CallWithCurrentContext(vm, ...)` when called synchronously from JS-facing code.
3. Replace promise settlements with `services.PostWithCustomContext(capturedCtx, ...)`.
4. Replace runtime-owned external event callbacks with `services.PostWithCustomContext(subscriptionCtx, ...)` or `PostWithLifetimeContext`.

### Phase 5: Harden runtime owner reentrancy

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
func TestRunnerCallFromOwnerWithLifetimeContextDoesNotDeadlock(t *testing.T) {
    // Start outer Owner.Call(commandCtx, ...).
    // Inside it, call Owner.Call(lifecycleCtx, ...).
    // It should run inline and return, not schedule behind itself.
}
```

### Phase 6: Link owner-call contexts with runtime lifetime

Files:

- `go-go-goja/pkg/runtimeowner/runner.go`
- `go-go-goja/pkg/runtimeowner/types.go`
- `go-go-goja/engine/factory.go`
- `go-go-goja/pkg/runtimebridge/runtimebridge.go`

Steps:

1. Add lifecycle context to the runner's options or constructor.
2. Implement a `LinkContexts` helper that returns a context canceled by either caller context or lifecycle context.
3. Use linked context for `Call`.
4. Use linked context carefully for `Post`, with cancellation owned by the scheduled callback rather than by the returning `Post` call.
5. Add tests proving active calls observe lifecycle cancellation.
6. Add tests proving posted callbacks do not receive a context canceled merely because `Post` returned.

### Phase 7: Add bounded shutdown coordination and interrupt fallback

Files:

- `go-go-goja/engine/runtime.go`
- `go-go-goja/pkg/runtimeowner/runner.go`
- `go-go-goja/pkg/runtimeowner/runner_test.go`

Steps:

1. Track active owner entries in the runner.
2. Add `WaitIdle(ctx)` on the concrete runner or public interface.
3. Add `Interrupt(reason)` wrapper around `goja.Runtime.Interrupt`.
4. Change `Runtime.Close` ordering so lifecycle cancellation happens first, active calls are given a bounded chance to finish, interrupt is used only as a fallback, closers run while owner/loop are still available, and runtimebridge bindings are deleted after owner-dependent cleanup.
5. Add tests for cooperative active-call shutdown.
6. Add tests for non-cooperative JavaScript requiring interrupt.
7. Add tests for closers that need owner access.

### Phase 8: Make `CurrentOwnerContext` goroutine-aware

Files:

- `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- possibly a small internal goroutine-id helper package

Steps:

1. Store goroutine id in call-context frames.
2. Return dynamic context only to the goroutine that owns the current frame.
3. Return lifecycle context from background goroutines.
4. Add tests with two goroutines: one owner call holds a context; a background goroutine calling `CurrentOwnerContext(vm)` should see lifecycle context, not the owner entry.

### Phase 9: Add mixed-package integration tests

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
ownerCtx := services.LifetimeContext

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
    result, err := services.CallWithCurrentContext(runtime, "ui.tile.text", func(_ context.Context, vm *goja.Runtime) (any, error) {
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
runtimeCtx := services.Lifetime()
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
listenerCtx := services.Lifetime()

session.OnMessage(func(msg Message) {
    eventCtx := listenerCtx // or a per-event context if the source provides one
    _ = services.PostWithCustomContext(eventCtx, "discord.message", func(ctx context.Context, vm *goja.Runtime) {
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
          ├─ LifetimeContext: context.WithCancel(background)
          ├─ runtimebridge.Store(VM, RuntimeServices{LifetimeContext, Loop, Owner})
          ├─ require registry + selected modules
          └─ runtime initializers
          │
          ▼
*engine.Runtime
          │
          └─ Close(closeCtx)
               ├─ cancel LifetimeContext
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
tile.text calls services.CallWithCurrentContext(vm, "ui.tile.text", ...)
          │
          ├─ current context has owner marker
          └─ runner invokes directly
```

## 15. Test strategy

### 15.1 Unit tests for context names

- `runtimebridge.CurrentOwnerContext` returns lifecycle outside owner calls.
- `runtimebridge.CurrentOwnerContext` returns call context inside owner calls.
- `RuntimeServices.Lifetime()` returns `LifetimeContext` or `context.Background()` when no lifetime is set.
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

The Go implementation should synchronously evaluate the callback during binding. Run it through `jsverbs.InvokeInRuntime`. The test should fail on the old self-deadlock path and pass after `CallWithCurrentContext`/runner hardening.

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

If background code currently calls `CurrentContext(vm)` and relies on seeing an active request context, that code is probably relying on accidental behavior. This cleanup intentionally removes that API and requires explicit helper names, so the behavior change should be covered by tests and release notes.

### Risk: too many helper methods can confuse authors

The API should not expose ten nearly identical ways to call the owner. The minimum useful set is:

- `Lifetime()` for lifecycle.
- `CurrentOwnerContext(vm)` for JS-facing functions.
- `CallWithCurrentContext` / `PostWithCurrentContext` for sync code already in JS.
- `CallWithCustomContext` / `PostWithCustomContext` for explicit external event or operation contexts.

Avoid adding more until real use cases prove they are needed.

### Risk: context cannot solve resource ownership by itself

Context tells goroutines when to stop; it does not define who owns a resource. Runtime packages still need explicit closers, subscription handles, and cleanup order. The xgoja provider capability pattern should remain the resource ownership layer.

## 17. Recommended naming decisions

Use these names consistently:

| Concept | Recommended API name | Avoid |
| --- | --- | --- |
| Runtime lifetime | `LifetimeContext`, `Lifetime()` | `Context` |
| Current JS owner entry | `CurrentOwnerContext(vm)` | `CurrentContext(vm)` as the only name |
| Captured async work | `operationCtx` | `ctx` without qualifier |
| External event callback | `eventCtx` or `listenerCtx` | `CurrentContext(vm)` from event goroutine |
| Cleanup deadline | `closeCtx` | lifecycle context |
| Serialized VM access | `Owner.Call` / `Owner.Post` behind helpers | direct `goja.Runtime` access |

The point of the table is not aesthetics. Good names keep code review honest. A line like this should raise an eyebrow:

```go
bindings.Owner.Call(services.LifetimeContext, "callback", fn)
```

A line like this explains itself:

```go
services.CallWithCurrentContext(vm, "ui.tile.text", fn)
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

1. Should `WithLifetimeContext` cancellation automatically call `Runtime.Close`, or should it only cancel runtime work while callers remain responsible for explicit close? This guide recommends explicit close by default.
2. Should `runtimebridge.Store`/`Lookup` be renamed to `StoreServices`/`LookupServices`, or is the new `RuntimeServices` type name enough?
3. Should owner reentrancy hardening be implemented before helper migration, or after? Doing it first makes the system safer immediately. Doing helpers first makes tests easier to express and documents desired behavior.
4. Should `WaitIdle` and `Interrupt` be public on `runtimeowner.Runner`, or should they remain private methods used only by `engine.Runtime.Close`? Keeping them private reduces API surface; exposing them helps advanced embedders.
5. What should the default graceful shutdown timeout be when callers pass a close context without a deadline? The runtime can require callers to provide one, or it can define a conservative default.
6. Should `Runtime.Close` run closers before or after `WaitIdle`? This guide recommends lifecycle cancel, wait, interrupt if needed, then closers while the loop is still alive, but some resource closers may need to run immediately after lifecycle cancellation to unblock active calls.
7. How should errors from external event callbacks be reported? HTTP has a response path. Hardware and Discord events need logging or event-emitter error channels.
8. Should promise waiting move from polling to a central async job/settlement abstraction? Polling works now, but mixed event packages may benefit from a clearer promise bridge.

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
