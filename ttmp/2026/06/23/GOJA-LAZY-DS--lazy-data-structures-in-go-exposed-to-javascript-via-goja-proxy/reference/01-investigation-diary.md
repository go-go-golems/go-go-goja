---
Title: "Investigation diary — GOJA-LAZY-DS"
Ticket: GOJA-LAZY-DS
Status: in-progress
DocType: reference
LastUpdated: 2026-06-23
---

# Diary

## Goal

Investigate how to build "lazy" Go data structures (on-demand maps, infinite
arrays, dynamic namespaces) and expose them to JavaScript as ordinary
objects/maps through goja's Proxy mechanism. Capture evidence and sharp edges so
an intern can implement a reusable `lazy` module confidently.

## Step 1: Orient and confirm the Proxy API in the local goja fork

The workspace `benchmark-goja-proxy-object` vendors three trees under a single
`go.work`: `goja` (a fork of dop251/goja), `go-go-goja` (the module/REPL layer),
and `glazed`. The decisive discovery is that the **Go-facing Proxy API is
already implemented in the local fork** and is the intended vehicle for this
work. The public entry points are:

- `Runtime.NewProxy(target *Object, nativeHandler *ProxyTrapConfig) Proxy`
- `ProxyTrapConfig` — a struct of optional Go callbacks, one per ES Proxy trap
- the internal `proxyHandler` Go interface in `proxy.go`, which lets traps be
  implemented purely in Go (no JS handler object required)

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to investigate building "lazy" data structures in go and expose them to javascript using goja (work within this workspace, all the necessary source has been provided). For example a map where a certain list only gets filled with say wifi networks on demand, when it is being read from JS. Or infinite streams or other datastructures one could figure being generated on demand from go, but appearing as an object or a map from the JS side."

**Assistant interpretation:** Stand up a docmgr ticket, build small runnable Go programs that prove goja Proxies can surface lazy Go-backed objects to JS, then write an intern-grade design/implementation guide and ship it to reMarkable.

**Inferred user intent:** The user wants a concrete, evidence-backed design (not speculation) for lazily-materialised Go data structures visible as normal JS objects, including the tricky behavioural details (enumeration, JSON, caching), with a clear path for a new engineer to implement it.

### What I did

- Read `goja/proxy.go` (the `proxyObject` + internal `proxyHandler` interface and all 13 trap methods) and `goja/builtin_proxy.go` (the public `ProxyTrapConfig`, `NewProxy`, and `nativeProxyHandler` adapter).
- Read `goja/builtin_proxy_test.go` for canonical Go-side usage patterns.
- Read `go-go-goja/modules/common.go` and `modules/fs/fs.go` to learn the `NativeModule` / `Loader` / `require()` registration pattern for the eventual reusable module.
- Read `go-go-goja/pkg/engine/factory.go` and `runtime_modules.go` for the `RuntimeFactoryBuilder` composition flow.

### Why

The whole design hinges on whether goja exposes traps to Go. It does — via
`ProxyTrapConfig` (high-level) and the internal `proxyHandler` interface
(low-level, used by `nativeProxyHandler`). That means a lazy data structure is
"just" a Go value behind a trap config.

### What worked

- The local fork's native-proxy tests pass: `go test -run 'TestProxy_native|TestProxy_Object_native' . -count=1` → `ok github.com/dop251/goja`.
- The public helpers needed (`NewProxy`, `ProxyTrapConfig`, `PropertyDescriptor`, `Set`, `RunString`, `ToValue`) all exist and are stable.

### What didn't work

- `go build ./...` in `go-go-goja` fails: the forked `goja` is older than the `goja_nodejs` revision go-go-goja pins, so `goja_nodejs/goutil` references `goja.IsNumber` / `goja.IsBigInt` / `goja.IsString`, which don't exist in the fork. Verbatim first errors:
  ```
  goja_nodejs@.../goutil/argtypes.go:12:10: undefined: goja.IsNumber
  goja_nodejs@.../goutil/argtypes.go:24:10: undefined: goja.IsNumber
  ...
  goja_nodejs@.../goutil/argtypes.go:76:10: undefined: goja.IsString
  ```
  This is a workspace-version skew, not a correctness problem for Proxy. I therefore ran all experiments against the local `goja` fork directly (see Step 2) rather than through go-go-goja.

