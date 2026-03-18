# Changelog

## 2026-03-18

Created GOJA-10 to address the remaining SDK ergonomics gap around result encoding. The ticket is intentionally narrow: normalize common Go slice/map shapes before `structpb` conversion without expanding the transport contract or introducing a reflection-heavy serializer.

## 2026-03-18

Implemented the normalization slice in `pkg/hashiplugin/sdk/convert.go` and `pkg/hashiplugin/sdk/sdk_test.go`. The SDK now rewrites common typed slices, arrays, pointers, interfaces, and `map[string]T` values into `structpb`-friendly container trees before encoding, while still preserving `*structpb.Value` as the explicit escape hatch and rejecting unsupported shapes such as non-string map keys and function values.

## 2026-03-18

- Initial workspace created
