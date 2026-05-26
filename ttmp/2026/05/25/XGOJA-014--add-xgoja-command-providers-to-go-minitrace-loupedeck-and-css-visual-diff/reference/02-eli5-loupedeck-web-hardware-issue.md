---
Title: ELI5 Loupedeck web hardware issue
Ticket: XGOJA-014
Status: complete
Topics:
    - xgoja
    - providers
    - command-registration
    - goja
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../loupedeck/examples/xgoja/loupedeck-command-provider/Makefile
      Note: |-
        Contains headless smoke, hardware test, and interactive hardware targets
        Headless
    - Path: ../../../../../../../loupedeck/examples/xgoja/loupedeck-command-provider/verbs/web-scene-switcher.js
      Note: |-
        Demo verb being changed to drive both web UI and real Loupedeck UI from shared JS state
        Demo verb with logging and shared web/hardware state
    - Path: ../../../../../../../loupedeck/runtime/js/module_state/module.go
      Note: |-
        State module has the same owner-context pattern as UI and was adjusted similarly
        Likely nested runtime owner context fix for state callbacks
    - Path: ../../../../../../../loupedeck/runtime/js/module_ui/module.go
      Note: |-
        UI module callback binding likely caused nested runtime owner deadlock
        Likely nested runtime owner context fix for UI callbacks
    - Path: ../../../../../../../loupedeck/runtime/js/provider/provider.go
      Note: |-
        xgoja provider now includes UI/state modules and a hardware capability for real deck rendering
        xgoja provider hardware capability and UI/state module registration
ExternalSources: []
Summary: Plain-language explanation of why the loupedeck web/hardware generated demo appeared to hang while configuring the first UI tile.
LastUpdated: 2026-05-26T07:25:00-04:00
WhatFor: Explain the current loupedeck xgoja web+hardware issue, why it is probably not a deck connection problem, and what fix is in progress.
WhenToUse: Read before continuing to debug the loupedeck generated hardware demo.
---


# ELI5: why the Loupedeck web + hardware demo looked stuck

## Goal

Explain, in simple terms, what is going wrong with the generated xgoja Loupedeck demo while we are trying to make one JavaScript verb drive both:

- a browser page served by `express`; and
- a real Loupedeck UI rendered through `require("loupedeck/ui")`.

This is the doc to read before continuing the debug session.

## Context

The target demo is:

```text
/home/manuel/workspaces/2026-05-24/add-js-providers/loupedeck/examples/xgoja/loupedeck-command-provider/verbs/web-scene-switcher.js
```

The desired behavior is:

1. Run the generated binary.
2. It connects to the real Loupedeck.
3. It starts an HTTP server, for example on `http://127.0.0.1:8791/`.
4. The web page and the hardware page share the same JS state.
5. Clicking the web “deal” button updates the hardware tiles/displays.
6. Pressing hardware `Button1` or `Touch1` updates the web state.

## ELI5 version

Think of the JavaScript runtime as a tiny kitchen with exactly **one cook**.

Only that cook is allowed to touch the JavaScript objects. This is normal: Goja JavaScript runtimes are not safe to use from multiple goroutines at the same time.

So we have a helper called the **runtime owner**. Its job is like a kitchen manager:

- If someone outside the kitchen wants JS work done, they ask the manager.
- The manager queues the work for the one cook.
- The caller waits until the cook finishes.

That works great unless the cook is already cooking and then asks the manager:

> “Please queue this new task for me, and I will wait here until it finishes.”

But the cook is the only person who can run queued tasks. So now the cook is waiting for a task that cannot start because the cook is waiting. That is a deadlock.

That is what the Loupedeck UI path appears to be doing.

## What we observed

After adding more logging to `web-scene-switcher.js`, the generated command got this far:

```text
[2026-05-26T11:10:49.902Z] webSceneSwitcher starting {"outDir":".../dist/web-scene","waitMs":1000,"exitOnDeal":true}
[2026-05-26T11:10:49.902Z] creating hardware UI page
[2026-05-26T11:10:49.902Z] configuring tile 0,0
```

Then it hung until the shell `timeout` killed it. The final error was:

```text
Error: runtimeowner jsverbs.invoke: runtime call canceled: context canceled
```

That means the JS verb invocation itself never returned. It was canceled from outside after the timeout.

## Why this is probably not the physical deck's fault

The hang also happened with:

```bash
--deck-enabled=false
```

So this is not primarily “the USB device failed to connect.” The hang happens before we even need real hardware rendering.

The last log line is:

```text
configuring tile 0,0
```

The next JS operation is:

```js
page.tile(0, 0, tile => {
  tile.text(() => scene.get() === "dealt" ? "DEALT" : "WAIT");
});
```

That `tile.text(() => ...)` call installs a reactive text binding. The Loupedeck UI module immediately evaluates that function through the runtime owner.

## The slightly more technical explanation

The generated command-provider path invokes a jsverb through:

```go
registry.InvokeInRuntime(ctx, rt, verb, parsedValues)
```

Inside `go-go-goja/pkg/jsverbs/runtime.go`, that invocation runs on the JS runtime owner:

