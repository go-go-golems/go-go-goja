# Changelog

## 2026-06-03

- Initial workspace created
- Created design document: ModuleConfigCapability analysis and implementation guide
- Created investigation diary with 2 steps
- Created architecture and extensibility analysis
- Uploaded to reMarkable
- Researched codebase (27 source files), wrote design doc and investigation diary
- Identified key findings: ModuleConfigCapability is well-proposed, map[string]any return type is a design smell, dual decodeConfig paths need unification


## 2026-06-03

Added independent review and implementation guide for Glazed section values as pre-runtime xgoja module config, including critique of prior docs, per-descriptor patch design, source-aware default handling, provider-owned command implications, and future plugin/codegen constraints.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/03-review-and-runtime-config-design.md — New review/design deliverable
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/tasks.md — Marked review/design guide task complete
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/vocabulary.yaml — Added missing vocabulary for validation


## 2026-06-03

Uploaded GOJA-053 Runtime Config Design Review bundle to reMarkable at /ai/2026/06/03/GOJA-053 and recorded upload results in the independent diary.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Upload result and final diary step


## 2026-06-04

Revised runtime config design after follow-up: simplified Geppetto config to profile/default-profile plus turns-dsn/turns-db, replaced map return recommendation with ModuleConfigPatch provenance wrapper, and documented SectionContext/ModuleDescriptor merge model.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/03-review-and-runtime-config-design.md — Follow-up design revision
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Diary step for follow-up revision


## 2026-06-04

Added second design pass for GOJA-053: using Glazed schema.Section and values.SectionValues as the static module config and CLI/config/env merge layer, avoiding a separate ModuleConfigPatch framework.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/04-glazed-sectionvalues-module-config-design.md — New Glazed-native design guide
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Diary step for Glazed SectionValues research
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/tasks.md — Marked second research/design pass complete


## 2026-06-04

Added Glazed config/flags merge research logbook and vocabulary entries for flags/research; documents useful, stale, and update-needed resources for the SectionValues design pass.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/05-glazed-config-research-logbook.md — Research logbook for Glazed config/flags merge pass
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/vocabulary.yaml — Added flags and research topic vocabulary


## 2026-06-04

Uploaded GOJA-053 Glazed SectionValues Config Design bundle to reMarkable at /ai/2026/06/03/GOJA-053.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded reMarkable upload step


## 2026-06-04

Re-uploaded GOJA-053 Glazed SectionValues Config Design bundle with the design guide, Glazed research logbook, and independent diary included.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/04-glazed-sectionvalues-module-config-design.md — Included in final reMarkable bundle
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded final reMarkable bundle contents
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/05-glazed-config-research-logbook.md — Included evidence logbook in final reMarkable bundle


## 2026-06-04

Added xgoja codegen and generated script execution runthrough, clarifying buildspec/codegen, provider registry, app RuntimeFactory, engine runtime creation, ModuleContext, RuntimeModuleContext, runtimebridge, and GOJA-053 config timing.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/05-xgoja-codegen-and-script-execution-runthrough.md — New lifecycle walkthrough deliverable
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded codegen/runtime walkthrough step
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/tasks.md — Marked walkthrough task complete


## 2026-06-04

Uploaded GOJA-053 xgoja Codegen Runtime Runthrough bundle to reMarkable at /ai/2026/06/03/GOJA-053.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/05-xgoja-codegen-and-script-execution-runthrough.md — Uploaded lifecycle walkthrough
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded upload step


## 2026-06-04

Added codegen/runtime runthrough research logbook, tracking usefulness, gaps, stale/confusing areas, and update needs for resources used in the lifecycle walkthrough.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded research logbook step
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/06-codegen-runtime-runthrough-research-logbook.md — New resource logbook for lifecycle walkthrough
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/tasks.md — Marked research logbook task complete


## 2026-06-04

Uploaded GOJA-053 Codegen Runtime Research Logbook bundle to reMarkable at /ai/2026/06/03/GOJA-053.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/05-xgoja-codegen-and-script-execution-runthrough.md — Included supporting lifecycle runthrough in upload
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded upload step
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/06-codegen-runtime-runthrough-research-logbook.md — Uploaded codegen/runtime resource logbook


## 2026-06-04

Added Sobek ECMAScript Modules analysis and research logbook, concluding that Sobek/ESM is useful for targeted experiments but is not currently a simplification replacement for xgoja require/native provider module machinery.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/06-sobek-esm-native-module-analysis.md — Sobek ESM design/implementation guide
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded Sobek ESM analysis step
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/07-sobek-esm-research-logbook.md — Sobek ESM resource logbook
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/tasks.md — Marked Sobek ESM analysis/logbook tasks complete


## 2026-06-04

Uploaded GOJA-053 Sobek ESM Native Module Analysis bundle to reMarkable at /ai/2026/06/03/GOJA-053.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/06-sobek-esm-native-module-analysis.md — Uploaded Sobek ESM analysis
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded Sobek ESM upload step
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/07-sobek-esm-research-logbook.md — Uploaded Sobek ESM resource logbook


## 2026-06-04

Added generic symbol inventory/glossary for Service/Context/Capability/Runtime/Module/Spec/Factory/Registry names and uploaded it to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/04-independent-review-diary.md — Recorded glossary and upload step
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/08-generic-symbol-inventory-and-glossary.md — New generic symbol glossary and rename/separation recommendations
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/tasks.md — Marked glossary task complete


## 2026-06-04

Implemented Geppetto provider cleanup for GOJA-053: adopted current xgoja provider/engine API names, removed legacy provider config gates/storage fields, renamed `profileRegistries` to `defaultProfileRegistries`, and added Glazed/xgoja config section mapping for supported default-profile overrides (Geppetto commit 6f0bc2d).

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/provider/provider.go — Simplified Geppetto provider config and added capability mapping
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/provider/provider_test.go — Updated provider tests for simplified config and ignored legacy fields
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/09-buildspec-spec-rename-diary.md — Recorded implementation step and validation caveats

## 2026-06-04

Ran a Pinocchio-hosted Geppetto JS smoke test with the Pinocchio profile registry, explicit `--profile`, and `--turns-db`; verified the SQLite turn store contained one persisted final turn and four blocks. Also migrated Pinocchio's remaining direct go-go-goja engine API imports for the local GOJA-053 workspace (Pinocchio commit 802620e).

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/pinocchio/cmd/pinocchio/cmds/js.go — Pinocchio JS host used for profile and turn-store smoke
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/pinocchio/cmd/examples/scopedjs-tui-demo/environment.go — Downstream engine API import migration
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/reference/09-buildspec-spec-rename-diary.md — Recorded smoke-test details and validation caveats
