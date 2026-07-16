# go-go-goja replapi types

This package contains generated TypeScript protobuf bindings for the `goja.replapi.v1` HTTP payloads.

The generated files are produced by Buf from `proto/goja/replapi/v1/replapi.proto` and live under `src/generated/`. Hand-written code in this package is limited to an index barrel and decode smoke tests.

## Usage

```ts
import { fromJson } from "@bufbuild/protobuf";
import { ErrorResponseSchema, EvaluateResponseSchema } from "replapi-types";

const response = await fetch(`/api/sessions/${sessionId}/evaluate`, {
	method: "POST",
	headers: {
		"Content-Type": "application/json",
		"X-Request-ID": crypto.randomUUID(),
	},
	body: JSON.stringify({ schemaVersion: 1, source: "1 + 2" }),
});
const body = await response.json();
if (!response.ok) {
	const failure = fromJson(ErrorResponseSchema, body);
	throw new Error(`${failure.code} (${failure.requestId}): ${failure.message}`);
}

const decoded = fromJson(EvaluateResponseSchema, body);
console.log(decoded.cell?.execution?.status);
```

## Validation and publishing

```bash
pnpm replapi-types:typecheck
pnpm replapi-types:test
pnpm replapi-types:build
pnpm replapi-types:pack-smoke
pnpm replapi-types:consumer-smoke
```

The package is published from `dist/`, not from the source package root. The GitHub Actions workflow `.github/workflows/publish-npm.yml` is designed for npm Trusted Publishing: it requests `id-token: write`, uses the `npm-production` environment, does not pass an npm token, and publishes with provenance on real publishes.

For the first publication of a new npm package, create the package once manually or through an authorized bootstrap path, then configure trusted publishing:

```bash
npx -y npm@latest trust github replapi-types \
  --repo go-go-golems/go-go-goja \
  --file publish-npm.yml \
  --env npm-production \
  --allow-publish
```

After a tokenless GitHub Actions publish has been verified under `next`, package settings can be hardened to require 2FA and disallow token publishing.

## int64 and JSON values

`int64` protobuf fields decode as JavaScript `bigint` values with `@bufbuild/protobuf` v2. Do not pass decoded messages with `bigint` fields directly to `JSON.stringify`; use protobuf JSON helpers such as `toJson()` when serializing wire payloads again.

Fields modeled as `google.protobuf.Value` decode to protobuf wrapper messages, but `toJson()` projects them back to ordinary JSON objects, arrays, strings, numbers, booleans, or `null`.
