# Tasks

## Phase 1: Inventory and classification

- [ ] Save a reproducible inventory script under `scripts/01-inventory-goja-bindings.*`
- [ ] Classify every discovered package by adapter pattern: loader, register, runtime registrar, host-coupled, internal/app-coupled, or defer
- [ ] Record module names, import paths, config needs, runtime owner needs, and security class for each candidate

## Phase 2: Provider adapter conventions

- [x] Decide provider wrapper location conventions: source repo vs first-party `go-go-goja/pkg/xgoja/providers/...`
- [x] Define provider ID, module alias, config-schema, and documentation naming conventions
- [ ] Decide whether to add reusable adapter helpers for `modules.NativeModule`, `Register(reg, opts)`, and runtime registrars

## Phase 3: Simple provider implementations

- [x] Implement first-party safe/core providers for simple `go-go-goja/modules/*` modules
- [x] Implement guarded host-capability providers for `fs`, `exec`, and `database` with explicit config/security docs
- [ ] Implement external simple providers for `cozodb-goja`, `workspace-manager`, `pinocchio`, `smailnail`, `goja-git`, and `devctl-logjs` as appropriate

## Phase 4: Multi-module provider sets

- [ ] Implement `loupedeck/runtime/js` provider set
- [ ] Implement `goja-github-actions/pkg/modules/*` provider set
- [ ] Implement `geppetto/pkg/js/modules/geppetto` provider
- [ ] Evaluate and optionally implement `zigctl` provider with safe device/network configuration

## Phase 5: Internal and app-coupled bindings

- [ ] Plan public extraction or same-module adapters for `css-visual-diff/internal/cssvisualdiff/*`
- [ ] Define host-service interfaces for `openai-app-server/pkg/js` modules
- [ ] Decide whether to extract reusable APIs from `discord-bot/internal/jsdiscord`
- [ ] Decide whether `plz-confirm`, `scraper`, `js-analyzer`, and `go-minitrace` contain provider-sized APIs or should remain runtime/tooling code

## Phase 6: Tests, examples, and security review

- [x] Add generated xgoja build/run smoke tests for every implemented provider
- [x] Add `examples/xgoja/providers/<provider>/` examples or per-repo equivalent Makefile smokes
- [ ] Add provider security matrix covering filesystem, process, network, credential, device, and server/listener capabilities
- [x] Document config fields, defaults, failure modes, and dangerous capabilities for every provider

## Phase 7: Validation and closure

- [x] Run focused Go tests for modified repositories/packages
- [x] Run generated xgoja provider smokes with `GOWORK=off` where needed
- [x] Update diary/changelog after each implementation tranche
- [x] Run `docmgr doctor --ticket XGOJA-006 --stale-after 30`
- [ ] Close XGOJA-006 after docs, examples, validation, and security review are complete
