# Changelog

## 2026-06-23

- Initial workspace created


## 2026-06-23

Added phase3_bindings_bench_test.go comparing all goja Go->JS binding mechanisms (native func, wrapped func, SetExport, reflect struct, map, DynamicObject, DynamicArray, ES6 Proxy). Measured Proxy Get trap at ~199ns/48B/2 allocs (~1.6x a plain object, +1 alloc); DynamicObject matches a plain object (125ns/1 alloc).

### Related Files

- /home/manuel/workspaces/2026-06-23/benchmark-goja-proxy-object/go-go-goja/perf/goja/phase3_bindings_bench_test.go — binding benchmark suite


## 2026-06-23

Fixed workspace build: bumped go.work to go 1.26.1 (go work use) and added IsNumber/IsBigInt/IsString to the local goja fork so goja_nodejs/goutil compiles.

### Related Files

- /home/manuel/workspaces/2026-06-23/benchmark-goja-proxy-object/goja/runtime.go — compatibility shim for goja_nodejs


## 2026-06-23

Uploaded design doc + diary bundle to reMarkable at /ai/2026/06/23/2026-06-23-goja-bindings-benchmark.


## 2026-06-23

Built self-contained HTML report pipeline (scripts 03-07): stats parser, pure-function SVG charts, @@token@@ template, driver, inkscape figure renderer. Report at various/06-bindings-report.html. Design: deterministic allocs as load-bearing signal, noisy ns as corroboration; one question per chart.

### Related Files

- /home/manuel/workspaces/2026-06-23/benchmark-goja-proxy-object/go-go-goja/ttmp/2026/06/23/2026-06-23-goja-bindings-benchmark--benchmark-goja-go-to-js-binding-mechanisms-native-func-reflect-dynamicobject-es6-proxy/scripts/06-make-report.py — report driver


## 2026-06-23

Published two textbook-style articles to go-go-parc Obsidian vault and pushed to origin/main (commits bd85144, b701288): binding-mechanisms analysis + the report-generation system. 4 figures + 3 report screenshots embedded.


## 2026-06-23

Ticket closed: benchmark suite, intern guide, reMarkable bundle, self-contained HTML report + figure pipeline, and two Obsidian vault articles all delivered. All tasks complete.

