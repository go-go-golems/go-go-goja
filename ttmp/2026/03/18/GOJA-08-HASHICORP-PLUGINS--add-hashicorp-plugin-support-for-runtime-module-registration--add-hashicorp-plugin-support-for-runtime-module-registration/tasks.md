# Tasks

## Done

- [x] Create a new docmgr ticket for HashiCorp plugin support in go-go-goja.
- [x] Import `/tmp/goja-plugins.md` into the ticket and read it as source material rather than copying it verbatim.
- [x] Inspect the current engine, module registry, runtime-owner, and runtime consumer code paths.
- [x] Write a detailed intern-oriented analysis, design, and implementation guide grounded in repository evidence.
- [x] Relate key files, validate the ticket with `docmgr doctor`, and upload the final bundle to reMarkable.

## In Progress

- [ ] Update GOJA-08 docs, changelog, and diary after each implementation phase.

## Next

- [ ] Validate with `go test`, rerun `docmgr doctor`, and re-upload the refreshed bundle to reMarkable.

- [x] Refactor `engine.Factory` and `engine.Runtime` so runtime-scoped module registrars and runtime cleanup hooks are supported.
- [x] Add HashiCorp `go-plugin` dependency and shared plugin contract/package scaffolding.
- [x] Implement plugin discovery, manifest validation, client lifecycle, and module reification under a new host package.
- [x] Add runtime integration tests covering per-runtime registration, invalid plugin rejection, and runtime cleanup.
- [x] Wire plugin configuration into at least one runtime entrypoint and provide a working example/test plugin.
