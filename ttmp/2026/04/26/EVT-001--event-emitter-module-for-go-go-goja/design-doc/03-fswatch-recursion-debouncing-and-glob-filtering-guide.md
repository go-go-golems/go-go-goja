---
Title: fswatch recursion debouncing and glob filtering guide
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
    - Path: pkg/doc/03-async-patterns.md
      Note: Connected-emitter docs updated for recursive/debounce/glob options.
    - Path: pkg/doc/16-nodejs-primitives.md
      Note: Node primitives docs updated for fswatch recursive/debounce/glob API.
    - Path: pkg/jsevents/fswatch.go
      Note: |-
        Existing fswatch helper that will be extended with typed structs
        Implemented typed options/payloads
    - Path: pkg/jsevents/fswatch_test.go
      Note: |-
        Existing fswatch tests that will be extended for recursion/debounce/glob behavior.
        Tests for recursive watching
    - Path: pkg/jsverbs/jsverbs_test.go
      Note: |-
        Existing jsverbs integration test that will be extended for recursive/debounce/glob options.
        jsverbs integration test invokes fswatch options in a custom runtime.
    - Path: testdata/jsverbs/fswatch.js
      Note: |-
        Existing jsverbs example that will be extended with new options.
        jsverbs example extended with recursive/debounce/glob options.
ExternalSources: []
Summary: Design and implementation guide for recursive watching, debounced event delivery, and glob filtering in the fswatch connected-emitter helper using typed Go structs.
LastUpdated: 2026-04-26T10:55:00-04:00
WhatFor: Guide implementation and review of recursive fswatch, trailing debounce, and include/exclude glob filtering.
WhenToUse: Use before implementing or reviewing fswatch recursive/debounce/glob behavior in pkg/jsevents.
---



# fswatch recursion debouncing and glob filtering guide

## Executive summary

The current `pkg/jsevents.FSWatchHelper` exposes an opt-in JavaScript helper:

```js
const EventEmitter = require("events");
const emitter = new EventEmitter();
const conn = fswatch.watch("/tmp/project", emitter);
```

This guide extends that helper with three explicit features:

1. **Recursive watching**: watch an existing directory tree and register newly-created subdirectories.
2. **Debouncing**: collapse noisy `fsnotify` bursts for the same path into a trailing event.
3. **Glob filtering**: include or exclude paths before emitting them to JavaScript, and avoid descending ignored directory trees during recursive setup.

The design must keep the existing safety model:

- `fswatch` remains opt-in host access.
- JavaScript creates and owns the `EventEmitter`.
- Go adopts that emitter through `Manager.AdoptEmitterOnOwner(...)`.
- Background goroutines never touch `goja.Runtime`, `goja.Value`, JS callbacks, or JS-owned objects directly.
- Go-to-JS communication uses typed Go structs and typed object builders, not ad-hoc `map[string]any` payloads.

The recommended JavaScript API is:

```js
const conn = fswatch.watch("/tmp/project", emitter, {
  recursive: true,
  debounceMs: 100,
  include: ["**/*.js", "**/*.ts"],
  exclude: ["**/node_modules/**", "**/.git/**"]
});
```

The recommended first implementation supports only trailing debounce and path-only debounce keys. More sophisticated modes can be added later without breaking the API.

## Problem statement

`fsnotify` is intentionally low-level. It watches one file or directory at a time, emits platform-specific event bursts, and does not apply application-level path filtering. JavaScript users usually want a higher-level watcher that can:

- observe a whole project tree;
- ignore noisy folders such as `.git`, `node_modules`, `dist`, or temporary files;
- avoid receiving multiple events for a single editor save;
- receive stable event payloads with lowerCamel JavaScript properties.

The existing helper already provides safe connection and owner-thread emission. This slice should enhance behavior without changing the core connected-emitter pattern.

## Non-goals

Out of scope for this slice:

- polling-based fallback watching;
- leading-edge debounce;
- synthetic startup events for existing files;
- glob negation syntax such as `!pattern`;
- full `minimatch` compatibility;
- symlink-following recursive traversal;
- public `conn.add(path)` / `conn.remove(path)` methods;
- a Node-compatible `fs.watch(...)` adapter.

## JavaScript API

### Options

```ts
interface FSWatchOptionsJS {
  recursive?: boolean;   // default false
  debounceMs?: number;   // default 0, disabled
  include?: string[];    // default [], meaning include all
  exclude?: string[];    // default [], meaning exclude none
}
```

Examples:

```js
fswatch.watch(root, emitter);
fswatch.watch(root, emitter, { recursive: true });
fswatch.watch(root, emitter, { debounceMs: 150 });
fswatch.watch(root, emitter, { recursive: true, include: ["**/*.js"] });
fswatch.watch(root, emitter, { recursive: true, exclude: ["**/.git/**", "**/node_modules/**"] });
```

Invalid inputs should throw synchronously from `fswatch.watch(...)`:

- negative `debounceMs`;
- non-finite `debounceMs`;
- `debounceMs` above host `MaxDebounce` if configured;
- `recursive: true` when host `AllowRecursive` is false;
- invalid glob patterns;
- non-array `include` / `exclude` values.

