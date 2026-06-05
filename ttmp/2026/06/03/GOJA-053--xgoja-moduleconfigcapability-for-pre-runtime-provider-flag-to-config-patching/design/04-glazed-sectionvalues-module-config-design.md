---
Title: Glazed SectionValues as the xgoja Module Config Merge Layer
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - provider
    - capability
    - glazed
    - config
    - architecture
    - review
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: glazed/pkg/cmds/fields/field-value.go
      Note: |-
        FieldValue provenance logs, merge behavior, and JSON-compatible map conversion.
        FieldValue provenance and merge semantics
    - Path: glazed/pkg/cmds/schema/section-impl.go
      Note: |-
        Section implementation APIs for schema creation, map gathering, Cobra parsing, and defaults.
        Section construction and map/Cobra parsing APIs
    - Path: glazed/pkg/cmds/sources/sections.go
      Note: |-
        Existing SectionValues merge middleware primitives.
        Existing SectionValues merge middleware
    - Path: glazed/pkg/cmds/sources/update.go
      Note: |-
        FromMap, FromMapAsDefault, FromEnv, and typed map-to-section parsing behavior.
        FromMap and FromEnv source parsing behavior
    - Path: glazed/pkg/cmds/values/section-values.go
      Note: |-
        SectionValues and Values merge/decode APIs that can become the module config merge layer.
        SectionValues/Values merge and decode APIs
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: |-
        Runtime factory location where final SectionValues should be converted back to ModuleContext.Config.
        xgoja runtime factory config insertion point
    - Path: go-go-goja/pkg/xgoja/providerapi/capabilities.go
      Note: |-
        Provider capability API location for internal module config section and config mapping hooks.
        Capability API extension point
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/52
Summary: 'Second GOJA-053 design pass: use Glazed schema.Section and values.SectionValues as the config parsing, provenance, and merge layer instead of adding a separate xgoja config patch framework.'
LastUpdated: 2026-06-04T00:00:00Z
WhatFor: Use when implementing a Glazed-native xgoja module config pipeline that parses static module config and command flags into SectionValues and merges them before Module.New.
WhenToUse: Before adding ModuleConfigCapability or modifying xgoja runtime factory config flow; before extending Glazed APIs for section mapping and provenance-preserving merges.
---


# Glazed SectionValues as the xgoja Module Config Merge Layer

## Executive summary

The second design pass changes the shape of GOJA-053. We do **not** need a separate `ModuleConfigPatch` mini-framework if Glazed can already parse, type-check, merge, and preserve provenance for structured values. The better design is to let providers expose an **internal module config `schema.Section`** and, separately, one or more **public CLI/config/env sections**. xgoja parses `xgoja.yaml` module config through the internal section, the existing command pipeline parses flags/config/env through public sections, and the provider maps public section values into an internal config `SectionValues` override. xgoja then merges `SectionValues` and converts the final field values back to JSON for today’s `providerapi.ModuleContext.Config`.

This gives us the key properties we wanted without inventing another config system:

- **Type checking** comes from `fields.Definition` and `GatherFieldsFromMap`.
- **Provenance/history** comes from `fields.FieldValue.Log`.
- **Precedence** comes from `FieldValues.Merge`, `MergeAsDefault`, and middleware ordering.
- **Source labels** come from existing parse options like `fields.WithSource("xgoja.yaml")`, `fields.WithSource("config")`, `fields.WithSource("env")`, and `fields.WithSource("cobra")`.
- **Provider control** comes from mapping public sections into internal config sections instead of exposing every internal config knob as a flag.

The recommended xgoja API is therefore:

1. Provider exposes an internal config section:
   `ModuleConfigSection(ctx, descriptor) (schema.Section, error)`.
2. xgoja parses `ModuleInstance.Config` into `*values.SectionValues` using that section.
3. Provider maps parsed command values into an internal config override:
   `ModuleConfigValuesFromSections(ctx, req) (*values.SectionValues, error)`.
4. xgoja merges `staticConfig.MergeFields(override)`.
5. xgoja converts final `SectionValues.Fields.ToInterfaceMap()` to JSON and passes that to `Module.New`.

This is exactly “merging values with values,” scoped to a concrete selected module instance. The important detail is namespace translation: CLI/public section values are in user-facing field names; module config values are in internal module config field names.

---

## 1. Problem statement

The original issue is still the same: xgoja can expose provider-owned Glazed command sections and can run runtime initializers, but parsed command/config/env values arrive after `Module.New`. Geppetto needs profile registry, default profile, and turn-store settings before `Module.New` creates module options.

