# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Created architecture design for reframing xgoja as a Go-backed JavaScript runtime compiler with source graph, import resolver, provider graph, build plan, and runtime plan.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/01-xgoja-source-graph-and-bundler-architecture.md — Primary architecture design
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/reference/01-investigation-diary.md — Investigation diary


## 2026-06-12

Updated architecture design v2 with Go workspace resolution for generated builds and DTS sidecars, including go.work discovery, GoModulePlan, precedence rules, doctor output, and implementation phases.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/gomod.go — Current generated module rendering target
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/01-xgoja-source-graph-and-bundler-architecture.md — Architecture v2 workspace section


## 2026-06-12

Added second architecture document for xgoja v2 spec and migration tooling, treating v1 as migratable legacy input and v2 as the native planner schema.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/02-xgoja-v2-spec-and-migration-architecture.md — V2 spec and migration architecture


## 2026-06-12

Simplified the v2 spec design around goja-executed source only: removed broad bundler/package-manager knobs, kept intent-level compile fields, and documented external frontend bundles as assets.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/02-xgoja-v2-spec-and-migration-architecture.md — Simplified v2 spec design


## 2026-06-12

Expanded tasks into a phase-by-phase hard-cutover plan for v2 xgoja, including migration tooling, workspace resolution, provider graph, source graph, command cutover, docs/examples, and v1 removal.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/reference/01-investigation-diary.md — Diary Step 5 task planning entry
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/tasks.md — Detailed v2 hard-cutover task list


## 2026-06-12

Implemented initial specv2 package with DTOs, defaults, strict loading, validation, rendering, tests, v1 migration diagnostic, and v1 spec inventory.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/specv2_test.go — Initial v2 schema test coverage
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/types.go — Initial v2 config DTOs


## 2026-06-12

Added v1-to-v2 migration conversion for providers, runtime modules, command surfaces, jsverbs, TypeScript compile intent, help/assets, target artifacts, replacements, and migration warnings.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/migrate_v1.go — V1-to-v2 conversion implementation
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/migrate_v1_test.go — Migration coverage for TypeScript jsverbs


## 2026-06-12

Added xgoja migrate-spec command with v1-to-v2 output, in-place migration, backups, check mode, warning output, and root command coverage.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_migrate_spec.go — migrate-spec CLI implementation
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/root.go — Wires migrate-spec into xgoja root
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/root_test.go — Root command migration output and in-place backup tests


## 2026-06-12

Added migration coverage for all v1 examples, migrate-spec check/warning tests, and the xgoja/v2 migration help page.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/doc/16-migrating-to-xgoja-v2.md — User migration guide
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/examples_migration_test.go — All example v1 specs migrate to valid v2
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/root_test.go — migrate-spec command output/check/in-place/warning tests


## 2026-06-12

Added workspace resolution package with go.work discovery/parsing, go.mod module mapping, GoModulePlan, and precedence for explicit replace, CLI replace, go.work, and versioned requirements.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/workspace/workspace.go — Workspace discovery and GoModulePlan resolution
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/workspace/workspace_test.go — Workspace resolution tests


## 2026-06-12

Added providergraph package over existing providerapi with selected provider/module/command validation, alias tracking, TypeScript descriptor extraction, and provider API audit note.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/providergraph/graph.go — Provider graph implementation
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/providergraph/graph_test.go — Provider graph validation tests
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/sources/local/03-provider-api-audit.md — Provider API audit conclusion


## 2026-06-12

Added sourcegraph package with disk/fs source origins, source kinds, include/exclude discovery, origin metadata, local import resolution, runtime alias classification, and unknown bare import diagnostics.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/sourcegraph/graph.go — Source graph implementation
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/sourcegraph/graph_test.go — Source graph discovery and import-resolution tests


## 2026-06-12

Added fs.FS-backed TypeScript bundling helper for embedded/provider source graphs, resolving relative imports without disk ResolveDir and preserving runtime module aliases as externals.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/tsscript/fs_bundle.go — fs.FS-backed virtual entry bundler
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/tsscript/fs_bundle_test.go — fs.FS bundling tests


## 2026-06-12

Threaded fs.FS source metadata through jsverbs scan/runtime transforms and used fs-backed TypeScript bundling for provider/embedded bundled jsverbs, preserving overlay-before-bundling behavior.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/jsverbs/model.go — Adds RootFS source metadata to scan/runtime DTOs
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/jsverbs/runtime.go — Passes RootFS to runtime transforms
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/jsverbs/scan.go — Carries fs.FS roots from ScanFS through registry file specs
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/typescript.go — Selects fs-backed TypeScript bundling when source metadata has RootFS
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/typescript_jsverbs_test.go — Provider fs.FS TypeScript jsverbs helper import regression test


## 2026-06-12

Step 14: wired fs.FS-backed TypeScript runtime bundling for jsverbs provider/embedded sources (commit 2b9873166f7bf464f181d347f21fbf3a357aec47)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/jsverbs/scan.go — Carries fs.FS roots through ScanFS (commit 2b9873166f7bf464f181d347f21fbf3a357aec47)
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/typescript.go — Uses BundleVirtualEntryFS for fs-backed runtime transforms (commit 2b9873166f7bf464f181d347f21fbf3a357aec47)


## 2026-06-12

Replaced xgoja jsverbs source scanning with a sourcegraph-backed adapter and threaded runtime module aliases into TypeScript externals.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/root.go — scanVerbSource now builds a sourcegraph and scans jsverbs from graph files
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/typescript.go — Runtime aliases are appended to TypeScript externals
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/sourcegraph/graph.go — File entries now carry origin metadata for graph-backed readers


## 2026-06-12

