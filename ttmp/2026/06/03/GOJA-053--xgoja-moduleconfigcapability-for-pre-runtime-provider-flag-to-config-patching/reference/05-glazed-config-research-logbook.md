---
Title: Glazed Config and Flag Merge Research Logbook
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - glazed
    - config
    - flags
    - research
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: glazed/pkg/cmds/fields/field-value.go
      Note: |-
        FieldValue provenance logs and merge behavior.
        FieldValue resource evaluated for provenance and merge logs
    - Path: glazed/pkg/cmds/fields/gather-fields.go
      Note: Map-to-field parsing and validation behavior.
    - Path: glazed/pkg/cmds/schema/schema.go
      Note: |-
        Schema/Section interfaces and defaults initialization behavior.
        Schema/defaults resource evaluated for module config precedence
    - Path: glazed/pkg/cmds/schema/section-impl.go
      Note: |-
        Main SectionImpl evidence for static map parsing and Cobra flag parsing.
        SectionImpl resource evaluated for config-only and CLI section parsing
    - Path: glazed/pkg/cmds/sources/load-fields-from-config.go
      Note: |-
        Config file loading and mapper APIs.
        Config mapper resource evaluated as possible generic mapping pattern
    - Path: glazed/pkg/cmds/sources/update.go
      Note: |-
        FromMap, FromMapAsDefault, FromEnv source middlewares.
        Source middleware resource evaluated for map/env/default merge behavior
    - Path: glazed/pkg/cmds/values/section-values.go
      Note: |-
        SectionValues and Values merge/decode representation.
        SectionValues resource evaluated as ModuleConfigPatch replacement
ExternalSources: []
Summary: Research logbook for the Glazed-specific GOJA-053 pass, tracking which config/flags/merge resources were useful, stale, incomplete, or candidates for API updates.
LastUpdated: 2026-06-04T00:00:00Z
WhatFor: Use when continuing the Glazed-native module config design and deciding which Glazed APIs are trustworthy versus candidates for cleanup or extension.
WhenToUse: Before implementing xgoja ModuleConfigSection/SectionValues merge support or changing Glazed config/flag parsing helpers.
---


# Glazed Config and Flag Merge Research Logbook

## Purpose

This logbook records the resources consulted for the GOJA-053 Glazed-focused design pass. The goal was to determine whether xgoja can avoid a separate module config patch framework by reusing Glazed `schema.Section`, `values.SectionValues`, `values.Values`, source middlewares, and field provenance logs.

Each entry documents what was being researched, why the resource was chosen, how it was found, what was useful, what was not useful, what appears out of date or wrong, and what should be updated.

---

## Resource 1: `glazed/pkg/cmds/schema/section-impl.go`

### What I was researching

How a concrete Glazed section is constructed and how it parses values from maps and Cobra flags. This was central to deciding whether an xgoja module's hidden internal config can be represented as a normal Glazed `schema.Section`.

### What I was looking for in this document in particular

- How to create a section programmatically.
- Whether a section can parse a `map[string]interface{}` without going through Cobra.
- Whether prefixes are specific to CLI flag exposure.
- Whether `ParseSectionFromCobraCommand` only gathers explicitly provided flags.
- Whether section defaults are mixed into parsing implicitly.

### Why I chose it

`SectionImpl` is the default reusable section implementation. If this file did not support map parsing or clean construction, the whole SectionValues-based approach would be weaker.

### How I found the resource

Repository search for Glazed schema and parsing APIs:

- `rg --files pkg/cmds pkg/cli pkg/config pkg/middlewares`
- `rg -n "GatherFieldsFromMap|SectionValues|FromMap|FromDefaults|FromCobra" pkg/cmds pkg/cli pkg/config`

### What I found useful

- `NewSection(slug, name, options...)` gives a direct way to create internal module config sections.
- `WithFields(...)` and `WithPrefix(...)` show that the internal section can omit CLI prefixes while public CLI sections can have them.
- `ParseSectionFromCobraCommand` parses Cobra flags using `onlyProvided=true`, matching the desired behavior that flags override only when users supplied them.
- `GatherFieldsFromMap` delegates to definitions and can parse static module config without Cobra.