The previous design introduced a `ModuleConfigPatch` idea: providers would translate parsed section values into a JSON-keyed patch map, and xgoja would merge that into `ModuleInstance.Config`. That works, but it duplicates concepts Glazed already has:

- a schema for fields (`schema.Section`),
- typed field definitions (`fields.Definition`),
- parsed field values (`fields.FieldValue`),
- section-scoped value bags (`values.SectionValues`),
- multi-section value bags (`values.Values`),
- source/provenance logs (`fields.ParseStep`), and
- merge semantics (`FieldValues.Merge`, `MergeAsDefault`, `Values.Merge`).

So the new question is: can xgoja treat module config as a hidden Glazed section and reuse the existing machinery? The answer is yes, with two small API additions and a few helper functions.

---

## 2. Glazed concepts an intern must understand

### 2.1 `schema.Section`: the field contract

A Glazed `schema.Section` groups named fields and their definitions. The interface lives in `glazed/pkg/cmds/schema/schema.go:13-31` and exposes:

- `GetDefinitions() *fields.Definitions`
- `GetSlug() string`
- `GetPrefix() string`
- `Clone() Section`

The standard implementation is `SectionImpl` in `glazed/pkg/cmds/schema/section-impl.go`. You create one with `schema.NewSection(slug, name, options...)` (`section-impl.go:60-74`), add fields with `schema.WithFields(...)` (`section-impl.go:110-115`), and optionally add a CLI prefix with `schema.WithPrefix(...)` (`section-impl.go:77-81`).

For xgoja we should distinguish two section kinds:

```text
Public command section
  slug:   geppetto
  prefix: geppetto-
  fields: profile-registries, default-profile, turns-dsn, turns-db
  exposed as flags/config/env

Internal module config section
  slug:   geppetto-module-config or geppetto.config
  prefix: usually empty; not exposed as flags
  fields: profileRegistries or profile-registries, defaultProfile, turnsDSN, turnsDB
  used to parse xgoja.yaml ModuleInstance.Config and initialize Module.New
```

These sections may share some fields, but they should not be forced to be identical. The provider controls the mapping.

### 2.2 `values.SectionValues`: parsed values for one section

`values.SectionValues` is the parsed data for one section. It is defined in `glazed/pkg/cmds/values/section-values.go:62-68`:

```go
type SectionValues struct {
    Section Section
    Fields  *fields.FieldValues
}
```

Important methods:

- `values.NewSectionValues(section, options...)` creates an empty parsed section (`section-values.go:103-117`).
- `SectionValues.MergeFields(other)` merges another section’s fields, with `other` taking precedence (`section-values.go:136-140`).
- `SectionValues.DecodeInto(&dst)` decodes field values into a struct (`section-values.go:151-152`).
- `SectionValues.GetField(k)` returns a raw value for quick inspection (`section-values.go:143-149`).

This is the type xgoja should use for internal module config state.

### 2.3 `values.Values`: parsed values for many sections

`values.Values` is an ordered map from section slug to `*SectionValues` (`section-values.go:155-177`). Existing xgoja commands already receive `*values.Values` after Glazed parses defaults/config/env/flags.

Important methods:

- `Values.GetOrCreate(section)` creates section values for a schema section (`section-values.go:219-233`).
- `Values.Merge(other)` merges multiple sections (`section-values.go:187-217`).
- `Values.DecodeSectionInto(slug, &dst)` decodes one section (`section-values.go:246-260`).
- `Values.GetField(slug, key)` retrieves a full `*fields.FieldValue` including log (`section-values.go:280-286`).

Raw `values.Values` is ideal as input to provider mapping, because it contains all parsed public sections. It is not ideal as the return type for module config mapping, because it is too broad. The provider should return a single internal `*values.SectionValues` for the module config section.

### 2.4 `fields.FieldValue`: value plus provenance

Each parsed field is a `fields.FieldValue`, defined in `glazed/pkg/cmds/fields/field-value.go:11-17`:

```go
type FieldValue struct {
    Value      interface{}
    Definition *Definition
    Log        []ParseStep
}
```

`Log` is the provenance trail. `FieldValue.Update` appends a parse step (`field-value.go:67-83`), `UpdateWithLog` preserves an existing log (`field-value.go:90-99`), and `FieldValue.Merge` appends provenance from the overriding value (`field-value.go:113-118`).

This matters because we can tell whether a value came from:

- `xgoja.yaml`,
- a config file,
- an environment variable,
- Cobra flags,
- defaults, or
- a provider mapping step.

### 2.5 `FieldValues.Merge` and `MergeAsDefault`

