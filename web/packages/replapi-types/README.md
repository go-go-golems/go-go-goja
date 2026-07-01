# go-go-goja replapi types

This package contains generated TypeScript protobuf bindings for the `goja.replapi.v1` HTTP payloads.

The generated files are produced by Buf from `proto/goja/replapi/v1/replapi.proto` and live under `src/generated/`. Hand-written code in this package is limited to an index barrel and decode smoke tests.

## Usage

```ts
import { fromJson } from "@bufbuild/protobuf";
import { EvaluateResponseSchema } from "@go-go-golems/go-go-goja-replapi-types";

const body = await fetch(`/api/v1/sessions/${sessionId}/evaluate`, {
	method: "POST",
	headers: { "Content-Type": "application/json" },
	body: JSON.stringify({ schemaVersion: 1, source: "1 + 2" }),
}).then((response) => response.json());

const decoded = fromJson(EvaluateResponseSchema, body);
console.log(decoded.cell?.execution?.status);
```

## int64 and JSON values

`int64` protobuf fields decode as JavaScript `bigint` values with `@bufbuild/protobuf` v2. Do not pass decoded messages with `bigint` fields directly to `JSON.stringify`; use protobuf JSON helpers such as `toJson()` when serializing wire payloads again.

Fields modeled as `google.protobuf.Value` decode to protobuf wrapper messages, but `toJson()` projects them back to ordinary JSON objects, arrays, strings, numbers, booleans, or `null`.