### What I learned

- Two layers of trap API: high-level `ProxyTrapConfig` (Go callbacks keyed by `*Idx`/str/sym) and low-level `proxyHandler` interface. For new work, `ProxyTrapConfig` is the right abstraction.
- The fork builds cleanly on its own (`go build ./...` in `goja/` → exit 0). The skew only bites the go-go-goja integration layer.

### What was tricky to build

Nothing yet at this step; the trickiness arrives in Step 3 (enumeration). The
only sharp edge here is the version skew, whose cause is that `go.work` forces
the older `./goja` onto a `goja_nodejs` that expects newer symbols.

### What warrants a second pair of eyes

- Confirm the team's intent: is the workspace fork meant to be upgraded to a goja version that ships `IsNumber`/`IsString`/`IsBigInt`? If yes, the go-go-goja `require("lazy")` module (Step 9) can be built inside go-go-goja; if no, the reusable module must live where it can compile against the fork.

### What should be done in the future

- Resolve the `goja` ↔ `goja_nodejs` version skew so the reusable module can be wired through `go-go-goja`'s default registry.

### Code review instructions

- Start at `goja/builtin_proxy.go:351` (`NewProxy`) and `goja/builtin_proxy.go` (`ProxyTrapConfig` struct).
- Cross-reference `goja/proxy.go:88` (the `proxyHandler` interface) to see every trap.
- Validate: `cd goja && go test -run 'TestProxy' . -count=1`.

### Technical details

Public surface used throughout:

| API | File:line | Purpose |
| --- | --- | --- |
| `Runtime.NewProxy` | `goja/builtin_proxy.go:351` | create a Proxy from a target + Go trap config |
| `ProxyTrapConfig` | `goja/builtin_proxy.go` (doc comment above struct) | one optional Go func per ES trap |
| `Proxy` (Go type) | `goja/proxy.go:8` | opaque wrapper; passable back via `Runtime.ToValue` |
| `PropertyDescriptor` | `goja/object.go:58` | data/accessor descriptor returned by `GetOwnPropertyDescriptor` |
| `Flag` (`FLAG_TRUE`/`FALSE`/`NOT_SET`) | `goja/runtime.go:157` | tri-state for descriptor attributes |

---

## Step 2: Stand up an isolated, runnable exploration module

To get a clean compile against the fork (avoiding the go-go-goja skew), I added
`explorations/lazyproxy/` with its own `go.mod` and a `replace github.com/dop251/goja => ../../goja`. It is a single `main` package dispatched on `os.Args[1]` (`01`–`04`), with each experiment in its own file. The canonical copies also live in this ticket's `scripts/` directory (numbered `00-go.mod` … `06-cheap-enum.go`).

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Provide runnable proof programs and keep them in the ticket `scripts/` dir per the docmgr convention.

**Inferred user intent:** Reproducible evidence, not just prose.

### What I did

- Created `explorations/lazyproxy/{main.go,helpers.go,lazy_map.go,infinite_array.go,lazy_object.go,cheap_enum.go}`.
- `GOWORK=off go mod tidy` then `GOWORK=off go run . NN`.

### Why

`go.work` would otherwise pull in the broken go-go-goja integration; `GOWORK=off` + the local `replace` keeps the experiment self-contained and reproducible for an intern.

### What worked

`go mod tidy` downloaded the fork's deps (`dlclark/regexp2`, `golang.org/x/text`, `go-sourcemap/sourcemap`, `google/pprof`) and all four experiments compile and run.

### What didn't work

First compile attempt failed on `target.HasProperty(property)` — `*goja.Object` has no public `HasProperty` (only unexported `hasProperty`). Fix: fall back through the public `target.Get(property)` (which walks the prototype chain) in the `Has` trap.

### What I learned

- Use `target.Get(name)` / `target.GetSymbol(...)` (public) for target fallbacks; most `hasProperty`/`getOwnProperty` style helpers on `*Object` are unexported and exist only on `objectImpl`.

