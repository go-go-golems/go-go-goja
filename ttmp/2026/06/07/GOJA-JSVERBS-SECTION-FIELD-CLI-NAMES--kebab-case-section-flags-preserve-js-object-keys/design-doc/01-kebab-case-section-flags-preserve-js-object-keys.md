---
Title: Kebab-case section flags while preserving JavaScript object keys
Ticket: GOJA-JSVERBS-SECTION-FIELD-CLI-NAMES
Status: active
Topics:
  - goja
  - xgoja
  - jsverbs
DocType: design-doc
Intent: implementation-guide
LastUpdated: 2026-06-07
RelatedFiles:
  - Path: /home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/pkg/jsverbs/command.go
    Note: Builds Glazed command descriptions from jsverb binding plans.
  - Path: /home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/pkg/jsverbs/runtime.go
    Note: Builds JavaScript invocation arguments from parsed Glazed values.
  - Path: /home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/pkg/jsverbs/binding.go
    Note: Computes the binding plan between JS parameters, sections, and fields.
  - Path: /home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/pkg/jsverbs/jsverbs_test.go
    Note: Regression tests for CLI names and JavaScript object keys.
---

# Kebab-case section flags while preserving JavaScript object keys

## Goal

The jsverbs command layer should expose idiomatic kebab-case CLI flags for all user-facing fields. JavaScript code should receive values under the names the JavaScript author declared. This means `localOnly` should appear as `--local-only` on the CLI, but a bound section object should still be read as `filters.localOnly` inside JavaScript.

The current conservative fix only normalizes default-section fields. It prevents broken JavaScript object keys, but it leaves named-section CLI flags inconsistent with the rest of the command surface. This design replaces that compromise with an explicit two-name model.

## Current behavior

The jsverbs system has three relevant layers:

1. `FieldSpec.Name` is the declared JavaScript-facing field name.
2. `fields.Definition.Name` is the Glazed field name and becomes the CLI/config key.
3. `buildArguments` converts parsed Glazed values into JavaScript function arguments.

After the top-level normalization change, default-section fields use kebab-case at the CLI boundary:

| Declared JS field | CLI/Glazed field | JS value delivery |
| --- | --- | --- |
| `profilePath` | `profile-path` | positional argument value |
| `docsJson` | `docs-json` | positional argument value |

For named-section fields we temporarily preserved names:

| Declared JS field | CLI/Glazed field | JS value delivery |
| --- | --- | --- |
| `localOnly` | `localOnly` | `filters.localOnly` |

The desired behavior is:

| Declared JS field | CLI/Glazed field | JS value delivery |
| --- | --- | --- |
| `localOnly` | `local-only` | `filters.localOnly` |
| `maxResults` | `max-results` | `filters.maxResults` |

## Design decision

Introduce explicit binding metadata for section field names:

- `JSName`: the declared JavaScript/object key.
- `CLIName`: the normalized Glazed/CLI/config key.
- `SectionSlug`: the section the field belongs to.

The implementation does not need to change `FieldSpec` itself. It can add derived maps to `VerbBindingPlan`, because the distinction is command/runtime binding metadata, not parse metadata.

## Proposed data structure

Add a small struct in `binding.go`:

```go
type FieldNameBinding struct {
    SectionSlug string
    JSName      string
    CLIName     string
}
```

Add this to `VerbBindingPlan`:

```go
type VerbBindingPlan struct {
    Verb               *VerbSpec
    Parameters         []ParameterBinding
    ExtraFields        []ExtraFieldBinding
    ReferencedSections []string
    FieldNames         []FieldNameBinding
}
```

Every field that is registered into a Glazed section should get a field-name binding. The mapping should include:

1. fields declared directly on referenced `__section__` sections,
2. positional parameter fields,
3. extra fields placed into the default section,
4. extra fields placed into named sections.

For default-section positional fields, `JSName` matters mainly for diagnostics and consistency. For named sections and `bind: all`/`bind: context`, the mapping is required to reconstruct JS-facing objects.

## Command construction

`command.go` should register every Glazed field using `CLIName`:

```go
field, err := buildFieldDefinitionWithName(fieldSpec, cliName)
```

A cleaner implementation is to replace the boolean `normalizeName` argument with an explicit output name:

