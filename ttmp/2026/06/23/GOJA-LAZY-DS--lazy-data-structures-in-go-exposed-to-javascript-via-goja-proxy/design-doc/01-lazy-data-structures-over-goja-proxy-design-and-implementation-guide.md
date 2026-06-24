---
Title: "Lazy data structures over goja Proxy — design & implementation guide"
Ticket: GOJA-LAZY-DS
Status: draft
DocType: design-doc
Intent: short-term
Topics:
  - goja
  - js-bindings
  - javascript
  - go
  - architecture
  - patterns
  - modules
Owners: []
RelatedFiles:
  - ../../../../../../../goja/proxy.go
  - ../../../../../../../goja/builtin_proxy.go
  - ../../../../../../../goja/object.go
  - ../../../../../../../goja/builtin_object.go
Summary: >
  How to expose lazily-materialised Go data (on-demand maps, infinite arrays,
  dynamic key/value namespaces) to JavaScript as ordinary objects/maps through
  goja's Proxy, with intern-grade explanations, trap-by-trap guidance, the
  enumeration sharp edge and its fix, API/file references, and a phased build
  plan for a reusable `lazy` module.
LastUpdated: 2026-06-23
---

# Lazy data structures over goja Proxy — design & implementation guide

> Audience: a new intern who knows Go and JavaScript but has not used goja or
> ES Proxies before. Read top to bottom; every claim is anchored to a file.

## 0. TL;DR (executive summary)

- **Goal.** Let JavaScript read a Go object that looks ordinary (`device.wifis`,
  `rows[1_000_000]`, `env.HOME`) but whose values are produced *on demand* by Go
  — a wifi scan runs only when JS actually reads `device.wifis`; an "infinite"
  array computes element *n* only when JS indexes it.
- **Mechanism.** goja ships ES **Proxy** support with a **Go-facing** trap API:
  `vm.NewProxy(target, &goja.ProxyTrapConfig{...})`. You implement ordinary Go
  funcs (the "traps") and goja calls them as JS touches the object. No JS
  handler object required.
- **Mental model.** A Proxy is an *interceptor*: JS sees a normal object; every
  operation (read, write, `in`, `Object.keys`, `JSON.stringify`, call, `new`)
  is routed to one of your Go funcs first.
- **The one sharp edge (read this twice).** `Object.keys()` / `for...in` call
  `[[GetOwnProperty]]` for *every* key. If your `GetOwnPropertyDescriptor` trap
  computes the value, you will materialise the whole object on enumeration.
  **Fix:** return a *flag-only* descriptor (no value, no provider call) from
  `GetOwnPropertyDescriptor`; produce real values only in the `Get` trap.
- **Invariants.** Proxy traps are not a free-for-all: the spec (and goja) force
  the trap results to stay "consistent" with the proxy's **target** object. We
  keep things trivial by using a fresh, empty, **extensible** target.
- **Deliverable plan.** Build a small reusable Go package `lazy` plus a JS
  `require("lazy")` module that exposes `LazyMap`, `LazyArray`, `LazyRecord`
  builders. This guide gives trap-by-trap pseudocode, a test strategy, and a
  phased plan.

---

## 1. Background you need first

### 1.1 What is goja?

