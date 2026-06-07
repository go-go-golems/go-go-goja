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