### What I didn't find useful

- Comments around `ParseSectionFromCobraCommand` include contradictory historical notes about toggling `onlyProvided` between false and true. The final code is useful; the comment history is noisy.
- The file does not provide a direct `SectionValuesFromMap` helper, so callers must assemble `FieldValues` and `SectionValues` themselves or go through `sources.FromMap`.

### What is out of date / what was wrong

- The comment saying parsing “will return a map containing the value (or default value) of each flag” is misleading for current behavior because `onlyProvided=true` means it should only override defaults with provided flags.
- `InitializeDefaultsFromFields` appears to call `p.GetDefinitions()` and initialize defaults on the returned definitions copy. If `GetDefinitions()` returns a fresh map of pointers, field pointer mutation may still work, but the semantics are subtle and should be reviewed.

### What would need updating

- Clarify `ParseSectionFromCobraCommand` comments to state current precedence behavior.
- Add or document a direct helper such as `values.NewSectionValuesFromMap(section, map, onlyProvided, opts...)` if this pattern becomes common.
- Add tests or comments around `InitializeDefaultsFromFields` mutation semantics.

---

## Resource 2: `glazed/pkg/cmds/schema/schema.go`

### What I was researching

How Glazed groups multiple sections and initializes defaults across a schema.

### What I was looking for in this document in particular

- Whether an internal module config section can be wrapped into a one-section `schema.Schema`.
- How defaults are initialized.
- Whether schema merge/clone behavior is simple enough for per-module-instance runtime use.

### Why I chose it

xgoja already has command-level schemas. The proposed internal module config section may need a one-section schema if we use source middlewares such as `sources.FromMap` or `sources.FromDefaults`.

### How I found the resource

It was linked by type references from `section-impl.go` and surfaced in `rg` results for `InitializeFromDefaults`, `UpdateWithDefaults`, and `Schema`.

### What I found useful

- `Section` is a small interface with `GetDefinitions`, `GetSlug`, `GetPrefix`, and `Clone`.
- `Schema.InitializeFromDefaults` and `UpdateWithDefaults` show the canonical way to populate defaults into `values.Values`.
- `InitializeSectionWithDefaults` uses `SetAsDefault`, which means defaults do not override existing values.

### What I didn't find useful

- The interface comment itself says the interface is messy. It does not explain which methods are required for non-Cobra internal sections versus CLI-visible sections.
- No direct guidance exists for “hidden config-only sections.”

### What is out of date / what was wrong

- The TODO about the Section interface being messy is still true and should be considered when adding more xgoja-facing capabilities.
- Schema defaults are available, but applying them to module config by default may be a behavior change for xgoja modules that currently rely on provider code defaults.

### What would need updating

- Add documentation or examples for config-only sections that are parsed from maps but not bound to Cobra.
- Consider documenting default precedence explicitly: defaults should be lowest precedence and opt-in for xgoja internal module config v1.

---

## Resource 3: `glazed/pkg/cmds/values/section-values.go`

### What I was researching

Whether `values.SectionValues` is a suitable replacement for a custom `ModuleConfigPatch` concept.

### What I was looking for in this document in particular

- Whether `SectionValues` stores a section pointer plus typed parsed fields.
- Whether it supports merging, cloning, decoding into structs, and conversion across sections.
- Whether `values.Values` can be used as the provider mapping input.

### Why I chose it

This file defines the core representation proposed by the new design. If merge or clone semantics were weak, xgoja would need more custom code.

### How I found the resource

It was explicitly named in earlier GOJA-053 discussion and appeared in search hits for `SectionValues`, `MergeFields`, and `DecodeInto`.

### What I found useful

- `SectionValues` is exactly “section + field values.”
- `MergeFields` delegates to `FieldValues.Merge`, so another section values object can override earlier values.
- `DecodeInto` allows providers to decode merged values into typed structs if needed.
- `Values.GetField(section, key)` gives provider mapping code access to the full `FieldValue`, including logs.
- `Values.Merge` exists for multi-section merge cases.

