# Changelog

## 2026-05-25

- Initial workspace created


## 2026-05-25

Created xgoja provider architecture review and onboarding report.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/pkg/xgoja/provider/provider.go — Reviewed existing-runner adapter pattern
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/module_sections.go — Reviewed module section aggregation and runtime initializer flow
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/capabilities.go — Reviewed capability abstractions
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-012--review-xgoja-provider-architecture-and-onboarding/design/01-xgoja-provider-architecture-review-and-onboarding-guide.md — Architecture review report with mental model


## 2026-05-25

Uploaded xgoja architecture review report to reMarkable and passed docmgr doctor.


## 2026-05-25

Expanded XGOJA-012 into detailed implementation plan for RuntimeFactory typing, provider utilities, capability cleanup, docs, and numbered examples.


## 2026-05-25

Typed CommandSetContext.RuntimeFactory and removed Discord adapter's local runtime-factory type assertion.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/pkg/xgoja/provider/provider.go — Discord provider now uses providerapi.RuntimeFactory directly
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/commands.go — RuntimeFactory interface added and CommandSetContext.RuntimeFactory typed


## 2026-05-25

Extracted shared providerutil helpers for module config sections and runtime initializers.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/pkg/xgoja/provider/provider.go — Discord adapter now uses providerutil
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/app/module_sections.go — Built-in command path now uses providerutil
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerutil/sections.go — Shared section collection and runtime initializer helpers


## 2026-05-25

Renamed package-scoped capability API and removed unused component initializer abstraction.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/capabilities.go — ModuleCapability renamed to PackageCapability and component initializer removed
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/registry.go — Registry now exposes package capability naming

