# Changelog

## 2026-05-25

- Initial workspace created


## 2026-05-25

Fixed PR 40 review issues: runtime override initializers now use selected profile descriptors, and package capabilities attach to every descriptor while providerutil dedupes package-level application.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/module_sections.go — selected descriptors carry package capabilities on every module
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/root.go — eval recomputes selected descriptors for parsed runtime
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/run.go — run recomputes selected descriptors for parsed runtime
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/tui.go — repl/TUI recomputes selected descriptors for parsed runtime
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerutil/sections.go — package capability section/init application deduped by package and capability


## 2026-05-25

Closed after addressing PR 40 review comments, adding regression tests, nearby runtime override hardening, and validation.


## 2026-05-25

Fixed follow-up PR 40 review issue: command providers now receive the resolved fallback runtime profile in CommandSetContext.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/command_providers.go — CommandSetContext.RuntimeProfile now uses resolved fallback profile
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/command_providers_test.go — regression test covers omitted command provider runtimeProfile with RuntimeFactory.NewRuntime