Fixed graph-backed scan compatibility by classifying all registered provider modules as scan-time runtime aliases for provider jsverb sources.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/jsverb_sources.go — command-provider JSVerbSourceSet scans use provider-wide aliases
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/root.go — sourceGraphRuntimeAliases includes provider module names/default aliases


## 2026-06-12

Added initial v2 plan compiler that combines specv2 validation, Go module planning, provider graph, source graph, command plans, artifact plans, and runtime aliases.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/plan/plan.go — Initial v2 Plan type and compiler
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/plan/plan_test.go — Planner coverage for runtime aliases


## 2026-06-12

Wired workspace Go module plans into generated go.mod rendering and into build/gen-dts sidecar generation.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_build.go — Build passes GoModulePlan to generated workspace writer
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_gen_dts.go — gen-dts sidecar go.mod uses GoModulePlan
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/gomod.go — RenderGoMod consumes workspace.GoModulePlan for require/replace output
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/workspace_plan.go — Buildspec-to-workspace requirement planning for commands


## 2026-06-12

Added xgoja doctor module-resolution rows showing module path, version, local dir, resolution kind/source, and required-by context.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_doctor.go — Doctor emits workspace module resolution diagnostics


## 2026-06-12

Updated xgoja doctor to detect xgoja/v2 specs, load specv2, compile through the v2 planner with a synthetic provider registry, and emit plan/source/module diagnostics.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_doctor.go — Doctor v2 detection and planner diagnostics
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/empty_fs.go — Empty fs.FS used for synthetic provider source planning
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/root_test.go — V2 doctor command smoke test


## 2026-06-12

Updated xgoja build to load v2 specs through the v2 planner and convert the artifact/runtime/source plan into the existing generator build spec bridge.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_build.go — Build command detects v2 specs and uses planner output
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/root_test.go — V2 build dry-run smoke test
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/v2_bridge.go — V2 plan to buildspec bridge for generated builds


## 2026-06-12

Updated xgoja gen-dts to load v2 specs through the planner bridge and use planned Go modules for sidecar go.mod rendering; normalized provider import module paths in the v2 planner.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_gen_dts.go — gen-dts detects v2 specs and uses planner bridge
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/plan/plan.go — Provider import paths are normalized to module roots for Go module planning
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/root_test.go — V2 gen-dts command smoke test


## 2026-06-12

Step 22: migrated examples/xgoja/15-typescript-jsverbs to native xgoja/v2 and validated doctor/gen-dts/build/run/serve smoke (commit f47d196)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/examples/xgoja/15-typescript-jsverbs/xgoja.yaml — First repository example migrated to v2


## 2026-06-12

Step 23: added xgoja/v2 reference doc and migrated HTTP serve jsverbs example to native v2 (commits 12ca66e, acbb5f9)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/doc/17-xgoja-v2-reference.md — New v2 reference
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/examples/xgoja/13-http-serve-jsverbs/xgoja.yaml — Native v2 HTTP serve jsverbs example


## 2026-06-12

Step 24: preserved embedded jsverb/help/assets semantics in v2 artifact bridge and migration, and updated HTTP serve example to embed local-sites (commit 82ea9e1)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/migrate_v1.go — Embed migration mapping
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/v2_bridge.go — V2 artifact source dependency bridge


## 2026-06-12

Step 25: hard-cut normal xgoja commands and examples to native v2; gen-dts now defaults from dts artifact; generate/list-modules also use v2 bridge (commits aa8ffa6, aa0436b)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/v2_bridge.go — Normal command v2 loading and migration diagnostic
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/examples/xgoja/README.md — Example index updated for v2


## 2026-06-12

Step 26: moved v2 plan-to-generation adaptation into internal/generate and made commands consume plan-based generator entrypoints (commits 023ffb3, 6fbbf12)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/plan.go — Plan-backed generator entrypoints and generator-local adaptation
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/v2_bridge.go — Now only loads/compiles native v2 plans and rejects legacy specs


## 2026-06-12

Step 27: rendered generator templates and embedded runtime JSON from v2 plans instead of the generator buildspec adapter (commit 38b1f10)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/plan.go — Plan-backed write functions now render templates from plan data
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/templates.go — Plan-native template data and embedded runtime JSON rendering


## 2026-06-12

Step 28: removed the v2 generator path's buildspec adapter; build/generate/gen-dts/list now use plan-native target, copy, go.mod, and runtime JSON helpers (commit 8e2ccc2)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/gomod.go — Plan-native go.mod rendering
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/plan.go — Plan-native copy/write helpers with BuildSpecFromPlan removed
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/v2_plan_helpers.go — Command-side v2 artifact target helper


## 2026-06-12

Step 29: deleted legacy buildspec-oriented generator APIs and replaced generator tests with v2 plan fixtures (commit 9a91695)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/generate.go — Legacy buildspec write/template APIs removed
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/generate_test.go — Generator tests now use v2 plan fixtures
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/generate/main.go — Legacy buildspec render/runtime-spec helpers removed


## 2026-06-12

Step 30: renamed legacy buildspec package to migratebuildspec and gave specv2 its own validation report types so v1 schema code is migration-only (commit ecb922f)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/cmd_migrate_spec.go — Only command path that loads migratebuildspec
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/migratebuildspec/build_spec.go — Legacy v1 schema quarantined for migrate-spec
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/specv2/report.go — Native v2 validation report types remove dependency on legacy schema package


## 2026-06-12

Step 31: archived buildspec-era help behind v2 docs and updated user guide/reference wording for plan-native v2 commands (commit a6fe8dd)

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/doc/02-user-guide.md — Rewritten as native xgoja/v2 user guide
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/doc/06-buildspec-reference.md — Legacy buildspec page reduced to migration pointer
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/doc/17-xgoja-v2-reference.md — Transition limits updated to plan-native command state

