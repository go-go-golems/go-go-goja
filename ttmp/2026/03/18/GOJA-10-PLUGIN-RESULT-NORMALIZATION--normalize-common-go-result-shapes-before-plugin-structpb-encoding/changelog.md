# Changelog

## 2026-03-18

Created GOJA-10 to address the remaining SDK ergonomics gap around result encoding. The ticket is intentionally narrow: normalize common Go slice/map shapes before `structpb` conversion without expanding the transport contract or introducing a reflection-heavy serializer.

## 2026-03-18

Implemented the normalization slice in `pkg/hashiplugin/sdk/convert.go` and `pkg/hashiplugin/sdk/sdk_test.go`. The SDK now rewrites common typed slices, arrays, pointers, interfaces, and `map[string]T` values into `structpb`-friendly container trees before encoding, while still preserving `*structpb.Value` as the explicit escape hatch and rejecting unsupported shapes such as non-string map keys and function values.

## 2026-03-18

Closed out GOJA-10 by repairing ticket frontmatter, normalizing the ticket index metadata to the local doc vocabulary, rerunning `docmgr doctor` to a clean pass, and uploading the final bundle to `/ai/2026/03/18/GOJA-10-PLUGIN-RESULT-NORMALIZATION` on reMarkable.

## 2026-03-18

- Initial workspace created