### What was tricky to build

The `Has` trap needs a prototype-chain-aware fallback but the natural `*Object.HasProperty` is unexported. Worked around with `target.Get`.

### What warrants a second pair of eyes

- The `Has` fallback via `target.Get` returns true for inherited props (e.g. `toString`). That is usually desirable but means `'toString' in lazy` is `true`.

### What should be done in the future

N/A.

### Code review instructions

- `cd explorations/lazyproxy && GOWORK=off go run . 01` (and `02`,`03`,`04`).
- Captured verbatim output: `scripts/outputs.txt`.

---

## Step 3: The enumeration sharp edge — and the cheap-descriptor fix

This is the central discovery of the investigation and the most important thing
for an intern to internalise.

A naive `GetOwnPropertyDescriptor` trap that calls the provider to build the
descriptor's `Value` makes **`Object.keys()` and `for...in` materialise every
value**, because enumeration calls `[[GetOwnProperty]]` for each key returned by
`ownKeys` in order to filter by enumerability.

Measured (experiment `01`, `03`): enumerating a 4-key object called the provider
for every not-yet-cached key (e.g. `provider called for: [battery uptime]`).

The fix (experiment `04`): make `GetOwnPropertyDescriptor` return a **flag-only
descriptor** — `Enumerable: FLAG_TRUE, Configurable: FLAG_TRUE`, no `Value`, no
provider call. Real values are produced only on the `Get` trap. Measured result:
`provider called for: []` for both `Object.keys` and `for...in`, while reads
still work and `JSON.stringify` still (unavoidably) materialises visited values.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Make sure the laziness holds under normal JS iteration, not just property reads.

**Inferred user intent:** A lazy map is only useful if enumerating it is cheap; prove the optimisation.

### What I did

- Ran `01` and `03` and observed the provider being hit during enumeration.
- Traced the cause in `goja/proxy.go`: `proxyObject.keys()` calls `p.val.getOwnProp(key)` for each key; `getOwnPropStr` invokes the `getOwnPropertyDescriptorStr` trap → my Go callback → the provider.
- Wrote `04` (flag-only descriptor) and confirmed enumeration is now free.

### Why

If enumeration materialised everything, lazy structures would be useless for
"list the keys without paying for the values" workflows (e.g. listing device
fields without scanning wifi).

### What worked

Flag-only descriptor:
```go
GetOwnPropertyDescriptor: func(t *goja.Object, p string) goja.PropertyDescriptor {
    return goja.PropertyDescriptor{Enumerable: goja.FLAG_TRUE, Configurable: goja.FLAG_TRUE}
}
```
Result (`04`): `Object.keys` → `[]`, `for..in` → `[]`, read → `[wifis]`.

### What didn't work

N/A — the fix worked first try once the root cause was traced.

### What I learned

- **Enumeration = ownKeys + getOwnPropertyDescriptor (flags only).** Keep `GetOwnPropertyDescriptor` side-effect-free.
- `JSON.stringify` is inherently value-materialising: it looks up `toJSON` first (a `Get` on the object for `"toJSON"`), then `Get`s each enumerable key. This is unavoidable; document it.
- `toValueProp` (`goja/builtin_object.go:114`) keeps a key enumerable based solely on the descriptor flags, irrespective of `value`, so a flag-only descriptor survives the `keys()` filter.

### What was tricky to build

The invariant reasoning: a flag-only descriptor returned to JS would read back as
`{enumerable:true, configurable:true}` (value omitted). Internally `complete()`
fills `value=undefined`, but because `Writable` ends up `FLAG_FALSE` the
non-fast-path `toValueProp` branch runs, yielding a `valueProperty` with
`enumerable=true` — so the key stays enumerable. Had `Writable` defaulted to
`TRUE` instead, the fast path `return resultDesc.Value` (undefined) would have
caused `keys()` to skip the key. The tri-state `Flag` + `complete()` interaction
is the trap.

### What warrants a second pair of eyes

