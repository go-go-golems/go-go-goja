# Tasks

## Done

- [x] Map the current goja runtime, module registry, and JS verb pipeline
- [x] Define the proposed JS sandbox API shape and host responsibilities
- [x] Write the detailed sandbox host architecture and implementation guide
- [x] Write the API reference with example bot scripts
- [x] Keep a detailed diary for the ticket

## Next

### Phase 1: Runtime module and state wiring
- [x] Add `modules/sandbox` with a runtime-scoped module loader and a per-runtime host state object.
- [x] Register the module through `engine/runtime.go` so `DefaultRegistryModules()` exposes `require("sandbox")` automatically.
- [x] Add a host-facing `Config`/`Registrar` API in `pkg/sandbox` that keeps the module name stable and stores the runtime state in `engine.RuntimeModuleContext.Values` for host-side access.

### Phase 2: Bot definition and dispatch semantics
- [x] Implement `defineBot(builderFn)` with the `command`, `event`, and `configure` helpers documented in the design guide.
- [x] Add a `BotHandle` abstraction that can dispatch command/event handlers by name from Go.
- [x] Add a small JS-facing context builder that provides `ctx.args`, `ctx.command`, `ctx.user`, `ctx.guild`, `ctx.channel`, `ctx.me`, `ctx.reply`, `ctx.defer`, `ctx.log`, and `ctx.store`.

### Phase 3: In-memory state
- [x] Add a concurrency-safe in-memory store with `get`, `set`, `delete`, `keys`, and `namespace` support.
- [x] Ensure store access is runtime-local, process-local, and scoped to the current sandbox runtime.
- [x] Keep the storage API synchronous in v1 while still returning normal JS values.

### Phase 4: Tests and smoke checks
- [x] Add runtime tests for `defineBot`, `command`, `event`, `configure`, and `store` behavior.
- [x] Add tests that prove separate runtimes do not share sandbox state.
- [x] Add a demo host / CLI harness for smoke testing a sample bot script.

### Phase 5: Docs and delivery hygiene
- [ ] Update the diary after each implementation step with the exact commands, outcomes, and commit hashes.
- [ ] Update the changelog after each completed step.
- [ ] Re-run `docmgr doctor` and re-upload the final markdown bundle to reMarkable after implementation.

### Phase 6: Follow-up work
- [ ] Add Promise settlement support for async handlers if the host needs awaited results rather than Promise objects.
