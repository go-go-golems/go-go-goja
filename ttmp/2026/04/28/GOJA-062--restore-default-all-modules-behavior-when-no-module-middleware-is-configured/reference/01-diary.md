---
title: GOJA-062 Diary — restore default all-modules behavior
---

# Diary

## Issue

Review feedback pointed out that `FactoryBuilder.Build()` only evaluated the module middleware pipeline under:

```go
if len(b.moduleMiddlewares) > 0 { ... }
```

That meant a plain `engine.NewBuilder().Build()` appended no default-registry module specs. Because `Factory.NewRuntime()` still has the historical data-only baseline, these runtimes ended up with only safe/data modules. This broke the intended compatibility contract: no middleware/flags should expose all default modules, while `--safe-mode`/`MiddlewareSafe()` should be the restriction.

## Fix

Changed the build condition to evaluate the default-registry selector when either:

- module middlewares are present, or
- the caller supplied no explicit `WithModules(...)` specs.

This preserves three cases:

1. Plain `NewBuilder().Build()` => all default-registry modules.
2. `NewBuilder().UseModuleMiddleware(MiddlewareSafe()).Build()` => safe/data-only modules.
3. `NewBuilder().WithModules(DefaultRegistryModule("fs")).Build()` => explicit module composition remains explicit.

## Validation

Updated tests in `engine/granular_modules_test.go` to assert plain default builder exposes host modules (`fs`, `os`, `exec`, `database`, `db`) and safe middleware removes them.

Commands run:

```bash
go test ./engine ./cmd/goja-repl -count=1
golangci-lint run
go test ./...
```

All passed.