`fields.FieldValues.Merge(other)` makes `other` take precedence and copies the overriding field’s log (`field-value.go:328-343`). `MergeAsDefault(other)` only sets fields that do not already exist (`field-value.go:345-357`).

This is the core primitive for xgoja module config precedence:

```text
module config defaults < xgoja.yaml static config < command config/env/flags mapped to module config
```

### 2.6 Map parsing: `GatherFieldsFromMap` and `sources.FromMap`

`fields.Definitions.GatherFieldsFromMap` parses a `map[string]interface{}` through field definitions (`gather-fields.go:21-75`). It validates types, parses strings through field parsers, honors `onlyProvided`, and attaches parse metadata.

`schema.SectionImpl.GatherFieldsFromMap` is a thin wrapper over definitions (`section-impl.go:263-268`). `sources.FromMap` turns a section-map into parsed values (`sources/update.go:31-45`, `95-117`):

```go
sources.FromMap(map[string]map[string]interface{}{
    "geppetto.config": instance.Config,
}, fields.WithSource("xgoja.yaml"))
```

This is exactly what xgoja needs to parse `ModuleInstance.Config` into a `SectionValues`.

### 2.7 Existing middleware order already models precedence

Glazed source middleware execution is documented in `sources/middlewares.go:37-56`. A middleware that modifies parsed values calls `next` first, which means lower-precedence sources run first and higher-precedence sources merge over them.

The existing xgoja command path already parses command-level values with a chain like:

```text
defaults → config files → env → args → cobra flags
```

For internal module config, xgoja can run a small, local chain:

```go
sources.Execute(
    schema.NewSchema(schema.WithSections(configSection)),
    internalValues,
    sources.FromMap(staticModuleConfigMap, fields.WithSource("xgoja.yaml")),
    // optionally sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
)
```

or it can call `configSection.GetDefinitions().GatherFieldsFromMap` directly.

---

## 3. Proposed xgoja design

### 3.1 New provider capabilities

Add two small capabilities. One exposes the internal module config schema. The other maps already-parsed public command values into that internal schema.

```go
// ModuleConfigSectionCapability exposes the provider's internal module config
// schema. This section is used to parse xgoja.yaml module config and to validate
// pre-runtime config overrides. It is not automatically exposed as CLI flags.
type ModuleConfigSectionCapability interface {
    PackageCapability

    ModuleConfigSection(SectionContext, ModuleDescriptor) (schema.Section, error)
}

// ModuleConfigValuesCapability maps parsed command/config/env values into the
// provider's internal module config section. It returns an override SectionValues
// whose Section must be req.ConfigSection.
type ModuleConfigValuesCapability interface {
    PackageCapability

    ModuleConfigValuesFromSections(
        context.Context,
        ModuleConfigValuesRequest,
    ) (*values.SectionValues, error)
}

type ModuleConfigValuesRequest struct {
    SectionContext SectionContext
    Descriptor     ModuleDescriptor

    ConfigSection schema.Section
    StaticConfig  *values.SectionValues

    // The already parsed command values from built-in xgoja or provider-owned commands.
    CommandValues *values.Values
}
```

Why split the capabilities?

- Some modules may only want typed static `xgoja.yaml` config and no CLI overrides.
- Some modules may expose public flags and map them into internal config.
- The internal section should not necessarily be shown in help or bound to Cobra.

A provider can implement both on one capability struct.

### 3.2 Runtime factory flow

The new runtime creation flow should be:

```text
RuntimeFactory.NewRuntimeFromSections(ctx, profile, commandValues, opts...)
  ↓
resolve selected module descriptors
  ↓
for each ModuleInstance + ModuleDescriptor:
  if provider exposes ModuleConfigSectionCapability:
      internalSection = capability.ModuleConfigSection(...)
      staticValues = parse ModuleInstance.Config through internalSection
      overrideValues = capability.ModuleConfigValuesFromSections(req{staticValues, commandValues})
      finalValues = staticValues.Clone()
      finalValues.MergeFields(overrideValues)
      configMap = finalValues.Fields.ToInterfaceMap()
      configJSON = json.Marshal(configMap)
      pass configJSON to Module.New
  else:
      keep existing json.Marshal(instance.Config) behavior
```

The low-level `engine` package does not need to know any of this. The insertion point remains `pkg/xgoja/app/factory.go`, where `providerRuntimeModuleSpec.RegisterRuntimeModule` currently marshals `s.instance.Config` into `ModuleContext.Config`.

### 3.3 Pseudocode for parsing static module config

