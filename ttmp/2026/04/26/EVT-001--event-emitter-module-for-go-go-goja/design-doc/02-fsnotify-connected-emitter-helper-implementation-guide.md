---
Title: fsnotify connected emitter helper implementation guide
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
    - Path: ../../../../../../../../../../go/pkg/mod/github.com/fsnotify/fsnotify@v1.9.0/fsnotify.go
      Note: External watcher API and non-recursive directory semantics.
    - Path: modules/events/events.go
      Note: Go-native EventEmitter that fswatch adopts from JavaScript.
    - Path: pkg/jsevents/manager.go
      Note: Existing connected-emitter manager that fswatch should reuse.
    - Path: pkg/jsevents/watermill.go
      Note: Existing helper pattern that fswatch should mirror.
ExternalSources: []
Summary: Detailed design and implementation guide for an opt-in fsnotify helper that connects filesystem events to JS-created EventEmitter instances.
LastUpdated: 2026-04-26T10:25:00-04:00
WhatFor: Guide implementation and review of the fsnotify connected-emitter helper in pkg/jsevents.
WhenToUse: Use before implementing fswatch.watch(path, emitter, options?) or reviewing filesystem watcher event delivery into JavaScript.
---


# fsnotify connected emitter helper implementation guide

## Executive summary

The fsnotify feature should add an **opt-in filesystem watcher helper** on top of the Go-native `EventEmitter` and the connected-emitter manager already implemented for EVT-001.

The helper should let embedding Go applications install a JavaScript function such as:

```js
const EventEmitter = require("events");
const watcher = new EventEmitter();
const conn = fswatch.watch("/tmp/demo", watcher);

watcher.on("event", (ev) => {
  console.log(ev.name, ev.op, ev.write);
});

watcher.on("error", (err) => {
  console.error(err.path, err.message);
});

// later
conn.close();
```

The helper must follow the same architecture as the Watermill helper:

- JavaScript creates a Go-native EventEmitter with `new EventEmitter()`.
- JavaScript passes that emitter into a Go-backed helper function.
- Go validates and adopts the emitter with `Manager.AdoptEmitterOnOwner(...)`.
- Go starts an `fsnotify.Watcher` only after JavaScript calls `fswatch.watch(...)`.
- The watcher goroutine never touches `goja.Runtime`, `goja.Value`, the emitter object, or JS listeners directly.
- The goroutine delivers events through `EmitterRef.Emit(...)`, which schedules back onto the runtime owner thread.
- The returned connection object owns lifecycle via `close()`.

This should **not** be part of `modules/events`. The `events` module should remain a pure data-only EventEmitter primitive. Filesystem watching is host access and must be installed explicitly by the embedding Go application.

## Current state this builds on

The following implementation already exists and should be reused:

- `modules/events/events.go`: Go-native EventEmitter constructor and listener methods.
- `events.FromValue(...)`: validates/adopts EventEmitter values passed from JavaScript back into Go.
- `pkg/jsevents/manager.go`: connected-emitter `Manager`, `EmitterRef`, owner-thread `Emit`/`EmitSync`, close lifecycle, and async error handler support.
- `pkg/jsevents/watermill.go`: example opt-in helper shape with `WatermillHelper(...)` and a JS global helper function.

The fsnotify helper should be implemented as another explicit helper in `pkg/jsevents`, likely in:

```text
pkg/jsevents/fswatch.go
pkg/jsevents/fswatch_test.go
```

The relevant external dependency is `github.com/fsnotify/fsnotify`, already present in `go.mod`. Its key contracts:

- `fsnotify.NewWatcher()` creates a watcher.
- `watcher.Add(path)` starts watching one path.
- `watcher.Events` yields filesystem events.
- `watcher.Errors` yields watcher errors.
- Directory watches are **not recursive** by default.
- `fsnotify.Op` is a bitmask; use `ev.Has(fsnotify.Write)` instead of only comparing equality.

## Problem statement and scope

