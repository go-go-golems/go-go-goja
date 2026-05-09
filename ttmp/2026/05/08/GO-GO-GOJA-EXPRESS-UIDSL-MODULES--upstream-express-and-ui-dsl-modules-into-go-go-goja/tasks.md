# Tasks

## GO-GO-GOJA-EXPRESS-UIDSL-MODULES

### T01 — Ticket and initial design

- [x] Create docmgr ticket workspace.
- [x] Store initial upstreaming analysis and design guide.
- [x] Add implementation task plan.
- [ ] Commit ticket planning docs.

### T02 — Upstream `pkg/gojahttp`

- [ ] In `/home/manuel/code/wesen/corporate-headquarters/go-go-goja`, create `pkg/gojahttp`.
- [ ] Copy host infrastructure from db-browser `internal/web`, excluding `express_module.go`.
- [ ] Rename package from `web` to `gojahttp`.
- [ ] Rename/update session cookie comments and default naming.
- [ ] Keep renderer injection and avoid importing `modules/uidsl`.
- [ ] Move/adapt route, body, session, request/response, and host tests.
- [ ] Run `go test ./pkg/gojahttp -count=1`.

### T03 — Add `modules/express`

- [ ] Create `modules/express` in go-go-goja.
- [ ] Move/adapt the Express registrar from db-browser `internal/web/express_module.go`.
- [ ] Implement `NewRegistrar(host *gojahttp.Host, opts ...Option)`.
- [ ] Keep `express` runtime-scoped via `engine.RuntimeModuleRegistrar`.
- [ ] Do not default-register `express` with `modules.Register` in the first version.
- [ ] Add runtime integration tests using `require("express")` and `httptest`.

### T04 — Add `modules/uidsl`

- [ ] Copy db-browser `internal/uidsl` into go-go-goja `modules/uidsl`.
- [ ] Keep `RenderAny`, `Loader`, and `NewRegistrar` public.
- [ ] Register aliases `ui.dsl` and `ui` through the registrar.
- [ ] Move/adapt render, table, filters, links, and component tests.
- [ ] Run `go test ./modules/uidsl -count=1`.

### T05 — Documentation and TypeScript declarations

- [ ] Add go-go-goja docs for the Express-style module.
- [ ] Add go-go-goja docs for the `ui.dsl` module.
- [ ] Add TypeScript declarations or declaration descriptors for `express`.
- [ ] Add TypeScript declarations or declaration descriptors for `ui.dsl`.
- [ ] Document that `express` is Express-style, not full Express-compatible.

### T06 — Upstream validation

- [ ] Run `go test ./pkg/gojahttp ./modules/express ./modules/uidsl -count=1`.
- [ ] Run `go test ./... -count=1` in go-go-goja.
- [ ] Run `GOWORK=off go test ./... -count=1` in go-go-goja.
- [ ] Run lint if practical.

### T07 — Migrate goja-hosting-site

- [ ] Replace local `pkg/web` and `pkg/uidsl` imports with upstream go-go-goja packages.
- [ ] Validate goja-hosting-site tests.
- [ ] Delete or deprecate local packages once green.

### T08 — Migrate db-browser

- [ ] Replace `internal/web` with `pkg/gojahttp` + `modules/express` imports.
- [ ] Replace `internal/uidsl` with `modules/uidsl` imports.
- [ ] Validate `go test ./...`.
- [ ] Run db-browser final validation and component smoke scripts.
- [ ] Delete local copied packages if no local behavior remains.

### T09 — Follow-ups after migration

- [ ] Decide whether `ui.dsl` should become a default registry module.
- [ ] Decide whether scoped table query state should be implemented upstream or first in db-browser.
- [ ] Consider static theme asset support after the upstream packages are stable.
- [ ] Consider server-interactive UI events as a separate project after the host boundary is stable.