### What I didn't find useful

- There is no “map this source section into that target section” helper.
- `Values.Merge` contains two loops and appears more complex than needed for a simple override merge.
- The file does not expose an obvious `ToMap` helper on `SectionValues` itself; callers use `SectionValues.Fields.ToMap()` or `ToInterfaceMap()`.

### What is out of date / what was wrong

- `SectionValues.Clone` appears wrong or at least surprising: it first creates cloned fields, then overwrites them with original `FieldValue` pointers. That undermines safe per-runtime config isolation.
- The comment says clone creates a fresh fields map but not a deep copy of fields, while the initial `FieldValues.Merge` call suggests the implementation may have intended deeper cloning.

### What would need updating

- Fix `SectionValues.Clone` to use `Fields.Clone()` and add tests that mutations to a clone do not mutate the original.
- Add a `SectionValues.ToInterfaceMap()` convenience wrapper.
- Add mapping helpers if xgoja proves the pattern useful.

---

## Resource 4: `glazed/pkg/cmds/fields/field-value.go`

### What I was researching

How Glazed records provenance and how merge behavior preserves or loses source history.

### What I was looking for in this document in particular

- Whether a field value carries source metadata.
- What happens to logs when values are merged.
- Whether final field values can be converted to JSON-compatible maps.

### Why I chose it

The earlier design had concerns about default-vs-explicit provenance. If `FieldValue` already solves this, xgoja should not create its own provenance wrapper.

### How I found the resource

`SectionValues` stores `*fields.FieldValues`, and search results for `WithSource`, `ParseStep`, `Merge`, and `ToInterfaceMap` pointed here.

### What I found useful

- `FieldValue` stores `Value`, `Definition`, and `Log`.
- `Update` appends a parse step with source and metadata.
- `UpdateWithLog` can preserve a source field's existing log during provider mapping.
- `FieldValues.Merge` gives override precedence to the incoming field and preserves log history.
- `FieldValues.MergeAsDefault` provides “only if unset” semantics.
- `ToInterfaceMap` converts object/list/key-value values into JSON-friendly shapes.

### What I didn't find useful

- There is no first-class helper for “was this value explicitly provided versus default-only.”
- `FieldValue.Merge` and `FieldValues.Merge` have slightly different log behavior and should be reviewed before relying on fine-grained provenance display.

### What is out of date / what was wrong

- Several comments still include TODO/XXX notes about proper error handling or merge complexity.
- `FieldValue.Set` panics on invalid values instead of returning an error.

### What would need updating

- Add a helper like `FieldValue.HasSource(source string)` or `FieldValue.WasProvided(excludeDefaults bool)`.
- Add a documented provenance contract for merge operations.
- Prefer error-returning APIs over panic-based helpers for new xgoja paths.

---

## Resource 5: `glazed/pkg/cmds/fields/gather-fields.go`

### What I was researching

Whether static `xgoja.yaml` module config maps can be decoded into typed Glazed values.

### What I was looking for in this document in particular

- Map key lookup behavior.
- `onlyProvided` behavior.
- String parsing and type validation.
- Required/default behavior.
- Metadata attached to map values.

### Why I chose it

Static module config arrives as a map. This file determines whether we can parse that map directly through the internal config section.

### How I found the resource

Search hits for `GatherFieldsFromMap` from `section-impl.go`, `sources/update.go`, tests, and Lua integration.

### What I found useful

- The function validates values against field definitions.
- It supports `onlyProvided=true`, which is ideal for parsing static module config without synthesizing defaults.
- It looks up both full field names and short flags.
- It annotates parse metadata with the raw map value.
- String values are parsed using field parsers before validation.

### What I didn't find useful

- The comment says the returned map includes defaults for missing optional fields, but that is not true when `onlyProvided=true`.
- There is a TODO about nil semantics that matters for config patching and clearing values.

### What is out of date / what was wrong

- Documentation should be conditional on `onlyProvided`.
- Nil handling is unresolved and should not be relied on for “unset” or “clear” semantics in xgoja v1.