- Re-confirm the `complete()`/fast-path interaction if the goja fork is upgraded (the relevant code is `goja/object.go:85` `toValue`, `object.go:124` `complete`, and `proxy.go` `keys()`).
- If a property genuinely should be non-enumerable, the `OwnKeys` list and the `GetOwnPropertyDescriptor` `Enumerable` flag must agree, or `Object.keys` will drop it.

### What should be done in the future

- Provide a helper `lazyDataProp(enumerable bool)` in the reusable module so authors don't hand-roll the flag-only descriptor and accidentally hit the fast-path skip.
- Consider a `Set` trap (and revocation) for mutable lazy stores; not exercised by these experiments.

### Code review instructions

- Compare `scripts/03-lazy-map.go` (`GetOwnPropertyDescriptor` calls provider → enumeration is O(n)) vs `scripts/06-cheap-enum.go` (flag-only → enumeration is free).
- Reproduce: `cd explorations/lazyproxy && GOWORK=off go run . 04`.

### Technical details

Proxy invariant checklist (from `goja/proxy.go`) that the empty-target pattern trivially satisfies:

- `ownKeys` must include every non-configurable target property — empty target ⇒ none.
- On a non-extensible target, `ownKeys` cannot introduce new keys — keep the target extensible (a fresh `vm.NewObject()` is extensible).
- `get`/`set`/`getOwnPropertyDescriptor` results must be compatible with any non-configurable target property — empty target ⇒ no constraint.

---

## Step 4: Implement the reusable `lazy` module (Go package + require("lazy"))

The design's Phase 0–3 are now implemented and committed inside
`go-go-goja/modules/lazy/`. The goja↔goja_nodejs version skew that blocked
go-go-goja in Step 1 has resolved (the workspace goja fork now exports
`IsNumber`/`IsBigInt`/`IsString` at `runtime.go:2534/2544/2550`), so the module
could be built and wired through the default registry like `fs`/`os`.

### Prompt Context

**User prompt (verbatim):** "Implement." / "Commit at appropriate intervals."

**Assistant interpretation:** Build the reusable `lazy` Go package and JS module
designed in the design doc, with tests, registration, and a runnable example; commit
in logical chunks.

**Inferred user intent:** Turn the design doc into working, tested code that an
intern can run and extend.

### What I did

- `modules/lazy/lazy.go` — core types `Map`, `Array`, `Record` with `Build(vm) goja.Proxy`
  and `ToValue(vm) goja.Value`; helpers `flagOnlyDescriptor`, `absentDescriptor`,
  `lazyGetStr`/`lazyGetInt`, `toValueAny`. `Map`/`Record` support `Mutable` (Set + deleteProperty traps).
- `modules/lazy/module.go` — `require("lazy")` NativeModule exposing `lazy.map`/`lazy.array`/`lazy.record`
  builders, with JS↔Go provider adapters.
- `modules/lazy/typescript.go` — `TypeScriptDeclarer` (verified via `cmd/gen-dts`).
- `modules/lazy/lazy_test.go` — 11 Go-API tests: laziness, caching, identity, **enumeration-free
  (the §6.3 regression)**, has-free, the descriptor-value-is-undefined tradeoff, JSON materialisation,
  spread/assign, mutable set/delete, infinite array, record.
- `modules/lazy/module_test.go` — 3 engine+require() integration tests.
- Registered `lazy` in the default registry (`pkg/engine/runtime.go`) and DTS generator (`cmd/gen-dts`).
- `examples/lazy/main.go` — runnable demo.

### Why

Deliver the concrete, tested artifact the design doc specified, in the proper home
(go-go-goja modules) rather than the throwaway exploration dir.

### What worked

- `go test ./modules/lazy/ ./pkg/engine/ -count=1` → all pass; `go build ./...` clean.
- `go run ./examples/lazy` → `{"keys":["name","wifis","battery"],"enumCalls":0,"wifiCount":3,"wifiCalls":1,"cached":true,"square1000":1000000,"length":1000000}` — proves enumeration is free, reads are cached, and a 10⁶-element lazy array reports length cheaply.
- `go run ./cmd/gen-dts --out /tmp/lazy.d.ts` → renders a valid `declare module "lazy"` block.

