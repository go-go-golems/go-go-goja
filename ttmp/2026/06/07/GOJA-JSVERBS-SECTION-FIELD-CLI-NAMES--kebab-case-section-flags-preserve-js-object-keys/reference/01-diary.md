---
Title: Diary
Ticket: GOJA-JSVERBS-SECTION-FIELD-CLI-NAMES
Status: active
Topics:
  - goja
  - xgoja
  - jsverbs
DocType: reference
Intent: diary
LastUpdated: 2026-06-07
---

# Diary

## Goal

This diary records the implementation of jsverb section-field CLI name normalization: CLI flags should be kebab-case everywhere, while JavaScript function parameters and bound section object keys should retain the author-declared names.

## Step 1: Planning the CLI-name to JS-name Boundary

The first step was to turn the review comment and follow-up discussion into a concrete implementation plan. The key design point is that the jsverbs system needs two names for a field: the CLI/Glazed name used for flags and parsed values, and the JavaScript name used for positional parameters or object keys.

This planning step intentionally happened before code changes. The previous conservative fix protected bound section object keys by not normalizing section fields, but that left named sections with inconsistent CLI flags. The design guide now records the stronger invariant: every CLI-facing field should be kebab-case, and JavaScript-facing values should be remapped to declared names.

### Prompt Context

**User prompt (verbatim):** "Ok, add tasks and a small design/implementation guide, then build it, commit at appropriate intervals, keep a diary as you work."

**Assistant interpretation:** Create project-tracking docs first, then implement the jsverbs field-name remapping behavior with tests and commits.

**Inferred user intent:** Make the section-field naming change deliberate and reviewable, not an ad hoc patch, while preserving a continuation-friendly record of decisions and validation.

**Commit (code):** N/A — planning only.

### What I did
- Created ticket workspace at `ttmp/2026/06/07/GOJA-JSVERBS-SECTION-FIELD-CLI-NAMES--kebab-case-section-flags-preserve-js-object-keys/`.
- Wrote `tasks.md` with implementation and validation tasks.
- Wrote `design-doc/01-kebab-case-section-flags-preserve-js-object-keys.md` with the two-name design and test plan.
- Started this diary.

### Why
- The behavior crosses command construction and runtime invocation, so a design note reduces the chance of fixing only one side.
- The previous regression showed that tests need to prove both the CLI field names and the JS-facing object keys.

### What worked
- The existing code layout made the boundary clear: `command.go` owns Glazed field registration, `runtime.go` owns JS argument construction, and `binding.go` owns the plan connecting the two.

### What didn't work
- `docmgr status --summary-only` from inside `go-go-goja` resolved to the parent rag-evaluation-system docmgr config rather than a repo-local go-go-goja ticket workspace, so I created the repo-local `ttmp` workspace manually.

### What I learned
- `go-go-goja` already has a historical `ttmp/` directory and vocabulary files, but no repo-local `.ttmp.yaml`; parent-directory discovery can pick up the surrounding workspace.

### What was tricky to build
- The tricky part is not command-name normalization itself. The tricky part is preserving two name domains: Glazed field names for CLI/config parsing and JavaScript names for object keys. A one-name model is simpler but leaks CLI conventions into JS objects.

### What warrants a second pair of eyes
- The design choice for `bind: "all"` and `bind: "context"` should be reviewed. The proposed behavior is to expose JS-facing values by default while preserving raw parsed values in context.

### What should be done in the future
- Implement the field-name binding map and verify it with bound-section, shared-section, `all`, and `context` tests.

### Code review instructions
- Start with `design-doc/01-kebab-case-section-flags-preserve-js-object-keys.md`.
- Then review `pkg/jsverbs/binding.go`, `pkg/jsverbs/command.go`, and `pkg/jsverbs/runtime.go` after implementation.
- Validate with `go test ./pkg/jsverbs -count=1`.

### Technical details
- Current conservative behavior: default section normalizes to kebab-case; named sections preserve declared field names.
- Target behavior: all CLI-facing field definitions use `cliFieldName(...)`; runtime remaps known parsed CLI names back to declared JS names before invoking JavaScript.

## Step 2: Implement Section Field CLI/JS Name Remapping

This step implemented the planned two-name boundary. The command schema now exposes named-section fields with kebab-case CLI names, while runtime invocation remaps known parsed fields back to the JavaScript names declared by the verb or shared section.

The implementation also preserved historical `bind: "all"` behavior by keeping it as a flat map, but with JavaScript-facing field names. `bind: "context"` now exposes `values` and `sections` with JS-facing names and adds `rawValues` for callers that need the literal parsed CLI/Glazed names.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Build the planned jsverbs section-field name remapping change, test it thoroughly, and commit the code separately from ticket bookkeeping.

**Inferred user intent:** Make jsverb CLIs idiomatic everywhere without breaking JavaScript authors who expect camelCase or otherwise authored object keys.

**Commit (code):** 5698f138feb768a583415e243769ea606be45d4b — "jsverbs: remap kebab-case section flags to JS keys"

### What I did
- Added `FieldNameBinding` metadata to `VerbBindingPlan` in `pkg/jsverbs/binding.go`.
- Registered positional, extra, local-section, and shared-section fields in the binding plan with both declared JS names and normalized CLI names.
- Changed `pkg/jsverbs/command.go` so named-section fields are also registered with `cliFieldName(...)`.
- Changed `pkg/jsverbs/runtime.go` to collect raw parsed section values, remap known CLI names to declared JS names, flatten JS-facing values for `bind: "all"`, and expose `rawValues` in `bind: "context"`.
- Added regression coverage in `pkg/jsverbs/jsverbs_test.go` for local bound sections, shared sections, top-level fields, `bind: "all"`, `bind: "context"`, and the existing fswatch jsverb path.
- Updated bundled docs in `pkg/xgoja/doc/02-jsverbs.md` and `pkg/doc/11-jsverbs-example-reference.md`.
- Ran targeted tests, package tests, app tests, and the repository pre-commit hook.

