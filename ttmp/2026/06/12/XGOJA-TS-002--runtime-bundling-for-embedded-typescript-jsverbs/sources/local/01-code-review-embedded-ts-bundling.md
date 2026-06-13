---
Title: Code review note on embedded TypeScript jsverb bundling
Ticket: XGOJA-TS-002
Status: active
Topics:
  - goja
  - xgoja
  - typescript
  - tooling
  - developer-experience
DocType: reference
Intent: source
Summary: Captures the code review issue that triggered XGOJA-TS-002.
---

# Code review note on embedded TypeScript jsverb bundling

The code review comment was attached to `pkg/xgoja/app/typescript.go` at the runtime bundling path:

```go
if source.TypeScript.Bundle {
    artifact, err := tsscript.BundleVirtualEntry(compileSource, tsOptions)
```

Reviewer note:

> P2 Badge Support bundled imports for embedded TypeScript jsverbs
>
> When a TypeScript jsverb source is loaded via ScanFS (embedded jsverbs or provider-shipped sources), the scanned files have no filesystem ResolveDir, but this runtime path still asks esbuild to bundle the original source from input.ResolveDir. In that context a bundled verb containing import './helper' is discovered during scan but fails when invoked because esbuild cannot resolve the relative import from the embedded/provider fs.FS. Since embedded TypeScript descriptors are allowed/generated, either prebundle before embedding or provide an fs-backed resolver instead of using BundleVirtualEntry with an empty resolve directory.

The issue is valid because filesystem-backed scans set `ResolveDir`, but `ScanFS` currently does not. TypeScript source scanning can still discover a verb after `TransformSource`, but runtime bundling fails for local value imports when esbuild cannot read imported files from the embedded/provider `fs.FS`.