JavaScript scripts need a safe way to react to filesystem changes without direct access to Go goroutines or `fsnotify` channels. Go needs to expose this functionality without violating goja's single-owner runtime rule and without creating ambient host watchers by default.

In scope:

- Add an opt-in `FSWatchHelper(...)` runtime initializer.
- Install a helper object, default global name `fswatch`.
- Expose `fswatch.watch(path, emitter, options?)`.
- Adopt JS-created Go-native EventEmitter instances.
- Start and stop one `fsnotify.Watcher` per connection.
- Emit local `"event"`, `"error"`, and optionally `"close"` events on the provided emitter.
- Provide path policy hooks (`AllowPath`, optional `Root`) so embedding applications can constrain host access.
- Add tests for event delivery, denied paths, invalid emitters, close behavior, and watcher add failures.

Out of scope for the first slice:

- Recursive directory watching.
- Debouncing/coalescing high-volume events.
- Glob pattern expansion.
- Watching multiple paths from one connection.
- Exposing filesystem write/read operations. Existing `fs` module covers file I/O separately and is opt-in host access.
- Putting `fswatch` into `DefaultRegistryModules()` or `modules/events`.

## JavaScript API contract

### Basic usage

```js
const EventEmitter = require("events");
const watcher = new EventEmitter();
const conn = fswatch.watch("/tmp/demo", watcher);

watcher.on("event", (ev) => {
  if (ev.write) {
    console.log("written", ev.name);
  }
});

watcher.on("error", (err) => {
  console.error(err.source, err.path, err.message);
});

watcher.on("close", () => {
  console.log("watcher closed");
});

conn.close();
```

### `fswatch.watch(path, emitter, options?)`

Parameters:

| Parameter | Type | Meaning |
|---|---|---|
| `path` | string | File or directory path to pass to `watcher.Add`. |
| `emitter` | `EventEmitter` | JS-created Go-native EventEmitter that receives watcher events. |
| `options` | object, optional | Reserved for future options. First slice may accept but ignore or support `recursive: false` only. |

Return value:

```ts
interface FSWatchConnection {
  id: string;
  path: string;
  close(): boolean;
}
```

`close()` should be idempotent. It should cancel the Go watcher context, close OS watcher resources, unregister the `EmitterRef`, and return `true` if cleanup was scheduled/accepted.

### Event names

The helper emits local events on the provided emitter:

| Event | Payload | When |
|---|---|---|
| `event` | `FileWatchEvent` | A filesystem notification arrives from `watcher.Events`. |
| `error` | `FileWatchError` | `NewWatcher`, `Add`, or `watcher.Errors` reports an error. |
| `close` | no payload or close payload | Watcher exits because channels closed or explicit close completes. |

Use local event names (`"event"`, `"error"`, `"close"`) rather than global names like `"fs:event"`. The emitter itself represents the resource connection.

### Event payload shape

Recommended filesystem event payload:

```ts
interface FileWatchEvent {
  source: "fsnotify";
  watchPath: string;
  name: string;
  op: string;
  create: boolean;
  write: boolean;
  remove: boolean;
  rename: boolean;
  chmod: boolean;
}
```

`op` should use `ev.Op.String()` for a human-readable representation. The boolean fields should use `ev.Has(...)` so scripts can handle bitmask combinations without parsing strings.

Recommended error payload:

```ts
interface FileWatchError {
  source: "fsnotify";
  path: string;
  message: string;
}
```

Recommended close payload, if any:

```ts
interface FileWatchClose {
  source: "fsnotify";
  path: string;
  reason: "closed" | "canceled" | "error";
}
```

For the first slice, `close` may emit no payload to match many EventEmitter conventions. If a payload is added, tests should assert it.

## Go API contract

Add this to `pkg/jsevents`:

```go
type FSWatchOptions struct {
    // GlobalName is the JavaScript helper object name. Default: "fswatch".
    GlobalName string

    // Root optionally restricts relative and absolute paths to a subtree.
    // If set, watch paths should be resolved/cleaned against this root.
    Root string

    // AllowPath decides whether the final path may be watched.
    // If nil, all paths are allowed after Root normalization.
    AllowPath func(path string) bool
}

func FSWatchHelper(opts FSWatchOptions) engine.RuntimeInitializer
```

