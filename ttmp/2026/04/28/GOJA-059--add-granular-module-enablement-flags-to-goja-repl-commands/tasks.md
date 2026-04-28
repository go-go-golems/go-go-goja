# Tasks

## TODO

- [x] Add tasks here
- [x] Investigate current module loading behavior across all goja-repl commands (run, tui, eval, create, etc.)
- [x] Design common Glazed schema/flags for module enablement (--enable-module, --disable-module, --safe-mode, etc.)
- [ ] Add module enablement flags to rootOptions and propagate to all commands
- [x] Implement module filtering logic in engine builder based on flag values
- [ ] Add module enablement to TUI/evaluator Config and wire through command flags
- [ ] Add module enablement to run command and runScriptOptions
- [x] Write tests for module enablement filtering (enable specific, disable specific, safe mode)
- [ ] Write tests for CLI flag parsing and integration
- [ ] Update documentation (pkg/doc, README, help) with module security model
- [ ] Consider jsverbs integration: extend registry/invoker to respect module enablement
- [x] Implement engine/module_middleware.go: ModuleSelector, ModuleMiddleware, built-in middlewares (Safe, Only, Exclude, Add, Custom), Pipeline helper, intersect/filterOut utilities
- [x] Add UseModuleMiddleware to FactoryBuilder and integrate into Build()
- [x] Deprecate old API family in engine/module_specs.go (DefaultRegistryModules, DataOnlyDefaultRegistryModules, DefaultRegistryModulesNamed, DefaultRegistryModule) as thin wrappers
- [ ] Wire module middleware through commandSupport.newAppWithOptions (replapi-backed commands)
- [ ] Wire module middleware through runScriptFile (run command)
- [ ] Wire module middleware through javascript.New (TUI evaluator)
- [ ] Write integration tests for CLI flags (--safe-mode, --enable-module, --disable-module)
- [ ] Update pkg/doc/04-repl-usage.md and README.md with module security model
- [ ] Final review: ensure no behavioral regression, old APIs still work, all tests pass