[goja](https://github.com/dop251/goja) is a pure-Go implementation of the
ECMAScript 5.1 spec (with many ES6+ features such as `Proxy`, `Symbol`, `Map`,
`Promise`). You embed it in a Go program, hand it JS source, and it runs it.
Everything lives on one type:

```go
vm := goja.New()                 // a runtime = a JS world
v, err := vm.RunString("1 + 2")  // run JS, get a goja.Value back
vm.Set("x", 42)                  // push a Go value into the JS world
```

`goja.Value` is the boundary type: numbers, strings, booleans, `*goja.Object`,
`*goja.Symbol`, and null/undefined all live behind it. You convert with
`vm.ToValue(goThing)` (Go → JS) and `value.Export()` / `value.ToObject(vm)`
(JS → Go).

### 1.2 What is an ES Proxy?

A Proxy wraps an object (the **target**) and routes operations through a
**handler** made of *trap* functions. There are 13 fundamental traps, one per
internal method of the spec's ordinary object:

| Operation in JS | Internal method | Trap name |
| --- | --- | --- |
| `obj.x`, `obj[k]` | `[[Get]]` | `get` |
| `obj.x = 1` | `[[Set]]` | `set` |
| `"x" in obj` | `[[HasProperty]]` | `has` |
| `delete obj.x` | `[[Delete]]` | `deleteProperty` |
| `Object.keys`, `for..in` | `[[OwnPropertyKeys]]` + `[[GetOwnProperty]]` | `ownKeys`, `getOwnPropertyDescriptor` |
| `Object.getOwnPropertyDescriptor` | `[[GetOwnProperty]]` | `getOwnPropertyDescriptor` |
| `Object.defineProperty` | `[[DefineOwnProperty]]` | `defineProperty` |
| `Object.getPrototypeOf` | `[[GetPrototypeOf]]` | `getPrototypeOf` |
| `Object.setPrototypeOf` | `[[SetPrototypeOf]]` | `setPrototypeOf` |
| `Object.isExtensible` | `[[IsExtensible]]` | `isExtensible` |
| `Object.preventExtensions` | `[[PreventExtensions]]` | `preventExtensions` |
| `proxy(...)` | `[[Call]]` | `apply` |
| `new proxy(...)` | `[[Construct]]` | `construct` |

If you don't implement a trap, the operation "falls through" to the target — so
a Proxy with zero traps behaves exactly like its target.

### 1.3 Why this is perfect for lazy data

Laziness means *defer the work until it's needed*. A Proxy intercepts the exact
moments "needed" happens (a read, an `in`, an enumeration). So a lazy Go value
behind a Proxy looks like a normal object from JS, and the Go side only does
work in response to a specific JS operation. No copying, no up-front
materialisation, no promise/callback ceremony.

---

## 2. The goja Proxy API (Go side)

> Evidence: `goja/builtin_proxy.go` (`ProxyTrapConfig`, `NewProxy`),
> `goja/proxy.go` (`Proxy` type, internal `proxyHandler`).

### 2.1 Creating a Proxy from Go

```go
target := vm.NewObject()                      // see §4.1 on target choice
p := vm.NewProxy(target, &goja.ProxyTrapConfig{
    Get:    func(target *goja.Object, prop string, receiver goja.Value) goja.Value { ... },
    OwnKeys: func(target *goja.Object) *goja.Object { ... },
    // ... only the traps you care about
})
vm.Set("device", p)                           // a Proxy is a goja.Value
```

`NewProxy` returns a `goja.Proxy` (an opaque Go wrapper). It is a `goja.Value`,
so `vm.Set`, slice elements, return values, etc. all accept it directly. On the
JS side it is indistinguishable from a normal object/function.

### 2.2 `ProxyTrapConfig` — one Go func per trap

Each field is an **optional** Go func. `nil` means "no trap → fall through to the
target". Three flavours exist for property-keyed traps to let you specialise
integers and Symbols without re-parsing strings:

- `Get` / `GetIdx` / `GetSym`
- `Has` / `HasIdx` / `HasSym`
- `Set` / `SetIdx` / `SetSym`
- `DeleteProperty` / `DeletePropertyIdx` / `DeletePropertySym`
- `GetOwnPropertyDescriptor` / `...Idx` / `...Sym`
- `DefineProperty` / `...Idx` / `...Sym`

> **Rule (from the source):** if an `*Idx` trap is set, goja routes *canonical*
> integer string keys (`"0"`, `"123"`) to it; otherwise the string trap handles
> them. See the doc comment on `ProxyTrapConfig` in `builtin_proxy.go`.

Remaining single traps: `GetPrototypeOf`, `SetPrototypeOf`, `IsExtensible`,
`PreventExtensions`, `OwnKeys`, `Apply` (callable proxy), `Construct`
(constructible proxy).

### 2.3 The internal `proxyHandler` interface (advanced)

`ProxyTrapConfig` is a *convenience* adapter. Underneath, goja has an internal
Go interface (`proxyHandler` in `proxy.go`) with one method per trap. A
`nativeProxyHandler` adapts your `ProxyTrapConfig` to it. For 99% of cases the
config is enough; you'd reach for the interface only to avoid the per-trap
`nil`-check overhead or to implement exotic custom semantics.

### 2.4 `Proxy` as a Go value

A JS Proxy, when it crosses back into Go (`value.Export()`), becomes a
`goja.Proxy`. You can call `p.Target()`, `p.Handler()`, and `p.Revoke()` from
Go. **Revocation** nulls the handler so further JS operations throw
`TypeError: Proxy already revoked` — handy for tearing down a lazy resource.

---

## 3. How goja actually dispatches operations (mental model)

This section is the difference between guessing and knowing. Trace what happens
for common JS idioms on `device` (a Proxy):

```
device.wifis
   └─ vm executes [[Get]]("wifis")
        └─ proxyObject.getStr -> nativeProxyHandler.getStr
             └─ ProxyTrapConfig.Get(target, "wifis", receiver)   <-- YOUR GO FUNC
```

```
"wifis" in device
   └─ [[HasProperty]]("wifis")
        └─ nativeProxyHandler.hasStr -> ProxyTrapConfig.Has(...)   <-- YOUR GO FUNC
```

```
Object.keys(device)
   └─ [[OwnPropertyKeys]]
        └─ ProxyTrapConfig.OwnKeys(target) -> ["name","wifis",...]   <-- YOUR GO FUNC
   then for EACH key:
      └─ [[GetOwnProperty]](key)   to decide enumerability
           └─ ProxyTrapConfig.GetOwnPropertyDescriptor(...)          <-- YOUR GO FUNC  (DANGER: see §6)
```

```
JSON.stringify(device)
   └─ [[Get]]("toJSON")           (looked up first!)
   then for each OWN enumerable key:
      └─ [[Get]](key)              <-- materialises the value (unavoidable)
```

> File anchors: dispatch lives in `proxyObject` methods (`proxy.go`), e.g.
> `getStr` → `checkHandler().getStr(...)`, `keys()` → `proxyOwnKeys()` +
> `getOwnProp(key)` filtering.

---

## 4. The four-part anatomy of a lazy Proxy

Every lazy structure in this guide is built from the same four decisions:

1. **The target** — what object the Proxy wraps (controls the invariants).
2. **The provider** — the Go func that produces a value for a key/index.
3. **The cache** — where memoised `goja.Value`s live.
4. **The traps** — which of the 13 you implement, and how.

### 4.1 The target (keep it empty + extensible)

Proxy invariants (enforced in `proxy.go`, mirroring the ES spec) say trap
results must be *consistent* with the target:

- `ownKeys` must include every **non-configurable** property of the target.
- On a **non-extensible** target, `ownKeys` may not add keys the target lacks.
- `get`/`set`/`getOwnPropertyDescriptor` must agree with any non-configurable,
  non-writable target property.

A **fresh, empty, extensible** target (`vm.NewObject()`) satisfies *all* of these
trivially (no own properties ⇒ nothing to be consistent with). **This is the
recommended default.** Only deviate if you need a specific prototype (e.g. an
Array target so the proxy reports `Array.isArray`-ish behaviour) — and even
then, prefer an empty/extensible target and handle `Array.isArray` expectations
explicitly if needed.

> Diagram: the "onion" of a lazy Proxy.
>
> ```
>                 JS sees a normal object
>                          │
>                  ┌───────▼────────┐
>                  │   Proxy shell  │   ← intercepts all 13 ops
>                  └───────┬────────┘
>            fall-through ─┤  (traps you leave nil delegate here)
>                  ┌───────▼────────┐
>                  │  empty target  │   vm.NewObject() — extensible, no own props
>                  └────────────────┘
>                  ┌───────▼────────┐
>                  │  Your Go traps │   Get / Has / OwnKeys / GetOwnPropertyDescriptor
>                  └───────┬────────┘
>                  ┌───────▼────────┐
>                  │ provider + cache│  lazyGet(key) -> goja.Value (memoised)
>                  └────────────────┘
> ```

### 4.2 The provider

A plain Go func: `func(key string) any` (maps/records) or `func(i int) any`
(arrays). It may block (e.g. perform a real wifi scan) because goja runs JS
synchronously on one goroutine — the JS caller simply waits. Return `nil` to
mean "no such property" (traps map that to `undefined`).

### 4.3 The cache

Memoise **`goja.Value`, not `any`**. Why: `vm.ToValue` allocates (a `*Object`
for maps/slices, etc.); doing it once per key keeps repeated reads cheap and
keeps object identity stable (`device.wifis === device.wifis`). Key the cache by
the same type the trap receives (`string` for maps, `int` for arrays).

```go
func lazyGet(vm *goja.Runtime, cache map[string]goja.Value,
             provider func(string) any, key string) goja.Value {
    if v, ok := cache[key]; ok { return v }
    raw := provider(key)
    if raw == nil { return goja.Undefined() }
    v := vm.ToValue(raw)
    cache[key] = v
    return v
}
```

### 4.4 The traps — the lazy five

For a read-only lazy object you almost always implement exactly these:

| Trap | When JS does | Lazy responsibility |
| --- | --- | --- |
| `Get` | `obj.k`, `obj[i]` | `return lazyGet(...)` — **the** lazy path |
| `Has` | `"k" in obj` | cheap set membership; **never** call the provider |
| `OwnKeys` | `Object.keys`, `for..in`, `Object.assign` | return the key list; **never** call the provider |
| `GetOwnPropertyDescriptor` | enumeration filter, `getOwnPropertyDescriptor`, `JSON.stringify` | return a **flag-only** descriptor; **never** call the provider (see §6) |
| (optional) `Set` | `obj.k = v` | record writes into a Go map; return `true` |

Everything else (`GetPrototypeOf`, `isExtensible`, …) can stay `nil` and fall
through to the empty target.

---

## 5. Worked patterns (pseudocode + what JS sees)

### 5.1 Lazy map / "device" object (keys known up front)

Use case: a device descriptor where `wifis` is an expensive scan you only want
to run if JS reads it.

```go
device := &lazyMap{
    keys: []string{"name", "wifis", "battery", "uptime"},
    provider: func(k string) any {
        switch k {
        case "wifis":   return scanWifis()   // expensive, runs only on read
        case "battery": return readBattery()
        ...
        }
        return nil
    },
}
vm.Set("device", newLazyMapProxy(vm, device))

func newLazyMapProxy(vm *goja.Runtime, lm *lazyMap) goja.Proxy {
    target := vm.NewObject()
    return vm.NewProxy(target, &goja.ProxyTrapConfig{
        Get: func(t *goja.Object, p string, r goja.Value) goja.Value {
            if !contains(lm.keys, p) { return goja.Undefined() }
            return lazyGet(vm, lm.cache, lm.provider, p)
        },
        Has: func(t *goja.Object, p string) bool {
            return contains(lm.keys, p) || defined(t.Get(p)) // incl. inherited
        },
        GetOwnPropertyDescriptor: func(t *goja.Object, p string) goja.PropertyDescriptor {
            // FLAG-ONLY: keep enumeration cheap (§6). Enumerable iff declared.
            return goja.PropertyDescriptor{
                Enumerable:   goja.ToFlag(contains(lm.keys, p)),
                Configurable: goja.FLAG_TRUE,
            }
        },
        OwnKeys: func(t *goja.Object) *goja.Object {
            out := make([]any, len(lm.keys))
            for i, k := range lm.keys { out[i] = k }
            return vm.NewArray(out...)
        },
    })
}
```

JS behaviour (verified by experiment `04`):

```js
device.name                       // -> "atom-3r"           (provider called: name)
device.wifis                      // -> ["scapegoat",...]   (provider called: wifis; cached)
device.wifis                      // -> same array, ===     (provider NOT called again)
"wifis" in device                 // -> true                (provider NOT called)
Object.keys(device)               // -> ["name","wifis","battery","uptime"]  (provider NOT called)
JSON.stringify(device)            // materialises all values (unavoidable)
```

### 5.2 Infinite lazy array

Use case: a sequence computed on demand (`fibs[i]`, `rows[i]`) where "length"
is large or symbolic.

```go
const N = 1 << 20
proxy := vm.NewProxy(vm.NewArray(), &goja.ProxyTrapConfig{
    GetIdx: func(t *goja.Object, i int, r goja.Value) goja.Value {
        if i < 0 || i >= N { return goja.Undefined() }
        return lazyGetInt(vm, cache, func(i int) any { return i*i }, i) // e.g. squares
    },
    HasIdx: func(t *goja.Object, i int) bool { return i >= 0 && i < N },
    Get: func(t *goja.Object, p string, r goja.Value) goja.Value {
        if p == "length" { return vm.ToValue(N) }
        return goja.Undefined()
    },
})
```

JS behaviour (verified, experiment `02`):

```js
infinite[5]                      // -> "item-25"
infinite[3]                      // -> "item-9"
infinite[5]                      // -> cached, provider not recalled
```

> Note: `for (const x of infinite)` uses the **iterable protocol** (`@@iterator`
> / `Symbol.iterator`), not array indexing. If you want `for..of` support,
> expose a `GetSym` trap returning a generator-like function, or implement
> `Symbol.iterator`. Out of scope for v1; see §9.

### 5.3 Dynamic key/value namespace (keys unknown until enumerated)

Use case: environment variables, a KV store, a live registry — where the *set*
of keys is itself produced on demand (e.g. a paged query). Here `OwnKeys`
calls a Go "list" function (cheap), and `Get` calls a "fetch" function per key.

```go
proxy := vm.NewProxy(vm.NewObject(), &goja.ProxyTrapConfig{
    Get:                      func(t, p, r) { return lazyGet(vm, cache, fetch, p) },
    Has:                      func(t, p) bool { _, ok := store[p]; return ok },
    GetOwnPropertyDescriptor: func(t, p) { return flagOnly(enumerable: true) },
    OwnKeys: func(t *goja.Object) *goja.Object {
        return toArrayObject(vm, listKeys(store)) // cheap listing query
    },
})
```

JS behaviour (verified, experiment `03`):

```js
env.EDITOR                       // -> "nvim"   (fetch called)
Object.keys(env).length          // -> N        (only listKeys called; fetch NOT called)
"HOME" in env                    // -> true     (fetch NOT called)
```

### 5.4 Writable lazy store (optional)

Add a `Set` trap to capture writes without losing laziness:

```go
Set: func(t *goja.Object, p string, v goja.Value, r goja.Value) bool {
    cache[p] = v          // overwrite cache so future reads see the write
    return true           // MUST return true to indicate success
}
```

Returning `false` makes strict-mode assignment throw. Combine with a `DeleteProperty` trap (`return true` after deleting from the cache) for a mutable namespace.

---

## 6. The enumeration sharp edge (read this section, then read it again)

### 6.1 The problem

`Object.keys(obj)` / `for (const k in obj)` do **not** stop at `ownKeys`. After
getting the key list, the engine calls `[[GetOwnProperty]](key)` on *each* key
to decide whether it is enumerable. In goja that means (see `proxyObject.keys()`
in `proxy.go`):

```
for key in ownKeys-result:
    prop = proxy.getOwnProp(key)        # <-- invokes YOUR GetOwnPropertyDescriptor trap
    if prop == nil or undefined: skip
    if prop is valueProperty and not enumerable: skip
    keep key
```

So if your `GetOwnPropertyDescriptor` builds the value (calls the provider),
**enumerating the object materialises every value**. Measured: enumerating a
4-key lazy device called the provider for every non-cached key
(`provider called for: [battery uptime]`). For a 10 000-row lazy record that is
a disaster.

### 6.2 The fix: flag-only descriptors

Return a descriptor that carries **only flags**, never a value:

```go
GetOwnPropertyDescriptor: func(t *goja.Object, p string) goja.PropertyDescriptor {
    return goja.PropertyDescriptor{
        Enumerable:   goja.FLAG_TRUE,
        Configurable: goja.FLAG_TRUE,
        // Value and Writable intentionally left unset
    }
}
```

Measured (experiment `04`): after this change, `Object.keys` and `for..in` log
`provider called for: []` — enumeration is now O(keys), values stay lazy, and
reads still work normally.

### 6.3 Why the flag-only descriptor survives (goja internals)

This is subtle and worth understanding, because a tiny change can silently break
`Object.keys` (keys get dropped):

1. Your Go callback returns `PropertyDescriptor{Enumerable: TRUE, Configurable: TRUE}` (no `Value`).
2. `nativeProxyHandler.getOwnPropertyDescriptorStr` calls `desc.toValue(r)` (`object.go:85`) → builds a JS descriptor object `{enumerable:true, configurable:true}` (no `value`/`writable` keys).
3. `proxyObject.proxyGetOwnPropertyDescriptor` (`proxy.go`) calls `r.toPropertyDescriptor(obj)` then `resultDesc.complete()` (`object.go:124`). `complete()` fills the gaps: `Value=_undefined`, `Writable=FLAG_FALSE`, keeps `Enumerable=FLAG_TRUE`.
4. Final branch (`proxy.go`):
   ```
   if Writable==TRUE && Configurable==TRUE && Enumerable==TRUE { return resultDesc.Value }
   return r.toValueProp(trapResultObj)
   ```
   Because `Writable` is `FLAG_FALSE`, the **fast path does not run**; instead
   `toValueProp` (`builtin_object.go:114`) builds a `valueProperty` with
   `enumerable=true`. That is what `keys()` sees, so the key is **kept**.
5. Had `Writable` defaulted to `TRUE`, the fast path `return resultDesc.Value`
   (which is `_undefined`) would run, and `keys()` treats `_undefined` as
   "absent" → **the key would vanish from `Object.keys`**.

> **Guidance:** always set `Configurable: FLAG_TRUE` and **leave `Writable`
> unset** in lazy descriptors, and set `Enumerable` explicitly. Provide a helper
> (`flagOnlyDescriptor(enumerable bool)`) in the reusable module so authors
> cannot accidentally hit the fast-path skip.

### 6.4 What is *unavoidably* materialising

- `JSON.stringify(x)` looks up `toJSON` first (a `Get` on the object) and then
  `Get`s each own enumerable key. If your object is large, tell consumers to
  pass a replacer or to avoid `JSON.stringify` on lazy objects. (Measured: a
  `Get("toJSON")` happens before any value read.)
- `Object.assign({}, x)`, spread `{...x}`, destructuring with defaults, and
  `structuredClone` all read values for the keys they touch.
- `for..of` on an array-like needs `Symbol.iterator` (see §5.2 note).

---

## 7. Caching, identity, and concurrency

- **Cache `goja.Value`.** Repeated `device.wifis` returns the same `*Object`, so
  `===` holds and downstream JS caches keyed on identity stay valid.
- **Per-runtime cache.** A `goja.Value` is bound to one `*goja.Runtime`
  (`proxy.toValue` even panics on cross-runtime use). Cache lifetimes must not
  outlive the runtime. If you expose a builder across runtimes, build a fresh
  proxy per runtime.
- **Single-threaded by contract.** goja runs JS on the goroutine that calls
  `RunString`; traps run on that same goroutine. A blocking provider blocks JS,
  which is usually fine. Do **not** share the cache/proxy across goroutines
  without a mutex; if a provider needs to fan out to background work, do it in
  Go and block JS only on the result.
- **Invalidation.** For live data (wifi list that changes), expose an explicit
  `refresh()` (drop the cache) rather than silently caching stale values, or
  return fresh values from the provider and skip the cache for volatile keys.

---

## 8. Proposed reusable module: `lazy`

A small Go package + a JS `require("lazy")` module so authors never hand-wire
`ProxyTrapConfig`.

### 8.1 Go API sketch

```go
package lazy

// Map is a lazily-valued string-keyed object.
type Map struct {
    Keys     []string
    Provider func(key string) any
    cache    map[string]goja.Value
}

// Array is a lazily-valued integer-indexed object with a reported length.
type Array struct {
    Len      int
    Provider func(i int) any
    cache    map[int]goja.Value
}

// ToValue wraps a lazy value into a goja Proxy bound to vm.
func (m *Map) ToValue(vm *goja.Runtime) goja.Value   { /* builds Proxy */ }
func (a *Array) ToValue(vm *goja.Runtime) goja.Value { /* builds Proxy */ }

// flagOnlyDescriptor is the §6 helper.
func flagOnlyDescriptor(enumerable bool) goja.PropertyDescriptor {
    return goja.PropertyDescriptor{
        Enumerable:   goja.ToFlag(enumerable),
        Configurable: goja.FLAG_TRUE,
    }
}
```

### 8.2 JS module (`require("lazy")`)

Wire it through go-go-goja's `NativeModule`/`Registry` (see
`modules/common.go`, mirrored on `modules/fs/fs.go`):