```go
func parseModuleConfigMap(
    section schema.Section,
    config map[string]any,
) (*values.SectionValues, error) {
    if config == nil {
        config = map[string]any{}
    }

    fieldValues, err := section.GetDefinitions().GatherFieldsFromMap(
        config,
        true, // onlyProvided: do not synthesize defaults unless we opt in later
        fields.WithSource("xgoja.yaml"),
        fields.WithMetadata(map[string]any{"config_kind": "module-static"}),
    )
    if err != nil {
        return nil, err
    }

    return values.NewSectionValues(section, values.WithFields(fieldValues))
}
```

Why `onlyProvided=true`?

- It preserves current behavior: provider code defaults still live in provider code.
- It prevents section defaults from silently changing module config behavior.
- We can add an explicit opt-in later, such as `ModuleConfigDefaultsMode`.

If a provider wants internal config schema defaults to become real module config, xgoja can add:

```go
schema.NewSchema(schema.WithSections(section)).UpdateWithDefaults(vals, fields.WithSource(fields.SourceDefaults))
```

But do not make that the v1 default without tests and migration notes.

### 3.4 Pseudocode for mapping public CLI values to internal config values

Provider-owned mapping is the core design point. The provider sees full command values and returns a section in the internal config namespace.

```go
type geppettoCapability struct{}

func (geppettoCapability) ModuleConfigSection(
    ctx providerapi.SectionContext,
    desc providerapi.ModuleDescriptor,
) (schema.Section, error) {
    return schema.NewSection(
        "geppetto.config",
        "Geppetto module config",
        schema.WithFields(
            fields.New("profileRegistries", fields.TypeStringList),
            fields.New("defaultProfile", fields.TypeString),
            fields.New("turnsDSN", fields.TypeString),
            fields.New("turnsDB", fields.TypeString),
        ),
    )
}

func (geppettoCapability) ConfigSections(ctx providerapi.SectionContext) ([]schema.Section, error) {
    // Public CLI/config/env section. Only expose what users should set.
    return []schema.Section{schema.NewSection(
        "geppetto",
        "Geppetto",
        schema.WithPrefix("geppetto-"),
        schema.WithFields(
            fields.New("profile-registries", fields.TypeStringList),
            fields.New("default-profile", fields.TypeString),
            fields.New("turns-dsn", fields.TypeString),
            fields.New("turns-db", fields.TypeString),
        ),
    )}
}

func (geppettoCapability) ModuleConfigValuesFromSections(
    ctx context.Context,
    req providerapi.ModuleConfigValuesRequest,
) (*values.SectionValues, error) {
    out, err := values.NewSectionValues(req.ConfigSection)
    if err != nil {
        return nil, err
    }

    // Copy public CLI/config/env fields into internal config fields.
    if err := providerutil.CopyFieldIfProvided(
        req.CommandValues, "geppetto", "profile-registries",
        out, "profileRegistries",
        fields.WithSource("module-config-map"),
    ); err != nil { return nil, err }

    if err := providerutil.CopyFieldIfProvided(req.CommandValues, "geppetto", "default-profile", out, "defaultProfile"); err != nil { return nil, err }
    if err := providerutil.CopyFieldIfProvided(req.CommandValues, "geppetto", "turns-dsn", out, "turnsDSN"); err != nil { return nil, err }
    if err := providerutil.CopyFieldIfProvided(req.CommandValues, "geppetto", "turns-db", out, "turnsDB"); err != nil { return nil, err }

    return out, nil
}
```

`CopyFieldIfProvided` should copy the source `FieldValue`’s typed value and append mapping metadata while preserving provenance.

### 3.5 Pseudocode for a Glazed helper: `CopyFieldIfProvided`

This helper could live in `pkg/xgoja/providerutil` first, then move upstream to Glazed if it proves generally useful.

```go
func CopyFieldIfProvided(
    all *values.Values,
    sourceSectionSlug string,
    sourceField string,
    target *values.SectionValues,
    targetField string,
    opts ...fields.ParseOption,
) error {
    if all == nil || target == nil || target.Section == nil {
        return nil
    }

    src, ok := all.GetField(sourceSectionSlug, sourceField)
    if !ok || !fieldWasProvided(src) {
        return nil
    }

    targetDef, ok := target.Section.GetDefinitions().Get(targetField)
    if !ok {
        return fmt.Errorf("target field %s not found in section %s", targetField, target.Section.GetSlug())
    }

    v, err := src.GetInterfaceValue()
    if err != nil {
        return fmt.Errorf("read %s.%s: %w", sourceSectionSlug, sourceField, err)
    }

    // Add mapping provenance while preserving the original source log.
    log := append([]fields.ParseStep(nil), src.Log...)
    step := fields.NewParseStep(append([]fields.ParseOption{
        fields.WithSource("module-config-map"),
        fields.WithParseStepValue(v),
        fields.WithMetadata(map[string]any{
            "source_section": sourceSectionSlug,
            "source_field":   sourceField,
            "target_section": target.Section.GetSlug(),
            "target_field":   targetField,
        }),
    }, opts...)...)
    log = append(log, step)

    return target.Fields.UpdateWithLog(targetField, targetDef, v, log...)
}
```