### Connection object

Extend the current connection object with the resolved per-call options:

```ts
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

### Event payload

Keep the existing event fields and add metadata needed by recursive/debounce/filter behavior:

```ts
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

`count` is the number of raw fsnotify events merged into the emitted event. Without debouncing it is `1`.

### Error payload

```ts
interface FileWatchError {
  source: "fsnotify";
  path: string;
  message: string;
}
```

## Typed Go structs

Do not send free-form maps across the Go/JS boundary. Use typed structs for decoded options, event payloads, error payloads, and connection metadata.

### Host options

```go
type FSWatchOptions struct {
    GlobalName string
    Root string
    AllowPath func(path string) bool

    AllowRecursive bool
    MaxDebounce time.Duration
    IgnorePath func(path string) bool
}
```

`AllowRecursive` defaults to false for conservative host access. Applications that want tree watching opt in explicitly:

```go
jsevents.FSWatchHelper(jsevents.FSWatchOptions{
    Root: "/tmp/sandbox",
    AllowRecursive: true,
    MaxDebounce: 2 * time.Second,
})
```

### Per-call options

```go
type fsWatchCallOptions struct {
    Recursive bool
    Debounce time.Duration
    Include []string
    Exclude []string
}
```

Decode from `goja.Value` into this struct via explicit property reads or `ExportTo` into a JavaScript-facing DTO, then normalize/validate into `fsWatchCallOptions`.

Suggested DTO:

```go
type fsWatchCallOptionsDTO struct {
    Recursive bool `json:"recursive"`
    DebounceMs int64 `json:"debounceMs"`
    Include []string `json:"include"`
    Exclude []string `json:"exclude"`
}
```

Because goja does not automatically use JSON tags when converting Go structs back into JavaScript objects, emitted payloads should use typed object builders that set lowerCamel properties explicitly.

### Event payload struct

```go
type fsWatchEventPayload struct {
    Source string
    WatchPath string
    Name string
    RelativeName string
    Op string
    Create bool
    Write bool
    Remove bool
    Rename bool
    Chmod bool
    Recursive bool
    Debounced bool
    Count int
}

func (p fsWatchEventPayload) ToValue(vm *goja.Runtime) goja.Value {
    obj := vm.NewObject()
    _ = obj.Set("source", p.Source)
    _ = obj.Set("watchPath", p.WatchPath)
    _ = obj.Set("name", p.Name)
    _ = obj.Set("relativeName", p.RelativeName)
    _ = obj.Set("op", p.Op)
    _ = obj.Set("create", p.Create)
    _ = obj.Set("write", p.Write)
    _ = obj.Set("remove", p.Remove)
    _ = obj.Set("rename", p.Rename)
    _ = obj.Set("chmod", p.Chmod)
    _ = obj.Set("recursive", p.Recursive)
    _ = obj.Set("debounced", p.Debounced)
    _ = obj.Set("count", p.Count)
    return obj
}
```

### Error payload struct

```go
type fsWatchErrorPayload struct {
    Source string
    Path string
    Message string
}
```

It should have the same kind of `ToValue(vm)` builder.

### Connection struct

```go
type fsWatchConnection struct {
    Ref *EmitterRef
    Path string
    Options fsWatchCallOptions
}
```

It should build the JS object through a method:

```go
func (c fsWatchConnection) ToValue(vm *goja.Runtime) goja.Value
```

This keeps the Go side typed while still preserving lowerCamel JavaScript properties.

## Glob filtering design

### Pattern semantics

Use Go's standard library where possible. The first implementation should support simple shell-style globs with `*`, `?`, character classes, and recursive `**` path segments.

Recommended behavior:

- Patterns are matched against slash-separated relative paths from `watchPath`.
- Absolute event names are converted with `filepath.Rel(watchPath, event.Name)` and `filepath.ToSlash(...)`.
- If `include` is empty, all paths are included unless excluded.
- If `include` is non-empty, a path must match at least one include pattern.
- If `exclude` matches, the path is excluded even if included.
- Exclude patterns should also be used to skip recursive directory traversal when possible.

### Matcher struct

```go
type fsWatchGlobMatcher struct {
    include []string
    exclude []string
}

func (m fsWatchGlobMatcher) Allows(rel string, isDir bool) bool
func (m fsWatchGlobMatcher) ShouldDescend(rel string) bool
```

### Matching `**`

`filepath.Match` does not implement `**`. Implement a small helper:

```go
func matchGlob(pattern, rel string) bool
```

Suggested semantics:

- Normalize both pattern and rel to slash paths.
- `**` as a full segment matches zero or more path segments.
- Other segments use `path.Match`.

Pseudocode:

```go
func matchGlob(pattern, rel string) bool {
    p := splitSlash(pattern)
    r := splitSlash(rel)
    return matchSegments(p, r)
}
```

This avoids a new dependency for now. If a future slice needs full minimatch semantics, add a dependency deliberately and document it.

## Recursive watching design