A helper initializer mirrors `WatermillHelper(...)`:

- Validate that `jsevents.Install(...)` already installed a `Manager`.
- Create a JS object with `watch` method.
- Set it as `globalThis[GlobalName]`, defaulting to `fswatch`.
- Do no filesystem work during initialization.

## Path policy

Filesystem watching is host access. Do not expose it without a policy in applications that run untrusted scripts.

Recommended normalization behavior:

1. Convert JS argument to string.
2. Reject empty paths.
3. If `Root` is set:
   - if the JS path is relative, join it with `Root`;
   - clean the final path;
   - reject paths that escape `Root`.
4. If `Root` is not set:
   - clean the path;
   - optionally convert to absolute path for stable policy checks.
5. Call `AllowPath(finalPath)` if non-nil.
6. Use the final path for `watcher.Add(...)` and for payload `watchPath`.

Pseudocode:

```go
func normalizeWatchPath(raw string, opts FSWatchOptions) (string, error) {
    raw = strings.TrimSpace(raw)
    if raw == "" {
        return "", fmt.Errorf("fswatch: path is empty")
    }

    if opts.Root != "" {
        root, err := filepath.Abs(opts.Root)
        if err != nil { return "", err }

        var candidate string
        if filepath.IsAbs(raw) {
            candidate = filepath.Clean(raw)
        } else {
            candidate = filepath.Join(root, raw)
        }
        candidate, err = filepath.Abs(candidate)
        if err != nil { return "", err }

        rel, err := filepath.Rel(root, candidate)
        if err != nil { return "", err }
        if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
            return "", fmt.Errorf("fswatch: path escapes root")
        }
        raw = candidate
    } else {
        raw = filepath.Clean(raw)
    }

    if opts.AllowPath != nil && !opts.AllowPath(raw) {
        return "", fmt.Errorf("fswatch: path %q is not allowed", raw)
    }
    return raw, nil
}
```

## Implementation plan

### Phase 1: Add helper skeleton

Files:

```text
pkg/jsevents/fswatch.go
pkg/jsevents/fswatch_test.go
```

Implement:

- `FSWatchOptions`.
- `FSWatchHelper(opts FSWatchOptions) engine.RuntimeInitializer`.
- `fsWatchHelper.ID()`.
- `fsWatchHelper.InitRuntime(...)`.
- `globalName` defaulting to `fswatch`.
- Manager lookup with a clear error if `jsevents.Install()` is missing.

Initializer pseudocode:

```go
func (h *fsWatchHelper) InitRuntime(ctx *engine.RuntimeContext) error {
    if ctx == nil || ctx.VM == nil {
        return fmt.Errorf("jsevents fswatch: incomplete runtime context")
    }
    manager, err := managerFromContext(ctx)
    if err != nil { return err }

    obj := ctx.VM.NewObject()
    if err := obj.Set("watch", h.watchFunc(ctx, manager)); err != nil {
        return err
    }
    return ctx.VM.Set(globalName(h.opts.GlobalName), obj)
}
```

### Phase 2: Add `watch(path, emitter, options?)`

Inside the JS-callable function:

1. Decode and normalize the path.
2. Adopt `call.Argument(1)` as a Go-native EventEmitter.
3. Create `watchCtx, cancel := context.WithCancel(ctx.Context)`.
4. Call `ref.SetCancel(cancel)`.
5. Start watcher goroutine.
6. Return connection object.

Pseudocode:

```go
func (h *fsWatchHelper) watch(call goja.FunctionCall) goja.Value {
    path, err := normalizeWatchPath(call.Argument(0).String(), h.opts)
    if err != nil { panic(vm.NewGoError(err)) }

    ref, err := manager.AdoptEmitterOnOwner(call.Argument(1))
    if err != nil { panic(vm.NewGoError(err)) }

    watchCtx, cancel := context.WithCancel(ctx.Context)
    ref.SetCancel(cancel)

    go runFSWatcher(watchCtx, ref, path)

    return fsWatchConnectionObject(vm, ref, path)
}
```