```go
package lazy

type m struct{}
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "lazy" }
func (m) Doc() string  { return "Builders for lazily-materialised objects." }
func (m) Loader(vm *goja.Runtime, exports *goja.Object) {
    // Lazy.map({ keys, get }) -> Proxy   (get receives a Go-backed hook)
    modules.SetExport(exports, "lazy", "map", func(opts map[string]any) goja.Value {
        keys := toStringSlice(opts["keys"])
        getter := vm.ToValue(opts["get"]).ToObject(vm) // JS function
        lm := &lazy.Map{
            Keys: keys,
            Provider: func(k string) any {
                v, _ := goja.AssertFunction(getter)(goja.Undefined(), vm.ToValue(k))
                return v.Export()
            },
        }
        return lm.ToValue(vm)
    })
    // ... array, record ...
}

func init() { modules.Register(m{}) }
```

From JS:

```js
const lazy = require("lazy")
const device = lazy.map({
    keys: ["name", "wifis", "battery"],
    get:  (k) => k === "wifis" ? scanWifis() : readCheap(k),
})
device.wifis          // lazy!
```

> Wiring note: go-go-goja's `NativeModule.Loader` is synchronous and runs once
> per runtime; builders like `lazy.map` then construct per-call Proxies. The
> go-go-goja `goja_nodejs` skew (see diary Step 1) must be resolved before this
> module can live inside go-go-goja; until then it can ship as a standalone
> package compiled against the fork.

