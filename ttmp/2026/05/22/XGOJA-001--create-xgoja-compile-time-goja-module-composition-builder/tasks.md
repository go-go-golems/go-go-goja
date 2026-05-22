# Tasks

## Done

- [x] Create docmgr ticket for xgoja.
- [x] Read the source article on Go plugin strategies and xgoja compile-time module composition.
- [x] Inspect current go-go-goja runtime, native module, jsverbs, REPL, and out-of-process plugin architecture.
- [x] Write the intern-facing analysis/design/implementation guide.
- [x] Create and update the implementation diary.
- [x] Validate the ticket with docmgr doctor.
- [x] Upload the ticket bundle to reMarkable.

## TODO

- [x] Decide whether xgoja starts as a new repository or as a package inside go-go-goja.
- [x] Implement Phase 1: CLI skeleton and command wiring.
- [x] Implement Phase 2: buildspec YAML parsing, defaults, normalization, and validation.
- [x] Implement Phase 3: provider API and fixture provider package.
- [ ] Implement Phase 4: deterministic go.mod/main.go/embed generation with golden tests.
- [ ] Implement Phase 5: pure xgoja generated app runtime with REPL/jsverbs command support.
- [ ] Implement Phase 6: go mod tidy/go build execution and workdir diagnostics.
- [ ] Implement Phase 7: doctor, inspect, and list-modules diagnostics.
- [ ] Implement Phase 8: STDBIN adapter and Cobra attach modes after pure mode works.
