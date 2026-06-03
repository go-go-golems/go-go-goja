---
Title: Time Module
Slug: time-module
Short: Monotonic timing helpers for JavaScript-side performance measurements
Topics:
- time
- modules
- goja
- javascript
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `time` module provides lightweight monotonic clock helpers. It is not a wall-clock date library; it is designed for benchmarking and measuring elapsed intervals inside a Goja runtime.

The counter starts when the module is first loaded into a runtime, so `now()` values are relative to that point.

## JavaScript usage

```javascript
const time = require("time");

const start = time.now();
for (let i = 0; i < 100000; i++) {
  Math.sqrt(i);
}
const elapsed = time.since(start);
console.log(`took ${elapsed} ms`);
```

## Module API

### `now()`

Returns the number of milliseconds elapsed since the module was initialized in the current runtime. The value is a monotonic float and never decreases.

### `since(startMs)`

Given a previous value returned by `now()`, returns the delta in milliseconds. Equivalent to `now() - startMs` but expressed as a helper.

## Design notes

`time` uses Go's `time.Since` over a baseline recorded at module load time. This avoids the overhead and non-monotonic behavior of wall-clock date APIs. If you need calendar dates, format strings, or time zones, use JavaScript's built-in `Date` object instead.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Negative `since()` result | The `startMs` argument came from a different runtime instance | Capture `now()` inside the same runtime before measuring |
| Very small `now()` values | The module was loaded seconds ago | This is expected; values are relative to module initialization |