### 8.3 Phased implementation plan

1. **Phase 0 — resolve skew (or bypass).** Either bump the local goja fork to a
   version exporting `IsNumber`/`IsString`/`IsBigInt`, or build `lazy` as a
   standalone package against the fork. Deliverable: `go build ./...` clean in
   the chosen home.
2. **Phase 1 — Go core.** `lazy.Map`, `lazy.Array`, `flagOnlyDescriptor`,
   `lazyGet`, with unit tests mirroring experiments `01`–`04`
   (assert `accessLog` is empty for enumeration, non-empty for reads).
3. **Phase 2 — JS module.** `require("lazy")` with `map`/`array`/`record`
   builders, a `lazy.doc()`, and a `tsgen` TypeScript declaration (see
   `modules/fs/fs.go` `TypeScriptModule()` for the pattern).
4. **Phase 3 — ergonomics.** `Set`/`DeleteProperty` traps (mutable stores),
   `Symbol.iterator` for array-likes, `revoke()` lifecycle, error surfacing
   (panics in traps become JS exceptions via `vm.NewGoError`).
5. **Phase 4 — docs & examples.** A `examples/lazy/` runnable demo and a help
   page in the goja-repl help tree.

---

## 9. Testing & validation strategy

- **Behavioural assertions on the access log.** Each trap path is tested by
  recording which keys the provider was called for:
  - read of `k` ⇒ log contains `k`
  - second read of `k` ⇒ log unchanged (cache)
  - `"k" in obj` ⇒ log unchanged
  - `Object.keys(obj)` / `for..in` ⇒ log unchanged (proves §6 fix)
  - `JSON.stringify(obj)` ⇒ log contains every enumerable key
