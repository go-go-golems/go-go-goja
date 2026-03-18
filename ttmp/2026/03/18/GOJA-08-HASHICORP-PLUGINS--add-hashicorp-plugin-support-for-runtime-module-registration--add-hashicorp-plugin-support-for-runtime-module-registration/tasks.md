# Tasks

## Done

- [x] Create a new docmgr ticket for HashiCorp plugin support in go-go-goja.
- [x] Import `/tmp/goja-plugins.md` into the ticket and read it as source material rather than copying it verbatim.
- [x] Inspect the current engine, module registry, runtime-owner, and runtime consumer code paths.
- [x] Write a detailed intern-oriented analysis, design, and implementation guide grounded in repository evidence.
- [x] Update GOJA-08 docs, changelog, and diary after each implementation phase.
- [x] Refactor `engine.Factory` and `engine.Runtime` so runtime-scoped module registrars and runtime cleanup hooks are supported.
- [x] Add HashiCorp `go-plugin` dependency and shared plugin contract/package scaffolding.
- [x] Implement plugin discovery, manifest validation, client lifecycle, and module reification under a new host package.
- [x] Add runtime integration tests covering per-runtime registration, invalid plugin rejection, and runtime cleanup.
- [x] Wire core plugin configuration into the runtime entrypoints and provide working example/test plugins.
- [x] Add Glazed help pages for plugin usage, plugin internals, and a build/install tutorial.
- [x] Add `--plugin-dir` to `js-repl` and make both REPLs default to `~/.go-go-goja/plugins/...`.
- [x] Move concrete plugin binaries/examples out of `pkg/hashiplugin/testplugin` into top-level `plugins/...` paths.
- [x] Add a user-facing sample plugin under a top-level examples path plus a small README for plugin authors.
- [x] Add plugin discovery visibility and diagnostics to the REPL surfaces so users can see what loaded and why.
- [x] Add CLI/config trust-policy knobs for plugin allowlisting and wire them into supported entrypoints.
- [x] Wire plugin configuration into one additional runtime consumer beyond `repl` and `js-repl`.
- [x] Validate with `go test`, rerun `docmgr doctor`, and re-upload the refreshed bundle to reMarkable.