### What didn't work

- First `go run ./examples/lazy` failed with `Invalid module` for `require("lazy")`: the example
  binary didn't import `modules/lazy`, so its `init()` (default-registry registration) never ran.
  Fix: register `lazy` in `pkg/engine/runtime.go`'s blank-import list (where `fs`/`os`/etc. live),
  so any `engine`-using binary gets it. This is also why the in-package tests passed but the
  standalone example didn't.
- `Record.Mutable` read-back returned `undefined` after `Set`: `has()` only consulted `Has`/`Keys`,
  which don't know about keys added via `Set`. Fix: `has()` now checks the internal `extra` set first.
- Test `TestMap_GetOwnPropertyDescriptorValueIsUndefined` initially asserted a JSON shape with `"val":null`,
  but `JSON.stringify` omits `undefined` fields; switched to a direct `.Export()` map assertion.

### What I learned

- **Registration is a build-time concern.** A NativeModule's `init()` only runs in binaries that
  transitively import it; the canonical side-effect import lives in `pkg/engine/runtime.go`.
- **Proxy `Build` vs `ToValue`:** returning `goja.Proxy` (a `valueContainer`) from `Build` lets callers
  `.Revoke()`; `ToValue` is the `vm.ToValue(proxy)` convenience. Both work because `Proxy` implements
  the unexported `valueContainer.toValue` that `Runtime.ToValue` dispatches on.
- **Array target invariant:** returning an empty/absent descriptor for the non-configurable `length`
  property of an Array target would panic. The `Array` design avoids this by NOT installing
  `GetOwnPropertyDescriptor`/`OwnKeys` (indices are surfaced via `Get`/`GetIdx`/`HasIdx` only) and
  handling `length` purely in the `Get` trap. Documented as a known own-property caveat.

### What was tricky to build

- The `length`-on-Array-target invariant (above) — the naive "flag-only descriptor for every key"
  pattern from `Map` does NOT transfer to `Array` because the Array target owns a non-configurable
  `length`. Solved by leaving `GetOwnPropertyDescriptor`/`OwnKeys` unset for `Array`.
- Keeping `Record.has` consistent across `Keys()`/`Has`/`Set` without forcing callers to provide all three.

### What warrants a second pair of eyes