`fieldWasProvided` should ignore pure defaults by default:

```go
func fieldWasProvided(v *fields.FieldValue) bool {
    if v == nil || len(v.Log) == 0 {
        return false
    }
    last := v.Log[len(v.Log)-1]
    return last.Source != fields.SourceDefaults && last.Source != "default"
}
```

We might eventually want options:

- copy defaults too,
- copy only if source exists regardless of log,
- copy only config/env/cobra and not args,
- copy only Cobra flags,
- treat empty string as absent.

### 3.6 Converting final `SectionValues` into `ModuleContext.Config`

After static and override config are merged, xgoja converts the internal config values to JSON:

```go
func sectionValuesToRawJSON(sv *values.SectionValues) (json.RawMessage, error) {
    if sv == nil || sv.Fields == nil {
        return nil, nil
    }
    m, err := sv.Fields.ToInterfaceMap()
    if err != nil {
        return nil, err
    }
    b, err := json.Marshal(m)
    if err != nil {
        return nil, err
    }
    return json.RawMessage(b), nil
}
```

This keeps the public `providerapi.ModuleContext` unchanged for v1. Providers that already decode JSON can continue to do so. Later, xgoja could add `ModuleContext.ConfigValues *values.SectionValues` for providers that want direct access to provenance.

---

## 4. How this avoids a second config framework

With this design, xgoja does not define its own patch format, deep merge rules, type system, or provenance system. It only orchestrates Glazed objects.

| Need | Glazed primitive |
|---|---|
| Field schema | `schema.Section`, `fields.Definition` |
| Parse static `xgoja.yaml` config map | `GatherFieldsFromMap`, `sources.FromMap` |
| Parse CLI flags | existing `ConfigSectionCapability` + Cobra/Glazed parser |
| Preserve source history | `FieldValue.Log []ParseStep` |
| Merge precedence | `FieldValues.Merge`, `MergeAsDefault`, `Values.Merge` |
| Convert final config to JSON | `FieldValues.ToInterfaceMap` + `json.Marshal` |
| Provider-specific field mapping | small helper around `Values.GetField` and `SectionValues.Fields.UpdateWithLog` |

The only new xgoja concepts are:

- **internal module config section**: hidden schema for `Module.New` config;
- **module config mapping hook**: provider maps public parsed values into the internal section.

Everything else is existing Glazed behavior.

---

## 5. API options and recommendations

### Option A: xgoja-local minimal API

Add xgoja capabilities only. Use Glazed types directly.

```go
type ModuleConfigSectionCapability interface {
    PackageCapability
    ModuleConfigSection(SectionContext, ModuleDescriptor) (schema.Section, error)
}

type ModuleConfigValuesCapability interface {
    PackageCapability
    ModuleConfigValuesFromSections(context.Context, ModuleConfigValuesRequest) (*values.SectionValues, error)
}
```

Pros:

- Small change.
- No Glazed API changes required.
- Easy to test in go-go-goja first.

Cons:

- Provider mapping helpers live in xgoja initially.
- Other Glazed users cannot reuse the mapping pattern until helpers move upstream.

### Option B: add generic Glazed section mapping helpers

Add helpers in Glazed, likely under `pkg/cmds/values` or `pkg/cmds/sources`:

```go
type SectionFieldMapping struct {
    SourceSection string
    SourceField   string
    TargetField   string
    IncludeDefault bool
}

func MapValuesToSection(
    source *values.Values,
    targetSection schema.Section,
    mappings []SectionFieldMapping,
    opts ...fields.ParseOption,
) (*values.SectionValues, error)
```

Pros:

- Reusable outside xgoja.
- Makes “public section → internal config section” a normal Glazed pattern.
- Keeps provenance-preserving copying in one place.

Cons:

- Requires careful API design in Glazed.
- Needs tests for provenance, type conversion, missing fields, aliases, and defaults.

### Option C: use existing `ConfigMapper` / patternmapper

