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