- `Array` own-property semantics: `idx in arr` is `true` but `Object.getOwnPropertyDescriptor(arr, idx)`
  is `undefined` (indices aren't own). Acceptable for v1 but worth confirming against consumer expectations.
- Mutable `Record`/`Map` mutate config (`Keys`/`extra`) on the JS goroutine; documented as not
  cross-runtime-shareable. Confirm no caller shares one across goroutines.
- The `getOwnPropertyDescriptor(k).value === undefined` tradeoff (§6) — confirm `Object.assign`/spread
  (which use `[[Get]]`) cover all real consumers; only explicit `getOwnPropertyDescriptor().value` is affected.

### What should be done in the future

- `Symbol.iterator` on `Array` for `for..of` without manual indexing.
- Go-level `Revoke()` ergonomics (already available via `Build().Revoke()`; could be surfaced in JS).
- Own-property semantics for `Array` indices if a consumer needs `getOwnPropertyDescriptor(arr, i)`.

### Code review instructions

- Start at `go-go-goja/modules/lazy/lazy.go` (`flagOnlyDescriptor`, `Map.Build`, `Record.has`).
- `cd go-go-goja && go test ./modules/lazy/ -count=1 -v` (esp. `TestMap_EnumerationIsFree`, `TestMap_GetOwnPropertyDescriptorValueIsUndefined`).
- `cd go-go-goja && go run ./examples/lazy`.

### Technical details

Module surface:

| Go type | JS builder | Traps installed |
| --- | --- | --- |
| `lazy.Map` | `lazy.map({keys,get,mutable?})` | Get, Has, GetOwnPropertyDescriptor, OwnKeys (+Set,Delete if Mutable) |
| `lazy.Array` | `lazy.array({length,get})` | Get, GetIdx, HasIdx |
| `lazy.Record` | `lazy.record({keys,get,has?,mutable?})` | Get, Has, GetOwnPropertyDescriptor, OwnKeys (+Set,Delete if Mutable) |

---

## Step 5: Symbol.iterator on lazy Array — and closing the ticket

The remaining follow-up was making `for...of` work on `lazy.Array`. A probe test
revealed it already worked: the inherited `Array.prototype[Symbol.iterator]`
reads `.length` (the `Get` trap → `a.Len`) and indexes (`GetIdx` → lazy value),
so no `GetSym` trap is required. Measured: `for (var x of arr)` over a 5-element
array produced `[0,10,20,30,40]` with the provider called exactly once per index,
and re-reading an index after iteration was free (shared cache).

### Prompt Context

**User prompt (verbatim):** "do the symbol iterator. no need for async providers. then close up the experiment ticket."

**Assistant interpretation:** Make `for...of` work on the lazy Array; async providers are explicitly descoped; then finalise and close ticket GOJA-LAZY-DS.

**Inferred user intent:** Complete the last functional gap (iteration) without scope-creep into async, then wrap the investigation up cleanly.

### What I did

- Probed whether the inherited Array iterator works through the proxy — it does.
- Locked it in with `TestArray_ForOfIsLazy` (Go API) and a `for..of` assertion in `TestLazyModule_Array`.
- Updated the `Array` docstring to state `for...of` is supported via inheritance and needs no `GetSym` trap.
- Committed (`5bdc2e9`).

### Why

Adding a `GetSym` trap that merely replicates the inherited iterator would be redundant code. The inherited iterator already produces the correct lazy behaviour because the Array target's prototype chain supplies `Symbol.iterator`, and the proxy's `Get`/`GetIdx` traps supply the length and values.

### What worked

`for...of` over `lazy.Array` iterates 0..Len-1, materialising each element at most once.

### What didn't work

N/A — the inherited path worked first try.

### What I learned

- A Proxy with an Array target inherits `Array.prototype[Symbol.iterator]` for free; `for...of` works whenever `length` and integer indexing are intercepted. No explicit iterator trap is needed for finite lazy arrays.
- `for...of` on a finite lazy array is inherently total: it materialises every element 0..Len-1. That is correct (iteration means reading all values) but means `for...of` on a Len=10⁶ array materialises 10⁶ values. Random access stays lazy; full iteration does not. Documented.

### What was tricky to build

Nothing implementation-side. The only subtlety is recognising that `for...of` does not need a `GetSym` trap, so the temptation to hand-roll an iterator was avoided.

### What warrants a second pair of eyes

- If a consumer needs an *infinite* stream (no finite length), the inherited iterator is the wrong tool — that would require a dedicated stream type with a `Symbol.iterator` whose `next()` has no termination. Out of scope here; noted as a separate future type.

### What should be done in the future

- (Descoped by user) Async/non-blocking providers — not needed now.
- A dedicated infinite-stream type with a terminating-condition iterator, if a consumer appears.
- Array index own-property semantics (`getOwnPropertyDescriptor(arr, i)`) remain a minor known limitation.

### Code review instructions

- `cd go-go-goja && go test ./modules/lazy/ -run ForOf -v`.
- `lazy.go` `Array` docstring; `lazy_test.go` `TestArray_ForOfIsLazy`.

### Technical details

`for..of` dispatch on a `lazy.Array` proxy:

```
for (const x of arr)
  [[Get]](Symbol.iterator)          -> no GetSym trap -> target -> Array.prototype[Symbol.iterator]
  iterator = Array.prototype[Symbol.iterator].call(proxy)
  iterator reads:
    Get(proxy, "length")             -> Get trap -> a.Len
    Get(proxy, i)                    -> GetIdx trap -> lazyGetInt(...)  (cached)
```

---

## Step 6: Alternative backends — DynamicObject vs Proxy (the 28× enumeration win)

The user asked whether lazy data structures can be implemented in goja by
means other than the ES6 Proxy. The answer is yes, and the strongest
alternative — goja's native `DynamicObject` — turns out to be dramatically
better than the Proxy for the common case. This step probed the alternatives,
implemented a `DynMap`/`DynArray` backend in `modules/lazy`, and benchmarked
it head-to-head against the Proxy-backed `Map`.

### Prompt Context

**User prompt (verbatim):** "are there other ways to implement lazy data structures in goja? besides the proxy?" → "yes, continue, keep a detailed diary. Create a detailed analysis / design / implementation guide that is for a new intern ... Store in the ticket and upload to remarkable."

**Assistant interpretation:** Identify and measure the non-Proxy lazy mechanisms, implement the best one (DynamicObject) as a real backend in the lazy module, benchmark it, and write a comprehensive intern guide.

**Inferred user intent:** Decide whether the lazy module's Proxy default is the right choice, with evidence; produce a durable, intern-grade guide covering the whole system.

### What I did

- Read `goja/object_dynamic.go`: the `DynamicObject` interface (`Get/Set/Has/Delete/Keys`) and `Runtime.NewDynamicObject`, plus its documented contract (all properties writable/enumerable/configurable; always extensible; **no invariant checks, no target** — "a lot more efficient" than Proxy).
- Probed enumeration behaviour with call-counting tests: `Object.keys`/`for..in` on a `DynamicObject` call `Get` **0 times** (enumeration uses `Keys()` alone); a single read calls `Get` once. Same probe for accessor getters: `Object.keys` calls the getter 0 times (JS-spec), a read calls it once.
- Implemented `modules/lazy/dyn.go`: `DynMap` and `DynArray` mirroring `Map`/`Array` (Keys/Provider/Mutable) but backed by `DynamicObject`/`DynamicArray`. `Get` returning `nil` delegates to the prototype automatically (no manual fallthrough). Read-only when `Mutable=false` via a `readOnlyDynMap` wrapper.
- Added `dyn_test.go` (6 tests: laziness, enumeration-free, has/inherited, mutable, read-only rejection, DynArray indexing/caching).
- Added `perf/goja/phase5_lazy_dyn_bench_test.go`: head-to-head Proxy vs DynamicObject on reads, enumeration, and scaling.

### Why

The Proxy's enumeration cost (15.8× eager, from Step 4's benchmark) is almost entirely the per-key descriptor construction the flag-only descriptor still pays. `DynamicObject` has no `GetOwnPropertyDescriptor` at all, so it cannot pay that cost. The question was whether that theoretical advantage shows up in measurement, and by how much.

