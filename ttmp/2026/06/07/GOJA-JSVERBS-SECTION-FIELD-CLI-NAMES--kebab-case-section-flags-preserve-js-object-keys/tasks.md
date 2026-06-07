# Tasks

## DONE

- [x] Create ticket workspace, tasks, design guide, and diary before implementation.
- [x] Add a binding-plan representation that distinguishes JavaScript field names from CLI/Glazed field names.
- [x] Register named-section fields with kebab-case CLI names.
- [x] Remap parsed named-section values back to declared JavaScript object keys before invoking JS.
- [x] Preserve current default/top-level kebab-case behavior.
- [x] Preserve `bind: "all"` and `bind: "context"` values with JavaScript-facing field names while keeping raw parsed values available in context.
- [x] Add regression tests for:
  - [x] `localOnly: { section: "filters" }` exposed as `local-only` in command schema.
  - [x] bound section object receives `filters.localOnly`, not `filters["local-only"]`.
  - [x] shared section fields get kebab-case CLI names but camelCase JS object keys.
  - [x] default/top-level `profilePath` remains `profile-path` at the CLI and positional JS parameter value still arrives.
  - [x] `bind: "all"` and `bind: "context"` see JavaScript-facing names for known jsverb fields.
- [x] Update bundled jsverbs documentation to describe the CLI-name/JS-name boundary.
- [x] Run targeted jsverbs tests plus `go test ./pkg/jsverbs -count=1` and `go test ./pkg/xgoja/app -count=1`.

## TODO

- [x] Run full repository pre-commit/full test suite at commit time.
- [ ] Consider a future public API for exposing both raw CLI names and JS names more explicitly in `bind: "context"`.