### What would need updating

- Clarify docs for `onlyProvided=true` versus `false`.
- Decide and test nil semantics for config loading before supporting explicit clearing.
- Consider exposing a section-level helper that returns `SectionValues` rather than only `FieldValues`.

---

## Resource 6: `glazed/pkg/cmds/sources/update.go`

### What I was researching

How Glazed sources merge maps, defaults, and environment variables into parsed values.

### What I was looking for in this document in particular

- `FromMap` behavior and precedence.
- `FromMapAsDefault` behavior.
- Environment variable source metadata and parsing.
- Whether these APIs are reusable for module config merge pipelines.

### Why I chose it

The user specifically asked about parsing/merging/patching richness in Glazed. This file contains core source middlewares.

### How I found the resource

Search for `FromMap`, `FromMapAsDefault`, `FromEnv`, and `updateFromMap`.

### What I found useful

- `FromMap` parses section maps through section definitions and merges them into parsed values.
- `FromMapAsDefault` uses `MergeAsDefault`, giving low-precedence source behavior.
- `FromEnv` uses field definitions to parse strings into typed values and adds `env_key` metadata.
- `updateFromMap` uses `onlyProvided=true`, matching the proposed static config parse behavior.

### What I didn't find useful

- `updateFromMap` is unexported, so xgoja can either use `sources.FromMap` with a schema or duplicate the small direct parse logic.
- There is no source middleware that maps already-parsed `values.Values` into another section while preserving field logs.

### What is out of date / what was wrong

- No obvious wrong behavior for the xgoja use case, but the API is oriented around raw maps/environment variables, not parsed-to-parsed mapping.

### What would need updating

- Consider adding a generic `FromMappedValues` or `MapValuesToSection` helper if the xgoja pattern is accepted.
- Document precedence examples for ad-hoc “static config + env/flags override” use cases.

---

## Resource 7: `glazed/pkg/cmds/sources/sections.go`

### What I was researching

Whether Glazed already has patch-like SectionValues merge middlewares.

### What I was looking for in this document in particular

- Existing APIs for replacing or merging a `SectionValues` into parsed values.
- Whether the name “patch” is already unnecessary because merge helpers exist.

### Why I chose it

Search results showed `MergeSectionValues`, `MergeValues`, and selective merge helpers.

### How I found the resource

Search for `MergeSectionValues` and `MergeValuesSelective` after discovering `SectionValues` merge methods.

### What I found useful

- `MergeSectionValues` already expresses “merge this section into that parsed section.”
- `MergeValues` and `MergeValuesSelective` already express multi-section override behavior.
- These helpers confirm that Glazed already models the patch concept as values merging.

### What I didn't find useful

- The middleware shape is less convenient than a direct helper for xgoja runtime factory code.
- It does not handle source-to-target field mapping.

### What is out of date / what was wrong

- No outdated behavior was identified.

### What would need updating

- Add examples showing these helpers as “config override” operations.
- If adding source-to-target mapping, place it near this file or in a new values mapping file.

---

## Resource 8: `glazed/pkg/cmds/sources/middlewares.go`

### What I was researching

Glazed middleware execution order and precedence semantics.

### What I was looking for in this document in particular

- Whether source middleware order can model defaults < config < env < flags.
- How middlewares should call `next` to get precedence right.

### Why I chose it

The design depends on Glazed ordering semantics rather than an xgoja-specific merge framework.

### How I found the resource

Search for `type Middleware`, `Execute`, and source chain comments.

### What I found useful

- The file explicitly documents that middlewares modifying parsed values should call `next` first.
- It gives a concrete example: defaults run first, then env, then command line arguments override.
- `ExecuteWithSchema` clones the schema, which helps avoid mutating command schemas during source processing.

### What I didn't find useful

- The wording about reverse ordering requires careful reading and can be confusing for new contributors.
- It does not include an intern-friendly precedence diagram.

### What is out of date / what was wrong

- The comment list still mentions Viper as checked, while other docs and code indicate Viper support has been removed/deprecated in favor of config files.

### What would need updating

