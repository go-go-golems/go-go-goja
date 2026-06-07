# Tasks

## TODO

- [ ] Add a binding-plan representation that distinguishes JavaScript field names from CLI/Glazed field names.
- [ ] Register named-section fields with kebab-case CLI names.
- [ ] Remap parsed named-section values back to declared JavaScript object keys before invoking JS.
- [ ] Preserve current default/top-level kebab-case behavior.
- [ ] Preserve `bind: "all"` and `bind: "context"` values in a way that gives JS code declared field keys for section data.
- [ ] Add regression tests for:
  - [ ] `localOnly: { section: "filters" }` exposed as `local-only` in command schema.
  - [ ] bound section object receives `filters.localOnly`, not `filters["local-only"]`.
  - [ ] shared section fields get kebab-case CLI names but camelCase JS object keys.
  - [ ] default/top-level `profilePath` remains `profile-path` at the CLI and positional JS parameter value still arrives.
- [ ] Update bundled jsverbs documentation to describe the CLI-name/JS-name boundary.
- [ ] Run `go test ./pkg/jsverbs -count=1` and relevant xgoja tests.
- [ ] Commit the implementation.

## DONE

- [x] Create ticket workspace, tasks, design guide, and diary before implementation.