### Why
- The prior conservative fix avoided breaking JS object keys by leaving named-section CLI fields unnormalized.
- The target invariant is stronger and cleaner: all CLI-facing fields should be kebab-case, and JavaScript-facing parameters/objects should use authored names.
- Keeping a remap table makes this boundary explicit rather than relying on whichever name Glazed stores internally.

### What worked
- Targeted tests passed after the remapping helpers were in place:
  - `go test ./pkg/jsverbs -run 'TestTopLevelFieldNamesUseKebabCaseCLI|TestBoundSectionFieldNamesPreserveJavaScriptObjectKeys|TestSharedSectionFieldNamesUseKebabCaseCLIAndCamelCaseObjectKeys|TestBindAllAndContextUseJavaScriptFieldNames|TestFSWatchJsverbUsesInstalledHelper' -count=1 -v`
- Broader validation passed:
  - `go test ./pkg/jsverbs -count=1`
  - `go test ./pkg/xgoja/app -count=1`
- The commit pre-commit hook passed:
  - `golangci-lint run -v`
  - `GOWORK=off go vet -vettool=/tmp/glazed-lint ...`
  - `go generate ./...`
  - `go test ./...`

### What didn't work
- The first targeted test run failed because `runtime.go` still imported `schema` after positional lookup no longer needed it:
  - `pkg/jsverbs/runtime.go:14:2: "github.com/go-go-golems/glazed/pkg/cmds/schema" imported and not used`
  - Command: `go test ./pkg/jsverbs -run 'TestTopLevelFieldNamesUseKebabCaseCLI|TestBoundSectionFieldNamesPreserveJavaScriptObjectKeys|TestSharedSectionFieldNamesUseKebabCaseCLIAndCamelCaseObjectKeys|TestBindAllAndContextUseJavaScriptFieldNames|TestFSWatchJsverbUsesInstalledHelper' -count=1 -v`
- The first full `pkg/jsverbs` run exposed that my initial `bind: "all"` implementation changed its shape from a flat map to a section map:
  - `expected: "go-go-golems/go-go-goja"`
  - `actual  : "undefined/undefined"`
  - Failing test: `TestFixtureCommandsExecute/summarize_bind_all`
- The same run exposed an older test still feeding `localOnly` as the parsed CLI key after section fields became kebab-case:
  - `expected: string("from-local")`
  - `actual  : <nil>(<nil>)`
  - Failing test: `TestLocalSectionOverridesRegistrySharedSectionDuringCommandExecution`

### What I learned
- `values.Values.GetDataMap()` historically returns a flat field map, not a map grouped by section. `bind: "all"` needed to preserve that flat shape.
- `bind: "context"` is the right place to expose both views: JS-facing `values`/`sections` for normal JavaScript code and `rawValues` for diagnostics or low-level callers.
- Tests that construct parsed maps directly need to use the CLI/Glazed names, because they are standing in for the command parser.

### What was tricky to build
- The sharp edge was preserving three shapes at once: raw parsed section maps keyed by CLI names, JS-facing section maps keyed by declared names, and historical flat `bind: "all"` maps. The initial implementation conflated section maps with the old flat `all` map, which broke existing examples.
- Another subtlety is that shared-section and local-section fields are discovered through different paths. The binding plan now records fields from positional parameters, extra/default fields, and every referenced section after section resolution, so remapping works for both local `__section__` declarations and registry-provided shared sections.

### What warrants a second pair of eyes
- Review the `bind: "context"` contract: `values` is now a flat JS-facing map, `sections` is grouped and JS-facing, and `rawValues` is grouped by section with CLI names. That is useful, but it is a public shape and should be confirmed before release notes are written.
- Review duplicate/alias behavior if a verb ever declares two fields whose names normalize to the same CLI name inside one section. The current change follows existing CLI collision behavior rather than adding a new conflict detector.

### What should be done in the future
- Consider documenting `rawValues` as an explicit supported context field if downstream scripts start depending on it.
- Consider adding a scanner-time or command-build-time validation error for normalized field-name collisions in a section.

### Code review instructions
- Start in `pkg/jsverbs/binding.go` at `FieldNameBinding` and `addFieldNameBinding` to see how the remap table is built.
- Then review `pkg/jsverbs/command.go` to confirm all fields are registered with `cliFieldName(...)`.
- Then review `pkg/jsverbs/runtime.go`, especially `collectSectionValues`, `remapSectionValues`, and `flattenSectionValues`.
- Validate with `go test ./pkg/jsverbs -count=1`; full pre-commit validation already passed during commit `5698f138feb768a583415e243769ea606be45d4b`.

### Technical details
- CLI/Glazed field names are stored as kebab-case definitions such as `local-only` and `profile-path`.
- JavaScript positional lookup reads raw parsed values by `cliFieldName(binding.Field.Name)`.
- Bound sections receive cloned JS-facing maps, so JavaScript sees `filters.localOnly` rather than `filters["local-only"]`.
- `bind: "all"` receives a flattened JS-facing map, preserving existing code that reads `options.owner`.
- `bind: "context"` receives:
  - `values`: flattened JS-facing map.
  - `sections`: grouped JS-facing section map.
  - `rawValues`: grouped raw CLI-name section map.