### What worked

Measured (5 samples, 1s each), `DynamicObject` vs `Proxy`:

| operation (100 keys) | Proxy | DynamicObject | DynamicObject advantage |
| --- | ---: | ---: | ---: |
| single cached read | 234 ns / 2 allocs | 155 ns / 1 alloc | 1.51× faster, 1 fewer alloc |
| `Object.keys` | 141,129 ns / 1033 allocs | 4,996 ns / 108 allocs | **28.2× faster**, 9.6× fewer allocs |
| `Object.keys` N=1000 | ~3,986,709 ns (phase4) | 50,033 ns | ~80× faster |

`provider/op` is **0** for both backends on enumeration — the laziness is preserved. The differentiator is the per-key descriptor machinery, which the DynamicObject does not have. DynamicObject enumeration (4,996 ns/100 keys) is even cheaper than an eager object's `Object.keys` (37,874 ns in phase4), because `Keys()` returns the slice directly and the dynamic object reports all keys as enumerable without iterating a stored property table.

### What didn't work

- First accessor-getter probe failed: I tried to define accessor properties on a Go `*Object` after placing placeholder data properties via `DefineDataProperty`, which interacted badly. Fixed by building the object with getters entirely in JS (`Object.defineProperty` with `get`). The JS-spec result (0 getter calls during `Object.keys`) then held.
- No correctness failures in `DynMap` itself. The `DynamicObject.Get` contract ("may return nil if the property does not exist") made prototype delegation automatic, which is cleaner than the Proxy's manual `t.Get(p)` fallthrough.

### What I learned

