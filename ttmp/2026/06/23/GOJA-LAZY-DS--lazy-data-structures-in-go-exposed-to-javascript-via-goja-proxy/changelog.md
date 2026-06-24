# Changelog

## 2026-06-23

- Initial workspace created


## 2026-06-23

Investigated lazy Go data structures over goja Proxy; confirmed NewProxy/ProxyTrapConfig API in local fork; found and fixed the Object.keys materialisation sharp edge via flag-only GetOwnPropertyDescriptor descriptors.

### Related Files

- /home/manuel/workspaces/2026-06-23/benchmark-goja-proxy-object/goja/builtin_proxy.go — ProxyTrapConfig + NewProxy


## 2026-06-23

Added runnable experiments (lazy map, infinite array, lazy kv object, cheap-enumeration) in explorations/lazyproxy and mirrored to ticket scripts/.

### Related Files

- /home/manuel/workspaces/2026-06-23/benchmark-goja-proxy-object/explorations/lazyproxy/cheap_enum.go — enumeration-free proof


## 2026-06-23

Uploaded bundle (design doc + diary + tasks) to reMarkable at /ai/2026/06/23/GOJA-LAZY-DS as a single ToC'd PDF.

### Related Files

- ttmp/2026/06/23/GOJA-LAZY-DS--lazy-data-structures-in-go-exposed-to-javascript-via-goja-proxy/design-doc/01-lazy-data-structures-over-goja-proxy-design-and-implementation-guide.md — primary deliverable uploaded


## 2026-06-23

Implemented reusable lazy module (modules/lazy): Map/Array/Record Go types + require('lazy') JS builders (map/array/record) + Mutable Set/Delete + tsgen; 14 tests passing; registered in default registry; runnable example.

### Related Files

- modules/lazy/lazy.go — core lazy Proxy types and flagOnlyDescriptor


## 2026-06-23

Added diary Step 4 (implementation record): registration-as-build-time-concern, Array length invariant, Record.has extra-set fix.

### Related Files

- modules/lazy/module.go — require('lazy') builders and JS<->Go provider adapters


## 2026-06-23

Re-uploaded bundle (v2: implemented) to reMarkable at /ai/2026/06/23/GOJA-LAZY-DS; design doc + diary (Step 4) + tasks.

### Related Files

- ttmp/2026/06/23/GOJA-LAZY-DS--lazy-data-structures-in-go-exposed-to-javascript-via-goja-proxy/reference/01-investigation-diary.md — diary Step 4 records the modules/lazy implementation


## 2026-06-23

Verified for..of on lazy Array works via inherited Array.prototype[Symbol.iterator] (no GetSym trap); locked in with TestArray_ForOfIsLazy + module test (commit 5bdc2e9). Async providers descoped per user.

### Related Files

- modules/lazy/lazy.go — Array docstring documents for..of support


## 2026-06-23

Closed ticket: design doc + diary (Steps 1-5) complete; lazy module implemented, tested, registered; bundle uploaded to reMarkable. Remaining items descoped or future work (infinite-stream type, Array own-property semantics).

