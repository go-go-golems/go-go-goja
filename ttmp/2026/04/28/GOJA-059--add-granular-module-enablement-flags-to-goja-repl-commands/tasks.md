# Tasks

## TODO

- [x] Add tasks here

- [ ] Investigate current module loading behavior across all goja-repl commands (run, tui, eval, create, etc.)
- [ ] Design common Glazed schema/flags for module enablement (--enable-module, --disable-module, --safe-mode, etc.)
- [ ] Add module enablement flags to rootOptions and propagate to all commands
- [ ] Implement module filtering logic in engine builder based on flag values
- [ ] Add module enablement to TUI/evaluator Config and wire through command flags
- [ ] Add module enablement to run command and runScriptOptions
- [ ] Write tests for module enablement filtering (enable specific, disable specific, safe mode)
- [ ] Write tests for CLI flag parsing and integration
- [ ] Update documentation (pkg/doc, README, help) with module security model
- [ ] Consider jsverbs integration: extend registry/invoker to respect module enablement