- **Identity test.** `assert(obj.k === obj.k)` (cache returns same `*Object`).
- **Invariant test.** Construct with a non-extensible target and assert goja
  throws a `TypeError` from `ownKeys` when the trap returns extra keys (proves
  the §4.1 reasoning is load-bearing).
- **Concurrency test.** Run the same proxy from two goroutines under `-race`
  to confirm the single-goroutine contract (expected: either guard with a mutex
  or document the single-threaded requirement).
- **Regression test for §6.3.** A dedicated `TestFlagOnlyDescriptorStaysEnumerable`
  that asserts `Object.keys` length equals the declared key count; this will
  fail loudly if a goja upgrade flips the `Writable` default into the fast path.

Run harness (against the local fork, per diary Step 2):

```
cd explorations/lazyproxy && GOWORK=off go run . 04   # quick manual check
cd goja && go test ./... -count=1                     # fork itself stays green
```

---

## 10. Risks, alternatives, open questions

### 10.1 Risks

- **Silent key-drop on goja upgrade.** The §6.3 fast-path interaction is
  fragile; mitigate with the dedicated regression test and the
  `flagOnlyDescriptor` helper.
- **Cross-runtime use panics.** A cached `goja.Value` used in another runtime
  panics ("Illegal runtime transition"). Mitigate by building per-runtime and
  documenting the constraint.