- Remove or update stale Viper comments.
- Add a short precedence table or diagram to the docs.

---

## Resource 9: `glazed/pkg/cmds/sources/load-fields-from-config.go`

### What I was researching

How Glazed config file loading maps raw config files into section maps and preserves file/layer metadata.

### What I was looking for in this document in particular

- `ConfigFileMapper` shape.
- Whether custom mappers can target sections/fields.
- Whether config plan loading records provenance.
- How close this is to the xgoja “public values → internal config section” mapping problem.

### Why I chose it

The user asked whether Glazed has parsing/merging/patching richness that can be reused or generalized. Config mappers are a likely reusable pattern.

### How I found the resource

Search for `FromConfigPlan`, `ConfigFileMapper`, `WithConfigMapper`, and `WithParseOptions`.

### What I found useful

- `ConfigFileMapper` maps arbitrary raw config into `map[sectionSlug]map[fieldName]value`.
- `FromFiles` and `FromResolvedFiles` preserve metadata such as config file path, layer, source name, and source kind.
- `FromConfigPlanBuilder` can choose config plans dynamically based on parsed lower-precedence values.
- This confirms that Glazed already thinks in terms of source metadata and section-targeted config mapping.

### What I didn't find useful

- This maps raw config file structures, not already parsed `values.Values`.
- The output is a raw map, so it does not preserve `FieldValue.Log` from an existing parsed source.
- It is a good analogy but not the direct API needed by xgoja.

### What is out of date / what was wrong

- No wrong behavior found, but the API name `ConfigFileMapper` is too file-specific for the more general “section values mapper” concept.

### What would need updating

- Consider adding a sibling concept such as `ValuesMapper` or `SectionValuesMapper` for parsed-to-parsed mapping.
- Add docs comparing raw config mappers and parsed values mappers.

---

## Resource 10: `glazed/pkg/cmds/sources/config-mapper-interface.go`

### What I was researching

Whether Glazed already has an interface abstraction for config mapping that could be extended.

### What I was looking for in this document in particular

- Minimal interface shape.
- Whether function adapters already exist.

### Why I chose it

The design may eventually move providerutil mapping helpers upstream into Glazed.

### How I found the resource

The file was listed by `rg --files` and referenced from `load-fields-from-config.go`.

### What I found useful

- `ConfigMapper` is small: `Map(rawConfig interface{}) (map[string]map[string]interface{}, error)`.
- `ConfigFileMapper` function adapter already satisfies it.

### What I didn't find useful

- The raw `interface{}` input and raw section-map output are not provenance-preserving.
- It does not operate on `values.Values`.

### What is out of date / what was wrong

- Nothing identified as wrong; it is just solving a different layer of the problem.

### What would need updating

- If Glazed adds parsed-value mapping, it should not overload this interface. Add a separate interface with `*values.Values` input and `*values.SectionValues` or `*values.Values` output.

---

## Resource 11: `glazed/pkg/cmds/sources/patternmapper/pattern_mapper.go`

### What I was researching

Whether the existing pattern mapper can be reused to map public CLI/config values into internal module config values.

### What I was looking for in this document in particular

- Rule structure.
- Validation of target sections and fields.
- Collision detection and multi-match behavior.
- Whether it can preserve provenance.

### Why I chose it

A declarative mapping layer could be useful for provider public-to-internal section mapping, especially when field names differ.

### How I found the resource

Search results for `ConfigMapper`, `patternmapper`, and mapping rules.

### What I found useful

- `MappingRule` has a clear source path, target section, target field model.
- It validates target sections and fields early.
- It detects collisions when multiple patterns write the same field.
- It has useful error handling for required patterns.

### What I didn't find useful

- It maps raw config tree paths, not Glazed `values.Values` and `FieldValue` objects.
- It returns raw maps, so source `ParseStep` logs would be lost.
- It is heavier than needed for the first xgoja implementation.

### What is out of date / what was wrong

- Comments mention “Proposal” items, suggesting the file contains design-era notes that might not be polished production documentation.

### What would need updating

