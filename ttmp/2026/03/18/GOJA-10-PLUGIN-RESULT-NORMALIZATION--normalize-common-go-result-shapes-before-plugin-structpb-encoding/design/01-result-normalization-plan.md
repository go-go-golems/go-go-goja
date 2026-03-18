# GOJA-10 Result Normalization Plan

## Goal

Normalize common Go result shapes returned by SDK-authored plugin handlers before they are passed to `structpb.NewValue(...)`.

## Problem

The current `sdk.encodeResult(...)` implementation forwards handler results directly into protobuf `structpb` conversion. That works for values that are already in the exact shapes `structpb` expects, but it fails for ordinary Go values such as:

- `[]string`
- `[]int`
- `map[string]string`
- nested combinations such as `map[string]any{"tags": []string{"a", "b"}}`

That forces plugin authors to widen values manually into `[]any` and `map[string]any`, which leaks transport details into the author-facing SDK.

## Scope

This ticket is intentionally narrow.

In scope:

- normalize common typed slices into `[]any`
- normalize common `map[string]T` maps into `map[string]any`
- recurse through nested arrays/maps
- preserve `nil`, `string`, `bool`, numbers, `[]any`, `map[string]any`, and `*structpb.Value`
- keep unsupported shapes explicit with clear errors

Out of scope:

- arbitrary struct serialization
- callback/function transport
- contract changes
- doc-surfacing work

## Planned implementation

Add a small recursive normalizer in `pkg/hashiplugin/sdk/convert.go`.

Pseudocode:

```text
encodeResult(value):
  if value is *structpb.Value:
    return it directly
  normalized = normalizeResultValue(value)
  if normalized is nil:
    return null structpb value
  return structpb.NewValue(normalized)

normalizeResultValue(value):
  switch concrete type:
    nil/string/bool/float64/int/...:
      return structpb-safe scalar
    []any:
      normalize each element
    []string / []int / []float64 / []bool:
      widen to []any and normalize recursively
    map[string]any:
      normalize each value
    map[string]string / map[string]int / ...:
      widen to map[string]any and normalize recursively
    default:
      return unsupported-type error
```

## Validation plan

- unit tests in `pkg/hashiplugin/sdk/sdk_test.go` for:
  - `[]string`
  - `[]int`
  - `map[string]string`
  - nested maps + typed slices
  - unsupported map key types or function values
- focused package tests for `pkg/hashiplugin/sdk`
- full `go test ./... -count=1`

## Expected outcome

Plugin authors can return common nested Go values without manually widening them into protobuf-friendly shapes, while unsupported values still fail clearly.