- **Blocking providers freeze JS.** A slow `scanWifis()` stalls the whole
  runtime. Mitigate by making expensive providers async-aware (return a Promise
  via `vm.NewPromise`) or running them off the JS goroutine in Go.
- **`JSON.stringify` cost.** Unavoidable per spec; document and offer a
  `toJSON()` trap that returns a compact/explicit shape if needed.

### 10.2 Alternatives considered

- **`vm.Set` of a fully-materialised map.** Simple but eager — defeats the
  purpose and blows memory for large/infinite data.
- **Go `map`/`struct` auto-wrapping (`ToValue`).** goja reflects Go maps/structs
  into objects, but eagerly (no per-key hooks) and without trap-level control.
  Good for static data, wrong for lazy.
- **JS-side `Proxy` with a JS handler calling Go via exported funcs.** Works,
  but adds a JS→Go→JS hop per trap and loses the chance to keep enumeration
  cheap in Go. The native `ProxyTrapConfig` is strictly better.
- **Iterators/generators instead of array Proxies.** Right tool for *streams*
  (`Symbol.iterator` + a Go-backed next()), complementary to (not replacing)
  indexable lazy arrays. See open question 3.

### 10.3 Open questions

1. Should the reusable `lazy` module live in go-go-goja (requires resolving the
   `goja_nodejs` skew) or as a standalone package for now?