- Use it as inspiration only for v1.
- If a generic `SectionValuesMapper` is added, borrow validation and collision-detection ideas from patternmapper but preserve `FieldValue.Log`.

---

## Resource 12: `glazed/pkg/cmds/sources/patternmapper/pattern_mapper_builder.go`

### What I was researching

Whether there is already a fluent mapping API that could be adapted for provider field mapping.

### What I was looking for in this document in particular

- Builder API shape.
- Simplicity of mapping declarations.

### Why I chose it

Provider code should ideally be concise and readable when mapping CLI fields to internal config fields.

### How I found the resource

It was listed alongside patternmapper files and surfaced in search results for builder APIs.

### What I found useful

- The builder shape is easy to read: `Map(source, targetSection, targetField)` and `MapObject(...)`.
- It validates rules through `NewConfigMapper`.

### What I didn't find useful

- It is tied to raw config patterns and target sections, not parsed section values.
- It does not include options about defaults/provided-only source values.

### What is out of date / what was wrong

- No wrong behavior identified.

### What would need updating

- A future `SectionValuesMapperBuilder` could mirror this style:
  `Map("geppetto", "profile-registries", "profileRegistries")`.

---

## Resource 13: `glazed/pkg/cmds/runner/run.go`

### What I was researching

How higher-level command parsing composes Glazed source middlewares.

### What I was looking for in this document in particular

- Normal ordering of additional middleware, env, config files, provided values, and defaults.
- Whether Glazed already exposes a convenient command parse entry point.

### Why I chose it

xgoja can align its module config merge pipeline with the normal Glazed command parse pipeline.

### How I found the resource

Search hits for `ParseCommandValues`, `WithValuesForSections`, and source middleware composition.

### What I found useful

- `ParseCommandValues` composes additional middlewares, env, config files, provided section values, then defaults.
- It confirms that Glazed already treats parsed values as a middleware-produced object rather than a one-shot parser result.

### What I didn't find useful

- It does not include Cobra parsing directly in the excerpt inspected, so the lower-level xgoja command path remains more relevant.
- It is command-oriented and not specific to module initialization timing.

### What is out of date / what was wrong

- The file contains comments noting Viper support removal, which is useful context. No wrong behavior found.

### What would need updating

- Add a library example for parsing static maps plus overrides into a section without running a command, if this becomes a common pattern.

---

## Resource 14: `glazed/pkg/cli/cobra-parser.go`

### What I was researching

How the high-level Cobra parser wires config plans, env, Cobra, and defaults.

### What I was looking for in this document in particular

- Existing command parse pipeline order.
- How config plan builders are integrated.
- Whether xgoja could reuse parser configuration instead of adding a separate flow.

### Why I chose it

xgoja generated commands already use Cobra/Glazed. Understanding this file helps keep the module config design compatible with generated CLI behavior.

### How I found the resource

It appeared in search results for `FromCobra`, `FromEnv`, `FromConfigPlanBuilder`, and `FromDefaults`.

### What I found useful

- It shows that Glazed config/env/flags are already layered through source middlewares.
- It confirms command-level values should remain the input to provider mapping rather than adding a parallel CLI flag parser in xgoja.

### What I didn't find useful

- The file is broad and contains multiple parser paths, so it is less direct than the lower-level source middleware files.

### What is out of date / what was wrong

- No concrete wrong behavior identified during this pass.

### What would need updating

- If xgoja adds the internal module config section concept, this file probably needs no immediate changes unless generated command parsing bypasses required parsed values.

---

## Resource 15: `go-go-goja/pkg/xgoja/app/factory.go`

### What I was researching

Where final module config bytes are produced before `Module.New`.

### What I was looking for in this document in particular

- Runtime factory insertion point for SectionValues merging.
- Whether config is currently marshaled directly from `ModuleInstance.Config`.
- Whether command parsed values can reach runtime construction.

### Why I chose it

The Glazed design must be integrated at the point where xgoja creates runtime modules.

### How I found the resource

It was a known relevant xgoja file from the earlier GOJA-053 review and listed in the compacted context.

### What I found useful