```go
runtime.Owner.Call(ctx, "jsverbs.invoke", func(...) { ... })
```

So, while `webSceneSwitcher(...)` is executing, we are already on the runtime owner's goroutine.

But `loupedeck/ui` previously captured this context when the module was loaded:

```go
ownerCtx := bindings.Context
```

That is the runtime background context, not necessarily the current owner-call context. Later, inside `tile.BindText`, it calls:

```go
bindings.Owner.Call(ownerCtx, "ui.tile.text", ...)
```

Because `ownerCtx` does not say “we are already on the owner goroutine,” the runtime owner thinks it must queue another call and wait for it. But the only goroutine that can process that queued call is currently blocked inside the first call.

That creates the deadlock.

## The fix in progress

The likely fix is to make `loupedeck/ui` and `loupedeck/state` use the current runtime call context when they are loaded/invoked from inside an owner call:

```go
ownerCtx := runtimebridge.CurrentContext(runtime)
if ownerCtx == nil {
    ownerCtx = bindings.Context
}
```

That context carries the “I am already the owner” marker. Then nested calls like `ui.tile.text` can run directly instead of queuing behind themselves.

This adjustment has already been started in:

- `loupedeck/runtime/js/module_ui/module.go`
- `loupedeck/runtime/js/module_state/module.go`

It still needs a clean validation run.

## What changed around the demo

The demo is being converted from a headless generated smoke into a real hardware/web demo.

### Makefile targets

The `Makefile` now has separate targets:

```bash
make smoke
```

Runs without hardware for automated validation. It uses `--deck-enabled=false`.

```bash
make test-hardware
```

Runs with hardware enabled, posts to `/deal`, exits, and checks output files.

```bash
make hardware
```

Runs with hardware enabled and stays open for manual browser + deck interaction.

### Why `--deck-enabled=false` existed

That flag is only for headless smoke tests. It is not the desired interactive mode.

The real demo should use:

```bash
make hardware
```

or:

```bash
./dist/loupedeck-command-provider loupe web-scene-switcher web-scene-switcher \
  --deck-enabled=true \
  --http-listen 127.0.0.1:8791 \
  --out ./dist/web-scene-hardware \
  --wait-ms 0
```

## Current status

The desired hardware demo is not proven working yet.

Known state:

- The generated buildspec now includes:
  - `loupedeck/state`
  - `loupedeck/ui`
  - `timer`
  - `fs`
  - `express`
- The provider has a hardware capability that can connect to the real deck and wire the retained renderer.
- The JS demo has console/file logging.
- The hang happens while configuring the first UI tile, before hardware-specific behavior is required.
- The likely owner-context fix has been started but not fully validated because the previous test command was interrupted.

## Quick reference

| Symptom | Meaning |
| --- | --- |
| Last log is `configuring tile 0,0` | JS reached `tile.text(() => ...)` and then blocked |
| Error is `runtimeowner jsverbs.invoke: runtime call canceled: context canceled` | The outer JS verb call was still running when the shell timeout canceled it |
| Happens with `--deck-enabled=false` | Not primarily a USB/hardware connection issue |
| Suspected cause | Nested `Owner.Call` using stale background context instead of current owner context |
| Likely fix | Use `runtimebridge.CurrentContext(runtime)` in `loupedeck/ui` and `loupedeck/state` loaders |

## Commands to continue debugging

Build the generated binary:

```bash
cd /home/manuel/workspaces/2026-05-24/add-js-providers/loupedeck/examples/xgoja/loupedeck-command-provider
make build
```

Run headless with logs and a timeout:

```bash
rm -rf dist/web-scene
./dist/loupedeck-command-provider loupe web-scene-switcher web-scene-switcher \
  --deck-enabled=false \
  --http-listen 127.0.0.1:8791 \
  --out ./dist/web-scene \
  --wait-ms 1000 \
  --exit-on-deal=true \
  --log-to-stdout
```

Inspect the file log:

```bash
cat dist/web-scene/scene-debug.log
```

Once the headless UI path no longer hangs, try hardware:

```bash
make test-hardware
```

or interactive:

```bash
make hardware
```

## Review checklist

Before claiming the demo works:

- [ ] `go test ./runtime/js ./runtime/js/provider ./runtime/js/module_state ./runtime/js/module_ui ./cmd/loupedeck/cmds/verbs -count=1` passes.
- [ ] `make smoke` passes with `--deck-enabled=false`.
- [ ] `make test-hardware` passes with the deck attached.
- [ ] `make hardware` keeps the server open and hardware page visible.
- [ ] Web `/deal` updates hardware state.
- [ ] Hardware `Button1` or `Touch1` updates web state.

## Bottom line

The issue is probably not “the deck does not work.”

The issue is more likely: while the JS verb is already running on the single JS owner thread, the Loupedeck UI module asks the same owner thread to synchronously call back into JS using a context that does not identify it as already on that thread. That makes the runtime wait for itself.

The fix is to preserve the current owner-call context in `loupedeck/ui` and `loupedeck/state` so nested reactive UI callbacks can run directly instead of deadlocking.