Connection object:

```go
func fsWatchConnectionObject(vm *goja.Runtime, ref *EmitterRef, path string) *goja.Object {
    obj := vm.NewObject()
    _ = obj.Set("id", ref.ID())
    _ = obj.Set("path", path)
    _ = obj.Set("close", func() bool {
        return ref.Close(context.Background()) == nil
    })
    return obj
}
```

### Phase 3: Implement watcher loop

Pseudocode:

```go
func runFSWatcher(ctx context.Context, ref *EmitterRef, path string) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        _ = emitFSWatchError(ctx, ref, path, err)
        _ = ref.Close(ctx)
        return
    }
    defer watcher.Close()

    if err := watcher.Add(path); err != nil {
        _ = emitFSWatchError(ctx, ref, path, err)
        _ = ref.Close(ctx)
        return
    }

    for {
        select {
        case <-ctx.Done():
            _ = ref.Emit(context.Background(), "close")
            return
        case ev, ok := <-watcher.Events:
            if !ok {
                _ = ref.Emit(context.Background(), "close")
                _ = ref.Close(context.Background())
                return
            }
            _ = ref.Emit(ctx, "event", fsWatchEventPayload(path, ev))
        case err, ok := <-watcher.Errors:
            if !ok {
                _ = ref.Emit(context.Background(), "close")
                _ = ref.Close(context.Background())
                return
            }
            _ = emitFSWatchError(ctx, ref, path, err)
        }
    }
}
```

Important detail: if `ctx` is canceled, `ref.Emit(ctx, "close")` may reject because `runtimeowner.Post` checks context cancellation. If close delivery matters, use `context.Background()` for the final close event or skip close event on explicit cancellation. The first implementation should choose and test one behavior.

Recommended first-slice behavior:

- Explicit `conn.close()` cancels and cleans up; do not guarantee a JS `close` event after explicit close.
- Channel closure emits `close` with `context.Background()` best-effort.
- Add failure emits `error` before close because setup is still active.

### Phase 4: Payload helpers

```go
func fsWatchEventPayload(watchPath string, ev fsnotify.Event) map[string]any {
    return map[string]any{
        "source": "fsnotify",
        "watchPath": watchPath,
        "name": ev.Name,
        "op": ev.Op.String(),
        "create": ev.Has(fsnotify.Create),
        "write": ev.Has(fsnotify.Write),
        "remove": ev.Has(fsnotify.Remove),
        "rename": ev.Has(fsnotify.Rename),
        "chmod": ev.Has(fsnotify.Chmod),
    }
}

func emitFSWatchError(ctx context.Context, ref *EmitterRef, path string, err error) error {
    return ref.Emit(ctx, "error", map[string]any{
        "source": "fsnotify",
        "path": path,
        "message": err.Error(),
    })
}
```

### Phase 5: Tests

#### Test 1: missing manager fails at runtime creation

```go
factory, err := engine.NewBuilder().
    WithRuntimeInitializers(jsevents.FSWatchHelper(jsevents.FSWatchOptions{})).
    Build()
require.NoError(t, err)
_, err = factory.NewRuntime(context.Background())
require.ErrorContains(t, err, "manager is not installed")
```

#### Test 2: invalid emitter throws

```js
fswatch.watch(tempDir, {});
```

Expect a Go/JS error containing `EventEmitter`.

#### Test 3: denied path throws

Install with:

```go
AllowPath: func(path string) bool { return false }
```

Expect `not allowed`.

#### Test 4: add failure emits error

Call `fswatch.watch(nonExistentPath, emitter)` with an error listener registered. Because `watcher.Add` happens in the goroutine, the JS call may return a connection before the error arrives. Test with `require.Eventually` against a JS `errors` array.

```js
const emitter = new EventEmitter();
globalThis.errors = [];
emitter.on("error", err => errors.push(err.message));
fswatch.watch("/definitely/missing", emitter);
```

#### Test 5: file event delivery

