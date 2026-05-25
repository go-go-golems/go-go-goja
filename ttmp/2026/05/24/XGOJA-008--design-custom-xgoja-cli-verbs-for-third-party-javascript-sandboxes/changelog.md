# Changelog

## 2026-05-24

- Initial workspace created


## 2026-05-24

Created custom xgoja CLI verbs design guide with evidence from xgoja, loupedeck, discord-bot, css-visual-diff, and go-minitrace.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/design-doc/01-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes.md — Primary design guide


## 2026-05-24

Validated XGOJA-008 docs and uploaded design bundle to reMarkable at /ai/2026/05/24/XGOJA-008.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/design-doc/01-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes.md — Uploaded design bundle


## 2026-05-25

Revised command-provider design to return Glazed command sets instead of raw Cobra commands.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/design-doc/01-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes.md — Revised Glazed command-set provider design


## 2026-05-25

Added module-provided Glazed configuration sections and DecodeSectionInto initialization to the command-provider design.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/design-doc/01-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes.md — Added configurable module capability and Discord/Loupedeck composition design


## 2026-05-25

Validated and uploaded the module-section design revision to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/reference/01-diary.md — Recorded validation and reMarkable upload of module-section revision


## 2026-05-25

Extended module section design to built-in run repl jsverbs and runtime initializer capabilities.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/design-doc/01-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes.md — Added built-in command section aggregation and runtime initializer design


## 2026-05-25

Validated and uploaded the built-in module-section revision to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/reference/01-diary.md — Recorded validation and reMarkable upload of built-in module-section revision


## 2026-05-25

Expanded implementation tasks into granular phases for provider capabilities built-ins command providers examples and final validation.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/tasks.md — Granular XGOJA-008 implementation phases


## 2026-05-25

Implemented providerapi module capability registration interfaces and focused tests.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/capabilities.go — Added module section and initializer capability APIs
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/registry.go — Registry now stores and resolves package capabilities
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/registry_test.go — Added capability registration and validation tests


## 2026-05-25

Implemented app runtime-profile section aggregation and runtime initializer helpers with focused tests.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/module_sections.go — Added selected module descriptor resolution section aggregation and runtime initializer helpers
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/module_sections_test.go — Added focused aggregation and initializer tests


## 2026-05-25

Wired module-provided Glazed sections and runtime initializers into the built-in run command.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/run.go — Run command now aggregates runtime-profile module sections and calls runtime initializers
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/run_module_sections_test.go — Added run command fixture tests for module sections and DecodeSectionInto initialization


## 2026-05-25

Wired module-provided Glazed sections and runtime initializers into the built-in repl/TUI command.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/tui.go — REPL/TUI command now aggregates runtime-profile module sections and initializes runtimes
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/tui_module_sections_test.go — Added REPL/TUI fixture tests for module sections and runtime initialization

