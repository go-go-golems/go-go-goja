# 09-provider-shipped-help-docs

This example demonstrates provider-shipped Glazed help documents in a generated xgoja binary.

The generated binary imports the Loupedeck xgoja provider and selects its `runtime-api` help source:

```yaml
help:
  sources:
    - id: loupedeck-runtime-api
      package: loupedeck
      source: runtime-api
```

That provider source is registered by the Loupedeck repository from its embedded `docs/help` package. The generated binary therefore exposes the Loupedeck JavaScript API reference without copying the Markdown files into this example.

## Smoke test

```bash
make smoke
```

The smoke test validates the spec, builds the generated binary, and checks that these provider-shipped help topics render:

```bash
./dist/provider-shipped-help-docs help loupedeck-js-api-reference
./dist/provider-shipped-help-docs help loupedeck-js-first-live-script
```

This example intentionally enables only a minimal `eval` command and selects only `loupedeck/state` in the module set. The point is documentation bundling, not hardware access.
