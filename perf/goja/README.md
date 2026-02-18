# go-go-goja Goja Performance Benchmarks

This directory contains benchmark suites focused on `go-go-goja` runtime behavior and Goja execution boundaries.

## Scope

The current suites cover:

- runtime lifecycle cost (`goja.New`, `engine.NewWithConfig`)
- runtime lifecycle cost with reusable bootstrap (`engine.NewFactory(...).NewRuntime()`)
- repeated spawn + execute patterns
- JS loading cost (compile, run-string, run-program)
- JS -> Go call overhead (direct baseline vs `vm.Set` vs `modules.SetExport`)
- Go -> JS call overhead (direct baseline vs `goja.AssertFunction` vs `calllog.CallJSFunction`)
- `require()` loading behavior (cold runtime vs warm cached runtime)

## Run

```bash
go test ./perf/goja -run '^$' -bench . -benchmem -count=5 > /tmp/goja-perf-new.txt
```

Compare two runs:

```bash
benchstat /tmp/goja-perf-old.txt /tmp/goja-perf-new.txt
```

## Notes

- Benchmarks deliberately include call logging modes; `engine.New()` now defaults to call logging disabled, but explicit enabled-mode benchmarks are still important for opt-in deployments.
- For stable numbers, run on a quiet machine, pin CPU governor where possible, and keep Go/toolchain constant across runs.