- It remains the correct location to compute final module config before `Module.New`.
- It suggests that the low-level Goja engine does not need to know about Glazed or provider mapping.

### What I didn't find useful

- Current code is not designed around precomputed config bytes or per-module config SectionValues.

### What is out of date / what was wrong

- Not wrong, but current direct JSON marshaling of static config is insufficient for CLI/env/config overrides before module initialization.

### What would need updating

- Add runtime factory support for parsed command values.
- Add internal module config section lookup and final config JSON generation.
- Preserve existing behavior when providers do not implement the new capability.

---

## Resource 16: `go-go-goja/pkg/xgoja/providerapi/capabilities.go`

### What I was researching

Where provider capability APIs should be added.

### What I was looking for in this document in particular

- Existing pattern for optional provider capabilities.
- Naming and package boundaries for config/section capabilities.

### Why I chose it

The design adds capability interfaces rather than hard-coding Geppetto behavior.

### How I found the resource

It was a known relevant xgoja file from the earlier design review and listed in the compacted context.

### What I found useful

- Existing capability style supports optional provider participation.
- The file is the right home for `ModuleConfigSectionCapability` and `ModuleConfigValuesCapability`.

### What I didn't find useful

- Existing capability APIs do not yet express pre-`Module.New` config mapping.

### What is out of date / what was wrong

- Not out of date; simply missing the new concept.

### What would need updating

- Add `ModuleConfigSectionCapability`.
- Add `ModuleConfigValuesCapability` and request struct.
- Document that this uses Glazed SectionValues, not a custom patch format.

---

## Resource 17: `go-go-goja/pkg/xgoja/providerutil/sections.go`

### What I was researching

Whether xgoja already has provider helper functions for section creation or conversion.

### What I was looking for in this document in particular

- Existing providerutil style.
- Potential home for xgoja-local mapping helpers before upstreaming to Glazed.

### Why I chose it

The design recommends adding helper functions in `providerutil` first.

### How I found the resource

It was listed in the compacted context and was already reviewed in the first GOJA-053 design pass.

### What I found useful

- `providerutil` is a reasonable package for provider-facing section helpers.
- It can host `ParseModuleInstanceConfig`, `CopyFieldIfProvided`, `MapFields`, and `SectionValuesToRawJSON` without immediately changing Glazed.

### What I didn't find useful

- Existing helpers do not already cover parsed values mapping.

### What is out of date / what was wrong

- No wrong behavior identified.

### What would need updating

- Add Glazed-native module config mapping helpers.
- Keep helper APIs generic enough to move to Glazed later.

---

## Resource 18: `geppetto/pkg/js/modules/geppetto/provider/provider.go`

### What I was researching

How Geppetto currently exposes provider sections and decodes module config.

### What I was looking for in this document in particular

- Existing provider section fields.
- Current config structure and default behavior.
- Which fields should be public CLI flags versus internal module config.

### Why I chose it

Geppetto is the motivating provider for GOJA-053.

### How I found the resource

It was a known relevant provider from the ticket context and previous analysis.

### What I found useful

- It provides the concrete case for separating public CLI sections from internal config sections.
- It confirms the need to configure profile registries and turn storage before module initialization.

### What I didn't find useful

- Current config shape includes concepts that the revised design intentionally removes or simplifies.

### What is out of date / what was wrong

- Broad allow-gate fields such as `allowRegistryLoad`, `allowNetwork`, `allowTools`, `enableStorage`, and nested `turns` should be considered superseded for this capability design unless another security design requires them.

### What would need updating

- Add an internal config section.
- Add a separate public CLI/config/env section.
- Map public flags like `turns-dsn` / `turns-db` into internal config fields.
- Keep storage opt-in based on explicit storage fields rather than a broad `enableStorage` gate.

---

## Resource 19: `pinocchio/cmd/pinocchio/cmds/js.go`

### What I was researching

How Pinocchio exposes JS runner flags for Geppetto/profile/turn storage behavior.

### What I was looking for in this document in particular

- Existing flag names and ergonomics.
- Evidence for explicit turn-store flags.
- User-facing naming conventions to reuse in xgoja provider sections.

