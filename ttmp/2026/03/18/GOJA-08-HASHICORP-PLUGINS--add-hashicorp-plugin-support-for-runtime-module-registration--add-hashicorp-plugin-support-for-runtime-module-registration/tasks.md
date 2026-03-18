# Tasks

## Done

- [x] Create a new docmgr ticket for HashiCorp plugin support in go-go-goja.
- [x] Import `/tmp/goja-plugins.md` into the ticket and read it as source material rather than copying it verbatim.
- [x] Inspect the current engine, module registry, runtime-owner, and runtime consumer code paths.
- [x] Write a detailed intern-oriented analysis, design, and implementation guide grounded in repository evidence.
- [x] Relate key files, validate the ticket with `docmgr doctor`, and upload the final bundle to reMarkable.
- [x] Update GOJA-08 docs, changelog, and diary after each implementation phase.
- [x] Validate with `go test`, rerun `docmgr doctor`, and re-upload the refreshed bundle to reMarkable.
- [x] Add a user-facing sample plugin under a top-level examples path plus a small README for plugin authors.

## In Progress

- [ ] Expand GOJA-08 from core plugin support into a user-facing productization pass with explicit follow-up tasks, commits, and diary entries.
- [ ] Add plugin discovery visibility and diagnostics to the REPL surfaces so users can see what loaded and why.

## Next

- [x] Refactor `engine.Factory` and `engine.Runtime` so runtime-scoped module registrars and runtime cleanup hooks are supported.
- [x] Add HashiCorp `go-plugin` dependency and shared plugin contract/package scaffolding.
- [x] Implement plugin discovery, manifest validation, client lifecycle, and module reification under a new host package.
- [x] Add runtime integration tests covering per-runtime registration, invalid plugin rejection, and runtime cleanup.
- [x] Wire plugin configuration into at least one runtime entrypoint and provide a working example/test plugin.
- [ ] Add plugin discovery visibility and diagnostics to the REPL surfaces so users can see what loaded and why.
- [ ] Add CLI/config trust-policy knobs for plugin allowlisting and wire them into supported entrypoints.
- [ ] Wire plugin configuration into one additional runtime consumer beyond `repl` and `js-repl`.
- [ ] Refresh the GOJA-08 task list, changelog, diary, validation results, and reMarkable bundle after the productization pass.