- goja exposes **two** virtual-object mechanisms: the ES6 `Proxy` (full trap matrix + invariants + target) and the `DynamicObject`/`DynamicArray` (a fixed 5-/4-method Go interface, no invariants, no target). The latter is the right default for lazy maps/arrays.
- The §6 enumeration sharp edge is **specific to the Proxy**. It exists because `Object.keys` calls `[[GetOwnProperty]]` per key and the Proxy routes that through a trap that constructs a descriptor. `DynamicObject` has no `GetOwnPropertyDescriptor`, so the edge does not exist there at all — there is nothing to "fix" with a flag-only descriptor.
- `provider/op` remains the deterministic laziness signal across both backends (0 for enumeration, N for serialisation). The backend choice changes the *machinery* cost, not the *provider* cost.
- Accessor getters on a plain object are a third viable mechanism for the known-finite-keys case: lazy reads via the getter, free enumeration by JS spec, but no dynamic/unknown keys and no `in`/`delete` interception.

### What was tricky to build

- Enforcing read-only semantics on a `DynamicObject`: the interface requires `Set`/`Delete` methods, so a non-mutable `DynMap` cannot simply omit them. Solved with a `readOnlyDynMap` wrapper whose `Set`/`Delete` return `false`, which goja treats as a rejected write (in non-strict mode the assignment is silently ignored; in strict mode it throws). This is the correct way to make a `DynamicObject` read-only.
- Resisting the urge to switch `Map`'s default backend to `DynamicObject` unconditionally: the Proxy is still required when an ES6-only trap (`construct`, prototype interception, `defineProperty`, `Symbol` own props, per-key enumerability) is needed. The module now offers both; the guide gives the decision rule.

### What warrants a second pair of eyes

- `DynamicArray.Get` returns `nil` for out-of-bounds indices. goja treats a nil `Get` as an absent property (undefined), which matches array semantics, but confirm against `object_dynamic.go`'s `getIdx` path if a consumer relies on a specific out-of-bounds representation.
- The cross-phase comparison (DynamicObject N=1000 vs Proxy N=1000) draws the Proxy number from phase4 and the DynamicObject number from phase5; both are 5-sample 1s runs but taken at different times. The within-phase5 Proxy-vs-DynamicObject ratio (28.2×) is the robust number; the 80× cross-phase figure is indicative.
- `DynMap.Mutable=true` mutates `m.Keys` (a slice) on `Set`; as with `Map`, a `DynMap` configured once and `ToValue`'d per runtime must not be shared across goroutines while mutable.

### What should be done in the future

- Decide whether `lazy.Map` should default to the DynamicObject backend internally (keeping the Proxy behind a flag) — a breaking-ish API decision worth a separate discussion. For now both `Map` (Proxy) and `DynMap` (DynamicObject) are offered, and the guide recommends `DynMap` unless an ES6-only trap is needed.
- Consider a `lazy.Map` with a `Backend` field (`"dynamic"` | `"proxy"`) to unify the two under one type, if callers prefer a single entry point.
- A dedicated infinite-stream type (pull-based `Symbol.iterator`) remains future work; `DynamicObject` cannot own `Symbol` properties, but a prototype with `Symbol.iterator` can be set (per `object_dynamic.go` docs).

### Code review instructions

- Start at `modules/lazy/dyn.go` (`DynMap`, `dynMapHandler.Get`, `readOnlyDynMap`).
- `cd go-go-goja && go test ./modules/lazy/ -run Dyn -v`.
- `cd go-go-goja && go test ./perf/goja -run '^$' -bench LazyBackend -benchtime=1s -count=5 -benchmem`.
- The head-to-head report: `GOJA-LAZY-BENCH/various/06-lazy-report.html` (Figure 5).

### Technical details

Backend decision matrix:

| Need | Backend |
| --- | --- |
| Lazy map/record, cheap reads + free enumeration, no ES6-only traps | `DynMap` (DynamicObject) |
| Lazy array, indexed + length | `DynArray` (DynamicArray) |
| `construct`, prototype interception, `defineProperty` interception, `Symbol` own props, per-key enumerability | `Map`/`Array` (Proxy) |
| Known finite set of lazy keys, no dynamic keys | accessor getters on a plain object |
| On-demand compute, no object semantics | exported Go functions (`get(k)`) |