2. Do we want a `lazy.stream()` (pull-based iterator over `Symbol.iterator`) as
   a first-class citizen alongside maps/arrays?
3. For very large arrays, should `length` be reported as a finite number or
   should we expose a paging helper? (`for..of` on a billion-element proxy is a
   footgun.)
4. Caching policy for volatile providers: opt-in `ttl`, explicit `refresh()`,
   or always-fresh? Prefer explicit `refresh()` to avoid surprises.

---

## 11. Decision records

### DR-1: Use `ProxyTrapConfig` (not the internal `proxyHandler` interface)

- **Context:** Two ways to implement traps in Go.
- **Options:** (a) high-level `ProxyTrapConfig` struct of optional funcs; (b)
  internal `proxyHandler` interface.
- **Decision:** (a).
- **Rationale:** Public, documented, sufficient for all lazy patterns; (b) is
  an internal optimisation we do not need.
- **Consequences:** Slight per-trap nil-check overhead (negligible). Status: accepted.

### DR-2: Empty, extensible target by default

- **Context:** Proxy invariants are checked against the target.
- **Options:** (a) fresh `vm.NewObject()`; (b) typed target (Array, custom proto).
- **Decision:** (a) by default; typed targets only when `Array.isArray`/prototype
  semantics are explicitly required.