### State object

Refactor the current free functions into a state object:

```go
type fsWatchState struct {
    watchPath string
    opts fsWatchCallOptions
    matcher fsWatchGlobMatcher
    watcher *fsnotify.Watcher
    ref *EmitterRef

    mu sync.Mutex
    watchedDirs map[string]struct{}

    debounceMu sync.Mutex
    pending map[string]pendingFSEvent
    timers map[string]*time.Timer
}
```

### Initial setup

- Non-recursive: add only the normalized path.
- Recursive:
  - stat the path;
  - if it is a file, add only the file;
  - if it is a directory, walk it with `filepath.WalkDir`;
  - skip symlink directories;
  - skip directories excluded by the matcher or host `IgnorePath`;
  - add every allowed directory.

### Dynamic directory registration

When a create event arrives in recursive mode:

1. Check whether the created path is a directory with `os.Stat`.
2. If it is a directory, call recursive add on that subtree.
3. Emit runtime errors through typed `fsWatchErrorPayload`.

Use `addRecursive` instead of `addPath` because a moved-in directory may already contain nested subdirectories.

### Directory removal bookkeeping

When a watched directory is removed or renamed, remove it from `watchedDirs` and call `watcher.Remove(path)` best-effort. Do not error for ordinary file paths.

## Debouncing design

### Semantics

First implementation: trailing debounce per path.

- `debounceMs <= 0`: disabled.
- `debounceMs > 0`: merge raw events by cleaned absolute event name.
- Reset timer on each new event for that path.
- When timer fires, emit one event with merged `Op`, `Debounced: true`, and `Count` equal to merged raw event count.
- On explicit close, stop all timers and drop pending events.

### Pending event struct

```go
type pendingFSEvent struct {
    Event fsnotify.Event
    Count int
}
```

Merging:

```go
pending.Event.Op |= event.Op
pending.Count++
```

### Timer cleanup

`fsWatchState.run` must defer cleanup:

```go
defer func() {
    s.stopDebounceTimers()
    _ = s.watcher.Close()
}()
```

`stopDebounceTimers` must stop all timers, clear `pending`, and clear `timers`.

## Implementation plan

### Phase 1: Design and tasks

- Create this guide.
- Add docmgr tasks.
- Commit planning docs.

### Phase 2: Typed structs and option decoding

- Add `fsWatchCallOptionsDTO`.
- Add `fsWatchCallOptions`.
- Add typed payload structs and `ToValue` methods.
- Replace existing `map[string]any` event/error payloads.
- Extend connection object with typed metadata.
- Add tests for invalid options and lowerCamel payload fields.

### Phase 3: fsWatchState refactor

- Introduce `fsWatchState`.
- Move watcher loop into `state.run(ctx)`.
- Move payload construction into `state.eventPayload(...)`.
- Keep behavior equivalent before recursion/debounce/filtering.

### Phase 4: Recursive watching

- Add host `AllowRecursive`.
- Parse JS `recursive`.
- Implement initial recursive walk.
- Skip symlink directories.
- Dynamically add newly-created directories.
- Add recursive tests.

### Phase 5: Glob filtering

- Parse `include` and `exclude` arrays.
- Validate patterns.
- Implement `**` matcher.
- Filter emitted events.
- Skip excluded directories during recursive traversal.
- Add include/exclude tests.

### Phase 6: Debouncing

- Parse and validate `debounceMs`.
- Add `MaxDebounce` host policy.
- Implement pending/timer merge.
- Stop timers on close.
- Add debounce tests.

### Phase 7: jsverbs and docs

- Extend `testdata/jsverbs/fswatch.js` with `recursive`, `debounceMs`, `include`, and `exclude` fields.
- Update jsverbs integration tests.
- Update package docs.
- Update diary/changelog and validate.

## Acceptance criteria

- No fswatch Go-to-JS payload uses `map[string]any`.
- `fswatch.watch(...)` accepts typed options for `recursive`, `debounceMs`, `include`, and `exclude`.
- Recursive mode detects writes in pre-existing nested directories.
- Recursive mode detects writes in newly-created subdirectories.
- Non-recursive mode does not emit nested file events.
- Include globs allow matching events and suppress non-matching events.
- Exclude globs suppress matching events and skip excluded directory trees.
- Debounce mode collapses repeated events for one path into one typed payload with `debounced: true` and `count > 1` when events merge.
- Explicit close cancels watcher resources and stops pending debounce timers.
- jsverbs example demonstrates the new options.
- Validation passes:

```bash
go test ./pkg/jsevents ./pkg/jsverbs -count=1
go test ./pkg/jsevents ./modules/events ./engine ./pkg/jsverbs -count=1
make lint
go test ./... -count=1
```

## Review checklist

- Confirm no background goroutine touches goja values directly.
- Confirm all JS-facing objects are built from typed Go structs.
- Confirm recursive traversal does not follow symlink directories.
- Confirm glob filtering is relative to the watched root.
- Confirm close stops debounce timers.
- Confirm tests avoid exact fsnotify event-count assumptions where platform behavior varies.