Use `t.TempDir()`:

1. Register `event` listener that pushes events to `globalThis.events`.
2. Call `fswatch.watch(tempDir, emitter)`.
3. Write a file under the directory.
4. `require.Eventually` until `events` includes the filename.

Caveat: fsnotify timing varies by platform. Tests should allow enough time, avoid asserting exact operation count, and accept either create/write as long as the event name matches.

#### Test 6: close stops delivery

1. Start watcher.
2. Call `conn.close()` from JS or through owner thread.
3. Write a file.
4. Assert no new events after a short stability window.

This test can be flaky if an event was already queued before close. Make it robust by closing before writing, then waiting for the event count to remain unchanged.

### Phase 6: Documentation and examples

Update:

- `pkg/doc/03-async-patterns.md`: add a short connected-emitter fswatch example.
- `pkg/doc/16-nodejs-primitives.md`: mention `fswatch` only as an opt-in helper in `pkg/jsevents`, not as a default primitive.
- Optional example script under ticket `scripts/` or `testdata/jsverbs` if an embedding runtime installs the helper.

Do not add `fswatch` to `modules.DefaultRegistry`. It is host access and requires application policy.

## Design decisions

### Decision 1: JS passes the emitter, Go adopts it

This keeps EventEmitter ownership explicit. The JS code decides which emitter receives events, and Go only keeps an `EmitterRef` handle.

### Decision 2: local event names, not global namespaced events

Use `"event"`, `"error"`, and `"close"` on the returned/adopted emitter. The emitter object already scopes the events to one watcher.

### Decision 3: path policy is part of helper configuration

Filesystem watching is host access. `AllowPath` and `Root` are part of the helper API so embedders can constrain what scripts watch.

### Decision 4: no recursive watching in the first slice

`fsnotify` does not recursively watch directories by default. Recursive watch support requires directory walking, handling new subdirectories, and cleanup complexity. Add it later behind an explicit option.

### Decision 5: close is connection lifecycle, not EventEmitter lifecycle

`conn.close()` should stop the Go watcher and unregister the connection. It should not remove JS listeners from the emitter. The emitter may still be used by JS for other events if the script wants.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Flaky fsnotify tests | Use `require.Eventually`, avoid exact operation count, use temp dirs. |
| Host path exposure | Require explicit helper installation and support `Root`/`AllowPath`. |
| Goroutine leaks | Per-connection context, `ref.SetCancel(cancel)`, `defer watcher.Close()`, close tests. |
| JS runtime race | Only call `ref.Emit`, never use goja directly in watcher goroutine. |
| Events after close | Make close idempotent and tolerate already-queued fsnotify events in tests. |
| Recursive watch surprise | Document non-recursive behavior clearly. |

## Acceptance criteria

The fsnotify helper slice is complete when:

- `pkg/jsevents/fswatch.go` exists.
- `FSWatchHelper(...)` installs `fswatch.watch(...)` only when explicitly configured.
- A JS-created `EventEmitter` can receive file events from a watched temp directory.
- Denied paths and invalid emitters fail clearly.
- Watcher add failures surface as `error` events.
- `conn.close()` cancels watcher resources.
- Tests pass:

```bash
go test ./pkg/jsevents -run 'TestFSWatch|TestManager|TestWatermill' -count=1
go test ./pkg/jsevents ./modules/events ./engine ./pkg/jsverbs -count=1
make lint
go test ./... -count=1
```

## Implementation checklist

1. Add `pkg/jsevents/fswatch.go`.
2. Define `FSWatchOptions` and `FSWatchHelper`.
3. Add manager lookup and default global name handling.
4. Implement path normalization and policy checks.
5. Implement JS `watch(path, emitter, options?)` function.
6. Implement connection object with `id`, `path`, and `close()`.
7. Implement watcher goroutine and payload helpers.
8. Add tests for missing manager, invalid emitter, denied path, add failure, file event delivery, and close.
9. Add docs/example snippets.
10. Run validation and update diary/changelog.
11. Commit the fsnotify helper slice.