- **Rationale:** Minimises invariant surface; empty + extensible ⇒ no trap
  result can violate an invariant. Status: accepted.

### DR-3: Flag-only `GetOwnPropertyDescriptor`

- **Context:** Enumeration materialises values if the descriptor carries a value.
- **Decision:** Lazy descriptors carry only `Enumerable`/`Configurable`; values
  come from `Get`.
- **Rationale:** Keeps `Object.keys`/`for..in` O(keys) and side-effect-free
  (measured). Status: accepted; guarded by a regression test (§9).

### DR-4: Cache `goja.Value` per runtime

- **Context:** Memoisation is required for identity and performance.
- **Decision:** Cache the converted `goja.Value`, keyed per `*goja.Runtime`.
- **Rationale:** Stable identity (`===`), single `ToValue` per key, avoids
  cross-runtime panics. Status: accepted.

---

## 12. References (key files & anchors)

| What | Where |
| --- | --- |
| Public Proxy creation + trap config | `goja/builtin_proxy.go` (`ProxyTrapConfig`, `NewProxy` at line ~351) |
| Proxy runtime behaviour + invariants | `goja/proxy.go` (`proxyObject`, internal `proxyHandler` at line ~88) |
| `PropertyDescriptor` struct | `goja/object.go:58`; `toValue` at `:85`; `complete` at `:124` |
| Why flag-only descriptors stay enumerable | `goja/builtin_object.go:114` (`toValueProp`) + `proxy.go` `keys()` |
| Go module/`require()` pattern for the reusable module | `go-go-goja/modules/common.go`, `go-go-goja/modules/fs/fs.go` |
| Runtime composition / lifecycle | `go-go-goja/pkg/engine/factory.go`, `runtime_modules.go` |
| Runnable evidence (lazy map, infinite array, kv object, cheap enum) | `explorations/lazyproxy/*.go` (mirror in this ticket's `scripts/`) |
| Investigation diary & measured outputs | this ticket `reference/01-investigation-diary.md`, `scripts/outputs.txt` |

---

## 13. Glossary

- **Trap.** A function the Proxy calls to intercept one internal object operation.
- **Target.** The object a Proxy wraps; invariants are checked against it.
- **Provider.** Your Go func that produces a value for a key/index on demand.
- **`Flag`.** goja's tri-state for descriptor attributes: `FLAG_NOT_SET`, `FLAG_FALSE`, `FLAG_TRUE` (`goja/runtime.go:157`).
- **Fall-through.** When a trap is `nil`, the operation is delegated to the target unchanged.
- **Materialise.** To actually compute/convert a lazy value into a concrete `goja.Value`.