### Why I chose it

The revised design should follow working Pinocchio conventions instead of inventing different names.

### How I found the resource

It was listed in the compacted context and was part of the earlier GOJA-053 review.

### What I found useful

- It supports the recommendation to use explicit `turns-dsn` / `turns-db`-style flags.
- It helps distinguish user-facing flags from internal module config fields.

### What I didn't find useful

- It is app-specific and not directly reusable as a Glazed mapping API.

### What is out of date / what was wrong

- No wrong behavior identified for this pass.

### What would need updating

- Use Pinocchio naming as a compatibility/reference point when adding Geppetto xgoja CLI sections.

---

## Resource 20: `pinocchio/cmd/pinocchio/doc/general/05-js-runner-scripts.md`

### What I was researching

User-facing documentation for JS runner flags and turn storage behavior.

### What I was looking for in this document in particular

- Whether documentation supports explicit turn-store configuration.
- What an end user expects from flags/config.

### Why I chose it

The xgoja generated runtime should feel aligned with existing Pinocchio JS runner behavior.

### How I found the resource

It was listed in the compacted context and reviewed in the earlier GOJA-053 pass.

### What I found useful

- It provides a user-facing comparison point for how `turns-dsn` / `turns-db` should be documented.

### What I didn't find useful

- It does not explain the internal Glazed `SectionValues` merge mechanics.

### What is out of date / what was wrong

- No concrete wrong behavior identified during this pass.

### What would need updating

- After xgoja implements equivalent flags, add generated-runtime documentation that mirrors or references the Pinocchio behavior.

---

## Resource 21: GitHub issue `go-go-goja#52`

### What I was researching

The originating product and implementation requirement for GOJA-053.

### What I was looking for in this document in particular

- Whether the requested behavior requires per-module-instance config.
- Whether provider-owned command paths and generated runtime initialization are in scope.
- Whether the issue implies a custom config framework or simply pre-runtime config injection.

### Why I chose it

It is the source of the requested change and guards against designing a solution that solves the wrong problem.

### How I found the resource

It was referenced in the existing GOJA-053 ticket and earlier analysis context.

### What I found useful

- It anchors the requirement that flags/config must affect module initialization, not only runtime initializers after `Module.New`.
- It reinforces that solution scope is xgoja/provider integration, not only Geppetto application code.

### What I didn't find useful

- It does not prescribe a Glazed-specific API shape.

### What is out of date / what was wrong

- Earlier interpretations that implied a separate `ModuleConfigPatch` framework are superseded by this Glazed-native research pass.

### What would need updating

- Link the issue to the new SectionValues design when implementation begins.
- Note explicitly that the proposed implementation uses Glazed `SectionValues` rather than a bespoke patch object.

---

## Overall findings

### Most useful resources

1. `section-values.go` — confirms `SectionValues` can replace a custom patch representation.
2. `field-value.go` — confirms provenance lives in `FieldValue.Log`.
3. `update.go` — confirms static maps and source precedence are already Glazed concepts.
4. `section-impl.go` — confirms internal config sections can parse maps and public sections can parse Cobra flags.
5. `load-fields-from-config.go` and patternmapper — useful analogies for future generic mapping APIs.

### Most important stale or risky findings

- `SectionValues.Clone` likely needs fixing before xgoja relies on clone isolation.
- Comments around `ParseSectionFromCobraCommand` and `GatherFieldsFromMap` are misleading about defaults when `onlyProvided=true`.
- No existing helper cleanly maps `values.Values` into a target `SectionValues` while preserving logs.
- Viper-related comments remain in middleware docs despite migration away from Viper.

### Recommended updates before or during implementation

- Add xgoja-local `providerutil` helpers for static config parsing and public-to-internal field mapping.
- Fix or avoid `SectionValues.Clone` shallow-copy behavior.
- Add tests around provenance-preserving mapping.
- Document internal config section versus public CLI section distinction.
- If the pattern stabilizes, upstream a generic `SectionValuesMapper` or `MapValuesToSection` helper to Glazed.