Glazed already has config mappers for raw config-file structures. `ConfigFileMapper` maps raw config into `map[sectionSlug]map[fieldName]value` (`load-fields-from-config.go:19-24`), and patternmapper has a builder for source-to-target rules (`pattern_mapper_builder.go:15-52`).

This is useful inspiration, but not directly enough for xgoja command-value mapping because:

- patternmapper maps raw config objects, not `values.Values` with `FieldValue.Log`;
- it returns raw maps, not `SectionValues`;
- it does not preserve `ParseStep` history from existing parsed fields.

Recommendation: do not use patternmapper directly for v1. Borrow the validated mapping-rule idea for a new `SectionValuesMapper` later.

### Recommended path

Use Option A now and design helpers so they can become Option B later.

---

## 6. Implementation plan

### Phase 1: Fix or avoid unsafe clone behavior

Before relying heavily on `SectionValues.Clone`, inspect and likely fix `glazed/pkg/cmds/values/section-values.go:119-133`. The function creates a merged field map and then overwrites it with original `FieldValue` pointers:

```go
fields_, err := fields.NewFieldValues().Merge(ppl.Fields)
ret := &SectionValues{Section: ppl.Section, Fields: fields_}
ppl.Fields.ForEach(func(k string, v *fields.FieldValue) {
    ret.Fields.Set(k, v) // overwrites cloned value with original pointer
})
```

This means clone is not as isolated as xgoja would want for per-runtime config mutation. Either fix it in Glazed or avoid clone by constructing fresh `SectionValues` through `FieldValues.Clone()`.

Suggested fix:

```go
func (ppl *SectionValues) Clone() *SectionValues {
    return &SectionValues{
        Section: ppl.Section,
        Fields:  ppl.Fields.Clone(),
    }
}
```

Add a test that mutating a cloned field log/value does not mutate the original.

### Phase 2: Add provider API in go-go-goja

Add to `pkg/xgoja/providerapi/capabilities.go`:

```go
type ModuleConfigSectionCapability interface {
    PackageCapability
    ModuleConfigSection(SectionContext, ModuleDescriptor) (schema.Section, error)
}

type ModuleConfigValuesRequest struct {
    SectionContext SectionContext
    Descriptor     ModuleDescriptor
    ConfigSection  schema.Section
    StaticConfig   *values.SectionValues
    CommandValues  *values.Values
}

type ModuleConfigValuesCapability interface {
    PackageCapability
    ModuleConfigValuesFromSections(context.Context, ModuleConfigValuesRequest) (*values.SectionValues, error)
}
```

### Phase 3: Add xgoja providerutil helpers

Add helpers to `pkg/xgoja/providerutil`, not app code:

```go
func ParseModuleInstanceConfig(
    section schema.Section,
    config map[string]any,
    opts ...fields.ParseOption,
) (*values.SectionValues, error)

func CopyFieldIfProvided(...)

func SectionValuesToRawJSON(*values.SectionValues) (json.RawMessage, error)
```

`ParseModuleInstanceConfig` can use `GatherFieldsFromMap` directly or `sources.FromMap` with a one-section schema. Direct is simpler and avoids middleware chain overhead.

### Phase 4: Extend `RuntimeFactory.NewRuntimeFromSections`

In `pkg/xgoja/app/factory.go`, when building each `providerRuntimeModuleSpec`, compute the final config bytes before creating the spec:

```go
configBytes := json.RawMessage(nil)
if configSectionCap := findModuleConfigSectionCapability(descriptor); configSectionCap != nil {
    section, err := configSectionCap.ModuleConfigSection(sectionCtx, descriptor)
    staticValues, err := providerutil.ParseModuleInstanceConfig(section, instance.Config, fields.WithSource("xgoja.yaml"))

    finalValues := staticValues.Clone()
    if mapper := findModuleConfigValuesCapability(descriptor); mapper != nil && vals != nil {
        override, err := mapper.ModuleConfigValuesFromSections(ctx, providerapi.ModuleConfigValuesRequest{
            SectionContext: sectionCtx,
            Descriptor:     descriptor,
            ConfigSection:  section,
            StaticConfig:   staticValues,
            CommandValues:  vals,
        })
        if override != nil {
            finalValues.MergeFields(override)
        }
    }

    configBytes, err = providerutil.SectionValuesToRawJSON(finalValues)
} else {
    configBytes, err = json.Marshal(instance.Config)
}
```

Then `providerRuntimeModuleSpec` can carry either `instance.Config` or the precomputed `json.RawMessage`:

```go
type providerRuntimeModuleSpec struct {
    instance ModuleInstance
    module   providerapi.Module
    services providerapi.HostServices
    config   json.RawMessage // if set, use directly
}
```

