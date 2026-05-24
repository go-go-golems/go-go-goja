# xgoja provider-shipped jsverbs example

This example builds a generated xgoja binary that mounts JavaScript verbs shipped by a Go provider package.

Use this mode when a provider package owns default JS commands alongside its native `require()` modules.

## Run

```bash
make smoke
```

The smoke target:

1. validates `xgoja.yaml`,
2. builds `dist/provider-shipped-jsverbs` with a local `go-go-goja` replace,
3. runs `repl` against the fixture provider module,
4. runs a provider-shipped verb,
5. runs a provider-shipped async verb that exercises runtime-owner bindings.

Expected final lines:

```text
hello provider
hello provider
pong
```

## Source mode

The important part of `xgoja.yaml` is:

```yaml
jsverbs:
  - id: provider-defaults
    package: fixture
    source: verbs
```

The provider fixture exposes that source from:

```text
pkg/xgoja/testprovider/provider.go
pkg/xgoja/testprovider/verbs/tools.js
```
