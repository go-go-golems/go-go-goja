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