```go
func buildFieldDefinition(spec *FieldSpec, fieldName string) (*fields.Definition, error)
```

Callers compute `fieldName` with `cliFieldName(spec.Name)` for all CLI-facing fields. This removes ambiguity about whether a caller should normalize.

## Runtime argument construction

`runtime.go` currently creates `sectionValues` by copying parsed Glazed fields under `fieldValue.Definition.Name`, which is the CLI name. That map is correct for the CLI layer but not for JavaScript objects.

Change `buildArguments` to produce a JS-facing section map by applying the plan's field-name bindings:

```go
parsedBySection := map[string]map[string]interface{}{}
// parsedBySection["filters"]["local-only"] = value

jsBySection := map[string]map[string]interface{}{}
for _, binding := range plan.FieldNames {
    value, ok := parsedBySection[binding.SectionSlug][binding.CLIName]
    if ok {
        jsBySection[binding.SectionSlug][binding.JSName] = value
    }
}
```

Then:

- `BindingModeSection` should pass `cloneMap(jsBySection[binding.SectionSlug])`.
- `BindingModeContext` should expose both the JS-facing `sections` and the raw parsed values if useful.
- `BindingModeAll` should preserve its historical shape: a flat map of all parsed fields. The field names in that flat map should be JavaScript-facing names, not CLI names. This keeps existing verbs like `options.owner` working while still remapping `profile-path` back to `profilePath`.
- `BindingModePositional` should look up values using the binding's CLI name, then pass the value positionally.

## Handling fields not known to the plan

Some provider/runtime sections may be attached outside the jsverbs binding plan. Those sections should remain accessible under their parsed field names because jsverbs does not know their JavaScript-facing names. The remapping should therefore:

1. start from raw parsed values,
2. overlay remapped jsverb fields where known,
3. leave unknown fields unchanged.

This keeps provider config and framework-owned sections intact.

## Pseudocode

```go
func buildSectionValueMaps(parsedValues, plan) (raw, js map[string]map[string]interface{}) {
    raw = collectRawParsedValues(parsedValues)
    js = deepClone(raw)

    for _, b := range plan.FieldNames {
        value, ok := raw[b.SectionSlug][b.CLIName]
        if !ok {
            continue
        }
        delete(js[b.SectionSlug], b.CLIName)
        js[b.SectionSlug][b.JSName] = value
    }
    return raw, js
}

func buildArguments(parsedValues, plan, rootDir) []interface{} {
    raw, js := buildSectionValueMaps(parsedValues, plan)

    for _, binding := range plan.Parameters {
        switch binding.Mode {
        case BindingModeSection:
            args = append(args, cloneMap(js[binding.SectionSlug]))
        case BindingModeAll:
            args = append(args, flatten(js))
        case BindingModeContext:
            args = append(args, map[string]any{
                "values": js,
                "rawValues": raw,
                "sections": js,
            })
        case BindingModePositional:
            args = append(args, raw[binding.SectionSlug][cliFieldName(binding.Field.Name)])
        }
    }
}
```

## Tests

Required regression tests:

1. A field declared as `localOnly: { section: "filters" }` is registered as `local-only` in the command schema.
2. A bound `filters` object receives `filters.localOnly`, not `filters["local-only"]`.
3. A shared section with `maxResults` is exposed as `max-results` but received as `filters.maxResults`.
4. Existing top-level behavior remains unchanged: `profilePath` is `--profile-path`, and JS receives the positional value.
5. `bind: "all"` and `bind: "context"` receive JS-facing values for known jsverb fields.

## Risks

The main compatibility risk is code that already reads section object keys in kebab-case because of the brief intermediate regression. That behavior was never documented as intended and should not be preserved.

The second risk is ambiguity for unknown provider/runtime fields. Leaving unknown fields unchanged avoids lossy behavior.

## Validation commands

```bash
go test ./pkg/jsverbs -run 'TestTopLevelFieldNamesUseKebabCaseCLI|TestBoundSectionFieldNamesPreserveJavaScriptObjectKeys|TestFSWatchJsverbUsesInstalledHelper' -count=1
go test ./pkg/jsverbs -count=1
go test ./pkg/xgoja/app -count=1
```