### Phase 5: Update built-in commands

Same as the earlier design: built-in runtime creation paths need to call `NewRuntimeFromSections` with parsed command values.

- `evalSourceWithInitializers`
- `runScriptFileWithInitializers`
- TUI `newXGojaTUIEvaluator`
- jsverbs invoker

Runtime initializers still run afterwards. This feature only changes what `Module.New` sees.

### Phase 6: Geppetto provider implementation

Define two sections:

```go
// Internal config section: used to parse xgoja.yaml config and build ModuleContext.Config.
func geppettoModuleConfigSection() (schema.Section, error) {
    return schema.NewSection("geppetto.config", "Geppetto module config",
        schema.WithFields(
            fields.New("profileRegistries", fields.TypeStringList),
            fields.New("defaultProfile", fields.TypeString),
            fields.New("turnsDSN", fields.TypeString),
            fields.New("turnsDB", fields.TypeString),
        ),
    )
}

// Public CLI section: exposed to generated commands.
func geppettoCLISection() (schema.Section, error) {
    return schema.NewSection("geppetto", "Geppetto",
        schema.WithPrefix("geppetto-"),
        schema.WithFields(
            fields.New("profile-registries", fields.TypeStringList),
            fields.New("default-profile", fields.TypeString),
            fields.New("turns-dsn", fields.TypeString),
            fields.New("turns-db", fields.TypeString),
        ),
    )
}
```

Then map CLI fields to config fields:

```go
func (c capability) ModuleConfigValuesFromSections(ctx context.Context, req providerapi.ModuleConfigValuesRequest) (*values.SectionValues, error) {
    out, err := values.NewSectionValues(req.ConfigSection)
    if err != nil { return nil, err }

    mappings := []providerutil.FieldMapping{
        {SourceSection: "geppetto", SourceField: "profile-registries", TargetField: "profileRegistries"},
        {SourceSection: "geppetto", SourceField: "default-profile", TargetField: "defaultProfile"},
        {SourceSection: "geppetto", SourceField: "turns-dsn", TargetField: "turnsDSN"},
        {SourceSection: "geppetto", SourceField: "turns-db", TargetField: "turnsDB"},
    }
    return providerutil.MapFields(req.CommandValues, out, mappings...)
}
```

Geppetto config simplification:

- remove `allowRegistryLoad`, `allowNetwork`, `allowTools`, `enableStorage`, and nested `turns` from this xgoja config path;
- interpret `turnsDSN` or `turnsDB` presence as explicit storage opt-in;
- keep host-mediated construction for actual turn store objects if needed.

---

## 7. Testing strategy

### 7.1 Glazed tests

If we modify Glazed or add helpers there:

1. `SectionValues.Clone` deep-clones `FieldValue` values/logs.
2. `MapValuesToSection` copies typed values and preserves original logs.
3. `MapValuesToSection` appends mapping provenance metadata.
4. `MapValuesToSection` skips default-only source values by default.
5. `MapValuesToSection` can include defaults when explicitly configured.
6. Type mismatch between source value and target field returns a useful error.

### 7.2 go-go-goja providerutil tests

1. Parse `map[string]any` static config into internal `SectionValues`.
2. Static config parse rejects unknown/wrong type according to section definitions.
3. Static config field logs include source `xgoja.yaml`.
4. CLI override SectionValues merge over static values.
5. Missing CLI fields do not create overrides.
6. Final `SectionValuesToRawJSON` produces the JSON shape expected by existing `Module.New` decoders.

### 7.3 Runtime factory tests

1. `NewRuntimeFromSections` passes final merged config to `Module.New`.
2. `NewRuntime` keeps existing static config behavior when no parsed values are supplied.
3. A provider can expose CLI field `profile-registries` and internal config field `profileRegistries` with a mapping between them.
4. Same package selected twice under two aliases gets independent `SectionValues` instances and independent config JSON.
5. Static config is not mutated across runtime creations.

### 7.4 Geppetto tests

1. `xgoja.yaml` static `profileRegistries` parses through internal config section and reaches `decodeConfig`.
2. CLI/config/env `profile-registries` maps to internal `profileRegistries` and overrides static config.
3. `turns-dsn` / `turns-db` map into internal config and enable storage construction.
4. No `turns-dsn` / `turns-db` means storage remains disabled; no `enableStorage` gate is needed.
5. Removed allow-gate fields are not accepted in new config schema unless retained for migration explicitly.

---

## 8. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Internal config section accidentally exposed as CLI flags | Keep `ModuleConfigSectionCapability` separate from `ConfigSectionCapability`; xgoja must never add internal sections to command descriptions. |
| Static config field names and CLI field names diverge confusingly | Document public-to-internal mappings in provider code and tests. |
| Section defaults change existing provider behavior | Use `onlyProvided=true` for v1; do not synthesize internal defaults unless explicitly requested. |
| `SectionValues.Clone` shares field pointers | Fix Glazed clone or avoid it before relying on per-runtime mutation. |
| Provenance lost during public-to-internal mapping | Copy `FieldValue.Log` and append a mapping parse step. |
| Generic mapper becomes too magical | Keep v1 helper explicit: source section, source field, target field. |
| Nested config remains awkward | Prefer flat internal fields for v1; use object fields only when the provider truly needs nested objects. |

---

## 9. Decision records

### Decision: Use Glazed SectionValues as the module config merge representation

- **Context:** Earlier designs introduced JSON maps or custom patch wrappers, but Glazed already has typed, provenance-preserving parsed value objects.
- **Options considered:** `map[string]any`, `json.RawMessage`, custom `ModuleConfigPatch`, raw `values.Values`, `values.SectionValues`.
- **Decision:** Use `values.SectionValues` for internal module config state and overrides.
- **Rationale:** It reuses Glazed parsing, validation, provenance, and merge behavior while keeping config scoped to one section/module.
- **Consequences:** Providers must expose an internal config section; xgoja must convert final section values back to JSON for current `ModuleContext.Config`.
- **Status:** proposed.

### Decision: Keep public CLI sections separate from internal config sections

- **Context:** We do not want every module config knob to become a public flag.
- **Options considered:** Reuse one section for both config and flags; expose all config fields as flags; use separate sections and provider mapping.
- **Decision:** Separate public and internal sections.
- **Rationale:** Providers control what is user-facing and can keep internal config stable without creating flag clutter or unsafe knobs.
- **Consequences:** Provider must write or declare field mappings.
- **Status:** proposed.

### Decision: Add xgoja-local helpers first, then upstream generic Glazed helpers if useful

- **Context:** The mapping pattern is likely generally useful but should be proven in xgoja first.
- **Options considered:** Implement all helpers in go-go-goja; immediately modify Glazed; use patternmapper unchanged.
- **Decision:** Start in go-go-goja providerutil, with API shape suitable for later move to Glazed.
- **Rationale:** Reduces blast radius while preserving a path to reuse.
- **Consequences:** Some helper names may change when upstreamed.
- **Status:** proposed.

### Decision: Do not synthesize internal config defaults in v1

- **Context:** Existing providers already have code defaults inside `Module.New` and config decoding.
- **Options considered:** Always apply section defaults; never apply defaults; make it configurable.
- **Decision:** v1 parses only provided static config and provided command overrides.
- **Rationale:** Minimizes behavior changes and avoids defaults overriding provider code accidentally.
- **Consequences:** Providers that want schema defaults need an explicit opt-in later.
- **Status:** proposed.

---

## 10. Implementation checklist

- [ ] Fix or avoid `SectionValues.Clone` shallow-copy behavior.
- [ ] Add `ModuleConfigSectionCapability` to `providerapi`.
- [ ] Add `ModuleConfigValuesCapability` and request struct to `providerapi`.
- [ ] Add `providerutil.ParseModuleInstanceConfig`.
- [ ] Add `providerutil.CopyFieldIfProvided` / `MapFields`.
- [ ] Add `providerutil.SectionValuesToRawJSON`.
- [ ] Add `RuntimeFactory.NewRuntimeFromSections` that uses internal config sections when available.
- [ ] Preserve old `NewRuntime` behavior when no command values are supplied or no config capability exists.
- [ ] Update built-in command runtime creation paths.
- [ ] Add Geppetto internal config section and public CLI section mapping.
- [ ] Simplify Geppetto config around turn storage flags.
- [ ] Add tests across Glazed helper behavior, providerutil parsing, runtime factory merge, and Geppetto.

---

## 11. File reference

### Glazed files

- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/schema/schema.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/schema/section-impl.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/values/section-values.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/fields/field-value.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/fields/gather-fields.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/sources/update.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/sources/sections.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/sources/middlewares.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/sources/load-fields-from-config.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/sources/patternmapper/pattern_mapper_builder.go`

### go-go-goja files

- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/module.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/factory.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/root.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/run.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/tui.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerutil/sections.go`

### Geppetto and Pinocchio files

- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/provider/provider.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/pinocchio/cmd/pinocchio/cmds/js.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/pinocchio/cmd/pinocchio/doc/general/05-js-runner-scripts.md`
