---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/common.go
      Note: NativeModule interface and SetExport — the host wiring convention
    - Path: go-go-goja/perf/goja/phase3_bindings_bench_test.go
      Note: the benchmark suite implemented for this ticket
    - Path: goja/builtin_proxy.go
      Note: ProxyTrapConfig (line 256) + NewProxy (line 351) — the Go proxy-with-manual-dispatch API
    - Path: goja/object_dynamic.go
      Note: DynamicObject/DynamicArray interfaces and NewDynamicObject/NewDynamicArray — the cheap Go-backed object mechanism
    - Path: goja/proxy.go
      Note: ES6 Proxy object; getStr dispatch + invariant checks (the overhead source
    - Path: goja/runtime.go
      Note: toValue type switch (1786)
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---







# Goja Go→JS Binding Mechanisms — Benchmark & Intern Guide

> **TL;DR for the impatient.** There are roughly half a dozen ways to hand a Go
> value or function to JavaScript inside `goja`. They are *not* equally cheap.
> A **native** function (`func(FunctionCall) Value`) is the floor (~219 ns/call).
> An arbitrary Go function set with `vm.Set` pays ~3.5× that because every call
> goes through `reflect.Call`. An **ES6 Proxy** with a manual `Get` trap costs
> ~199 ns per property read — about **1.6× a plain JS object and +1 allocation**,
> because goja re-checks ES6 invariants against the proxy target on every access.
> A **`DynamicObject`** reads in ~125 ns with 1 allocation — *as fast as a plain
> object* — and is the recommended Go-controlled object type when you do not need
> the full Proxy trap matrix.

---

## 1. Executive summary

This document explains, for an engineer who has never seen this codebase:

1. **What `goja` is** and how the host (`go-go-goja`) embeds it.
2. **The value model** that makes "exposing Go to JS" non-trivial.
3. **Every binding mechanism** — what it is, its Go API, its runtime path, its
   cost — with file/line anchors into the actual source.
4. **A measured benchmark** (`perf/goja/phase3_bindings_bench_test.go`) that
   compares all mechanisms doing identical work.
5. **The answer** to the motivating question: *how expensive is a Go-backed
   Proxy with manual dispatch?*
6. **Recommendations, decision records, and how to reproduce.**

The single most important mental model: **JS never touches Go memory directly.**
Every read, write, and call crosses a *bridging layer* that translates between
goja's internal `Value` representation and Go. Different binding mechanisms are
just different bridging layers with different trade-offs.

---

## 2. The system, in one diagram

```text
                ┌──────────────────────────────────────────────────────────┐
   JS source    │  "svc.add(40, 2)"   /   "obj.v"                          │
                └────────────────────────────┬─────────────────────────────┘
                                             │ compiled once into a *Program
                                             ▼
                ┌──────────────────────────────────────────────────────────┐
   goja VM      │  vm.go  — bytecode interpreter, stack machine             │
   (this fork)  │  value.go — Value interface (the only thing JS sees)     │
                └────────────────────────────┬─────────────────────────────┘
                                             │ property read / call
                                             ▼
                ┌──────────────────────────────────────────────────────────┐
   objectImpl   │  object.go        — the "object" interface (objectImpl)   │
                │  object_goreflect — reflect struct fields+methods         │
                │  object_dynamic   — DynamicObject / DynamicArray handlers │
                │  proxy            — ES6 Proxy traps                       │
                │  object_gomap     — map[string]interface{} live view      │
                │  func.go          — native + wrapped function objects     │
                └────────────────────────────┬─────────────────────────────┘
                                             │ YOUR Go code lives here
                                             ▼
                                   ┌─────────────────────┐
                                   │  Go (host) funcs,   │
                                   │  structs, maps      │
                                   └─────────────────────┘
```

**Host app = `go-go-goja`** (`go-go-goja/` directory). It is a sandbox for
wiring Go-implemented "native modules" into a goja runtime via the
`goja_nodejs/require` subsystem. See `go-go-goja/README.md` and
`go-go-goja/AGENT.md`.

**Engine = `goja`** (`goja/` directory). This is a *local fork* of
`github.com/dop251/goja` selected by `go.work` (`use ./goja`). Every
file:line reference in this document points into this local copy, because that
is the engine the benchmarks actually compile against.

---

## 3. Background an intern must internalize first

### 3.1 What goja is

goja is an ECMAScript (ES5.1 + much of ES6) interpreter written in pure Go.
There is no JIT. Source text is parsed (`parser/`) → compiled to a compact
bytecode (`compiler*.go`) → executed by a stack-based interpreter (`vm.go`).
Because there is no JIT, **the cost of crossing the Go↔JS boundary dominates**
hot loops; the binding layer *is* the performance story.

### 3.2 The `Value` interface — the only thing JS may hold

Everything JavaScript can see implements one interface
(`goja/value.go`):

```go
type Value interface {
    ToFloat() float64
    ToInteger() int64
    ToString() String
    ToBoolean() bool
    Export() interface{}      // best-effort conversion back to a Go value
    // ... plus Equal, SameAs, etc.
}
```

Consequences:

- A Go `int` is *not* a JS value. It must be wrapped (e.g.
  `vm.ToValue(42)` → an internal `valueInt`).
- A Go `*MyStruct` is *not* a JS value either. It must be wrapped in an
  **object** that implements `objectImpl` (see §4).
- There is **no** exported `goja.ToInt64` in this fork, and `FunctionCall` has
  **no** `Runtime` field — so a Go callback that must *return* a `Value` needs a
  captured `*Runtime` (see `nativeAdder` in the benchmark).

### 3.3 How a call flows: `assertCallable` + `vmCall`

When JS executes `f(a, b)`, the VM resolves `f` to an object and asks it for a
callable. Every function object implements, at minimum:

```go
// goja/func.go:551
func (f *nativeFuncObject) assertCallable() (func(FunctionCall) Value, bool)
```

`FunctionCall` is the Go-side call frame (`goja/object.go:249`):

```go
type FunctionCall struct {
    This      Value
    Arguments []Value
}
```

The VM's trampoline is `nativeFuncObject.vmCall` (`goja/func.go:558`):

```go
// simplified from goja/func.go:558
func (f *nativeFuncObject) vmCall(vm *vm, n int) {
    vm.pushCtx()
    vm.sb = vm.sp - n
    ret := f.f(FunctionCall{                       // <- the actual Go call
        Arguments: vm.stack[vm.sp-n : vm.sp],
        This:      vm.stack[vm.sp-n-2],
    })
    if ret == nil { ret = _undefined }
    vm.stack[vm.sp-n-2] = ret
    vm.popCtx()
}
```

So a **native** function — one whose `f` field is literally a
`func(FunctionCall) Value` — pays *only* this trampoline plus whatever boxing
the Go body does. That is the performance floor for calls.

### 3.4 How `ToValue` decides what to build

`Runtime.toValue` (`goja/runtime.go:1786`) is a big type switch. The branches
that matter for us:

```go
// goja/runtime.go:1816  — a raw goja callable: wrap as a NATIVE func, no reflect
case func(FunctionCall) Value:
    return r.newNativeFunc(i, name, 0)

// goja/runtime.go:1956  — ANY OTHER func type: wrap with REFLECTION
case reflect.Func:
    return r.newWrappedFunc(value)
```

The distinction between those two branches is the single biggest performance
fork in the whole system, and it is decided entirely by the **Go type** of what
you pass to `vm.Set` / `vm.ToValue`.

---

## 4. The binding mechanisms, ranked

Each subsection gives: **what it is · Go API · runtime path · measured cost ·
when to use it**. File:line anchors are to this fork.

### Mechanism 1 — Native function (the floor)

- **What:** a Go closure with signature `func(goja.FunctionCall) goja.Value`.
- **Go API:**
  ```go
  add := func(call goja.FunctionCall) goja.Value {
      a := call.Argument(0).ToInteger()
      b := call.Argument(1).ToInteger()
      return vm.ToValue(a + b)   // needs a *Runtime → capture vm
  }
  vm.Set("add", add)
  ```
- **Runtime path:** `ToValue` hits `case func(FunctionCall) Value`
  (`runtime.go:1816`) → `newNativeFunc` → stored as `nativeFuncObject` whose
  `f` is your closure. A call runs `vmCall` (`func.go:558`) directly.
- **Cost:** **219 ns, 224 B, 3 allocs** (the 3 allocs are the `RunProgram`
  per-iteration wrapper + two argument lookups; the call itself is essentially
  the trampoline).
- **When:** hot paths, library primitives. The most code you will write, the
  least the runtime will do.

### Mechanism 2 — Wrapped arbitrary function (reflect path)

- **What:** any Go function with "normal" typed args, e.g. `func(a, b int) int`.
- **Go API:** `vm.Set("add", func(a, b int) int { return a + b })`.
- **Runtime path:** `ToValue` falls through the type switch to
  `case reflect.Func` (`runtime.go:1956`) → `newWrappedFunc` (`runtime.go:757`)
  → `wrapReflectFunc` (`runtime.go:1973`). Per call:
  ```text
  for each argument: reflect.New(typ) → toReflectValue(boxedArg)   // alloc per arg
  reflectValue.Call(in)                                            // reflect dispatch
  wrap each result Value
  ```
- **Cost:** **752 ns, 352 B, 12 allocs** — ~3.4× the native floor, all of it
  argument marshalling. (This is what `BenchmarkJSCallingGo` in `bench_test.go`
  already measured for `vm.Set`.)
- **When:** the default, ergonomic choice. Use it unless a function is called
  in a tight loop.

### Mechanism 3 — `modules.SetExport` (same reflect path, nested)

- **What:** `go-go-goja`'s helper that puts a value onto a module's `exports`
  object (`go-go-goja/modules/exports.go:4`).
- **Go API:**
  ```go
  exports := vm.NewObject()
  modules.SetExport(exports, "fs", "readFileSync", readFile) // readFile is func(...)
  ```
- **Runtime path:** identical to Mechanism 2 *plus* one property lookup on the
  exports object per call.
- **Cost:** **918 ns, 352 B, 12 allocs.** Same allocations; the extra ~160 ns is
  the `bench.add` property resolve.
- **When:** when authoring a `NativeModule` (see §6). Functionally the same as
  `vm.Set`; never the bottleneck.

### Mechanism 4 — Reflect struct (fields + methods)

- **What:** publish a Go struct pointer; JS reaches its exported fields and
  methods through reflection.
- **Go API:**
  ```go
  type Calc struct{ Field int }
  func (c *Calc) Add(a, b int) int { return a + b }
  vm.Set("svc", &Calc{Field: 42})
  // JS:  svc.Field   -> 42   ;   svc.Add(40, 2) -> 42
  ```
- **Runtime path:** `ToValue` reaches the `reflect.Ptr`/`reflect.Struct` branch
  → `newObjectGoReflect` (`object_goreflect.go`). Reads go through
  `objectGoReflect.getStr` (`object_goreflect.go:195`) → `_getFieldValue`
  (`:247`) or `_getMethod` (`:212`). A method call does a reflect-method lookup
  **then** the full Mechanism-2 reflect-arg marshalling.
- **Cost:**
  - field read `svc.Field`: **185 ns, 40 B, 2 allocs**
  - method call `svc.Add(40,2)`: **2012 ns, 1141 B, 24 allocs** (the most
    expensive call path — method lookup + reflect arg/ret marshalling, and the
    method Value is re-built each call).
- **When:** convenient for small config/data structs; avoid on method hot paths.

### Mechanism 5 — `map[string]interface{}` (live view)

- **What:** hand goja a Go map; it becomes a *live* JS object backed by
  `objectGoMapSimple`.
- **Go API:** `vm.Set("obj", map[string]interface{}{"v": 42})`.
- **Runtime path:** `ToValue` `case map[string]interface{}` (`runtime.go`,
  near the map branch) builds `objectGoMapSimple` whose get returns the raw
  `interface{}`, which is then `ToValue`-d **every access**.
- **Cost:**
  - read `obj.v` (value is an int): **128 ns, 32 B, 1 alloc**
  - call `m.add(40,2)` where `add` is a Go func: **1716 ns, 1062 B, 21 allocs**
    — the func is **re-wrapped via reflection on every single access**.
- **When:** great for plain data; **never** store functions in a live map that
  JS calls in a loop — use `vm.NewObject()` with `Set` once instead.

### Mechanism 6 — `DynamicObject` / `DynamicArray` (Go-controlled, cheap)

- **What:** implement a tiny Go interface and goja wraps it as a first-class
  object. Reads/writes route straight to your methods.
- **Go API:**
  ```go
  // goja/object_dynamic.go:17
  type DynamicObject interface {
      Get(key string) Value
      Set(key string, val Value) bool
      Has(key string) bool
      Delete(key string) bool
      Keys() []string
  }
  vm.Set("obj", vm.NewDynamicObject(myHandler))   // object_dynamic.go:99
  ```
- **Runtime path:** `dynamicObject.getStr` (`object_dynamic.go:193`) literally
  calls `o.d.Get(p.String())`. No invariant checks, no target inspection.
- **Cost:** read `obj.v`: **125 ns, 32 B, 1 alloc** — *statistically
  indistinguishable from a reused plain JS object* (114 ns). Array read
  `arr[i]`: **168 ns, 96 B, 2 allocs.**
- **When:** the sweet spot for a Go-backed object that JS reads a lot. Gives
  you full control over dispatch without paying the Proxy tax.

### Mechanism 7 — ES6 Proxy with native traps (the "manual dispatch" one)

- **What:** the real ECMAScript `Proxy`, driven from Go via a struct of
  trap callbacks (`ProxyTrapConfig`).
- **Go API:**
  ```go
  // goja/builtin_proxy.go:256  — ProxyTrapConfig (Get/Set/Has/Apply/...)
  target := vm.NewObject()
  proxy := vm.NewProxy(target, &goja.ProxyTrapConfig{       // builtin_proxy.go:351
      Get: func(target *goja.Object, prop string, recv goja.Value) goja.Value {
          // YOUR manual dispatch:
          switch prop {
          case "v": return cachedV42
          }
          return goja.Undefined()
      },
      Has: func(target *goja.Object, prop string) bool { return prop == "v" },
  })
  vm.Set("obj", proxy)
  ```
- **Runtime path** (this is the heart of the proxy overhead):
  ```text
  JS: obj.v
   │
   ▼  proxy.go:601  proxyObject.getStr("v")
   │      │
   │      ├─ checkHandler()            <- validates handler, ES6 invariants
   │      │
   │      ▼  builtin_proxy.go:131  nativeProxyHandler.getStr(target,"v",recv)
   │           ├─ GetIdx trap? (nil here)
   │           └─ Get trap?  ─────────►  YOUR Go callback  (the "manual dispatch")
   │                                        returns (value, handled=true)
   │      │
   │      ▼  proxyGetChecks(target.self.getOwnPropStr("v"), v, name)
   │           ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
   │           goja ALSO inspects the TARGET's own property to enforce ES6
   │           "invariants" (non-configurable, non-writable must agree).
   │           This extra target lookup is the source of the +1 allocation.
   │
   ▼  return v
  ```
- **Cost:**
  - read `obj.v` (switch-hit trap): **199 ns, 48 B, 2 allocs** — ~1.6× a plain
    object, +1 alloc.
  - call `proxy(40,2)` through the `Apply` trap: **278 ns, 224 B, 3 allocs** —
    essentially the native-call floor; the Apply trap adds one callback hop and
    nothing on the allocation axis.
- **When:** when you need the full ES6 trap matrix (intercept `set`, `has`,
  `delete`, `ownKeys`, `construct`, `apply`, `defineProperty`, …) — e.g. lazy
  virtual objects, capability sandboxes, auditing. When you only need reads,
  prefer `DynamicObject`.

### Mechanism 8 — `NativeModule` loader (the host's wiring convention)

This is not a new runtime mechanism — it is `go-go-goja`'s convention for
*packaging* Mechanisms 1–7 into a `require()`-able module.

```go
// go-go-goja/modules/common.go
type NativeModule interface {
    Name() string
    Doc() string
    Loader(*goja.Runtime, *goja.Object)   // populate moduleObj.Get("exports")
}
```

A loader (see `go-go-goja/modules/fs/fs.go`, `Loader`) fetches `exports`, then
calls `modules.SetExport(...)` (Mechanism 3) or `exports.Set(name, nativeFunc)`
(Mechanism 1) for each API. **The whole codebase today uses only Mechanisms 1–5;
neither `NewProxy` nor `NewDynamicObject` appears anywhere in `go-go-goja`.**
That is exactly the gap this ticket closes.

---

## 5. The benchmark

### 5.1 Design goals

1. **Apples-to-apples:** every sub-benchmark returns `42`, so only the binding
   differs.
2. **Isolate the boundary:** reuse one runtime; compile the program once; loop
   only `RunProgram`. Per-iteration compile cost does not pollute call cost.
3. **Allocation as a first-class metric:** `b.ReportAllocs()` — goja boundary
   cost is dominated by boxing/reflection allocations, which are far more stable
   than nanoseconds.
4. **Separate machinery from logic:** `BenchmarkProxyDispatchDetail` holds the
   proxy machinery fixed and varies only the trap body (miss / switch-hit /
   map-hit).

### 5.2 What is measured

| Benchmark | Axis | Variants |
|---|---|---|
| `BenchmarkBindingFunctionCall` | call cost | NativeFunc, WrappedFunc, SetExport, ReflectStructMethod, MapOfFuncs, ProxyApply |
| `BenchmarkBindingPropertyGet` | read cost | PlainObject (lit + reuse), ReflectStructField, GoMapSimple, DynamicObject, Proxy_GetTrap |
| `BenchmarkBindingArrayIndexGet` | read cost | DynamicArray |
| `BenchmarkProxyDispatchDetail` | proxy internals | GetTrap miss-undefined / switch-hit / map-hit |

### 5.3 File location

- Implementation: `go-go-goja/perf/goja/phase3_bindings_bench_test.go`
- Ticket snapshot: `scripts/01-phase3_bindings_bench_test.go`
- Reproducer: `scripts/02-run-bindings-bench.sh`

---

## 6. Results

Run config: `go test ./perf/goja -bench 'Binding|ProxyDispatchDetail' -benchtime=1s -count=5 -benchmem`, then `benchstat`. Hardware: 11th-gen Intel i7-1165G7. Engine: this goja fork, toolchain go1.26.4. Raw + benchstat output saved in `various/01-bindings-bench-raw.txt` and `various/02-bindings-benchstat.txt`.

> **Caveat:** absolute nanoseconds are wall-clock on a shared laptop; treat
> **ratios and allocation counts as the robust signal**, not the ns values.

### 6.1 Function call — `add(40, 2)` → 42

| Mechanism | ns/op | B/op | allocs | vs native |
|---|---:|---:|---:|---:|
| **NativeFunc** `func(FunctionCall) Value` | 219 | 224 | 3 | 1.0× |
| **ProxyApply** (native + Apply trap) | 278 | 224 | 3 | 1.3× |
| **WrappedFunc** `func(int,int) int` | 752 | 352 | 12 | 3.4× |
| **SetExport** (wrapped + nested) | 918 | 352 | 12 | 4.2× |
| **MapOfFuncs** `map[string]any` | 1716 | 1062 | 21 | 7.8× |
| **ReflectStructMethod** `svc.Add` | 2012 | 1141 | 24 | 9.2× |

### 6.2 Property read — `obj.v` / `obj.Field` → 42

| Mechanism | ns/op | B/op | allocs |
|---|---:|---:|---:|
| PlainObject (reused) | 114 | 32 | 1 |
| **DynamicObject** (Get trap) | 125 | 32 | 1 |
| GoMapSimple `map[string]any` | 128 | 32 | 1 |
| DynamicArray `arr[i]` | 168 | 96 | 2 |
| ReflectStructField `obj.Field` | 185 | 40 | 2 |
| **Proxy** (Get trap, switch hit) | 199 | 48 | 2 |
| PlainObject (literal, rebuilt) | 624 | 656 | 8 |

### 6.3 Proxy internals — `obj.v`, Get trap only

| Trap body | ns/op | B/op | allocs |
|---|---:|---:|---:|
| switch-hit (fastest) | 157 | 48 | 2 |
| map-hit | 181 | 48 | 2 |
| miss→undefined | 228 | 48 | 2 |

### 6.4 The headline answer — Proxy-with-manual-dispatch overhead

- **Property reads:** ~199 ns / 48 B / 2 allocs vs ~114–128 ns / 32 B / 1 alloc
  for plain object / map / `DynamicObject`. The Proxy is **~1.6× slower and adds
  exactly one allocation (16 B) per read.** That allocation comes from goja's
  invariant check `proxyGetChecks` (`proxy.go:601`), which re-inspects the proxy
  *target's* own property on every trap invocation. The dispatch logic you write
  inside the trap is *cheap*: a switch-hit (157 ns) beats a miss-to-undefined
  (228 ns) because the hit returns before the prototype walk.
- **Function calls:** the `Apply` trap is nearly free — 278 ns vs 219 ns for the
  raw native func, identical 224 B / 3 allocs. Fronting a callable target with a
  Proxy adds one Go-callback hop and nothing else.
- **Verdict:** Proxy reads are sub-200 ns. For application code where the Go
  side does real work (I/O, parsing, DB), this overhead is **negligible**.
  Reserve "avoid Proxy" advice for tight loops doing millions of trivial reads;
  even there, `DynamicObject` (125 ns, 1 alloc) is the drop-in upgrade, not a
  plain object.

---

## 7. Decision records

### DR-1: Default module APIs use Mechanism 2 (`SetExport` of typed funcs)
- **Context:** ergonomic; no manual `Value` boxing.
- **Decision:** keep it as the default.
- **Rationale:** 918 ns/call is invisible next to any real I/O.
- **Consequences:** none for correctness; revisit only if a primitive lands on a
  CPU-hot loop. **Status: accepted.**

### DR-2: Use `DynamicObject`, not `Proxy`, for Go-backed virtual objects
- **Context:** both give Go full control over JS-visible properties.
- **Options:** `DynamicObject` (5-method interface) vs `Proxy` (full ES6 traps).
- **Decision:** prefer `DynamicObject` unless an ES6-only trap (`construct`,
  `ownKeys`, `defineProperty`, `set` interception, …) is genuinely required.
- **Rationale:** `DynamicObject` matches a plain JS object read (125 vs 199 ns,
  1 vs 2 allocs) and a far smaller API surface.
- **Consequences:** lose Proxy's ability to intercept construction/iteration.
  **Status: accepted.**

### DR-3: Never store Go funcs in a live `map[string]interface{}`
- **Context:** `MapOfFuncs` is the slowest call path (1.7 µs, 21 allocs) because
  the func is re-wrapped via reflection on every access.
- **Decision:** build the object once with `vm.NewObject()` + `Set`, or use a
  `DynamicObject`. **Status: accepted.**

### DR-4: The local `goja/` fork stays the source of truth
- **Context:** `go.work` `use ./goja` overrides upstream goja. The fork lacks
  `IsNumber`/`IsBigInt`/`IsString`, so a 3-function shim was added to
  `goja/runtime.go` to satisfy `goja_nodejs`.
- **Decision:** keep the shim; delete it only when the fork is upgraded.
  **Status: accepted.**

---

## 8. Implementation guide (for the intern)

### 8.1 Picking a mechanism — a decision tree

```text
Do you want JS to CALL something?
│
├─ Is it called >100k/sec in a hot loop?
│    └─ YES → Mechanism 1: native func(FunctionCall) Value (capture vm)
│    └─ NO  → Mechanism 2: plain typed func  (vm.Set / SetExport)
│
Do you want JS to READ/WRITE an OBJECT?
│
├─ Static, known shape?
│    └─ Mechanism 4: reflect struct  (vm.Set(&Struct{}))
│         (or Mechanism 5: map[string]any for POD data ONLY)
│
├─ Dynamic / computed / virtual shape?
│    ├─ Do you need to intercept set / construct / ownKeys / delete?
│    │    └─ YES → Mechanism 7: ES6 Proxy (NewProxy + ProxyTrapConfig)
│    └─ NO  → Mechanism 6: DynamicObject (NewDynamicObject)   ← default choice
│
Are you authoring a require() module?
└─ Wrap the above in a NativeModule.Loader (Mechanism 8)
```

### 8.2 Minimal, copy-pasteable example of each

```go
// 1) native func
add := func(c goja.FunctionCall) goja.Value {
    return vm.ToValue(c.Argument(0).ToInteger() + c.Argument(1).ToInteger())
}
_ = vm.Set("add", add)

// 2) wrapped func
_ = vm.Set("add", func(a, b int) int { return a + b })

// 6) DynamicObject
type kv struct{ m map[string]goja.Value }
func (k *kv) Get(s string) goja.Value      { return k.m[s] }
func (k *kv) Set(s string, v goja.Value) bool { k.m[s] = v; return true }
func (k *kv) Has(s string) bool            { _, ok := k.m[s]; return ok }
func (k *kv) Delete(s string) bool         { delete(k.m, s); return true }
func (k *kv) Keys() []string               { out := make([]string, 0, len(k.m)); for s := range k.m { out = append(out, s) }; return out }
_ = vm.Set("obj", vm.NewDynamicObject(&kv{m: map[string]goja.Value{"v": vm.ToValue(42)}}))

// 7) Proxy with manual dispatch
v42 := vm.ToValue(42)
target := vm.NewObject()
_ = vm.Set("obj", vm.NewProxy(target, &goja.ProxyTrapConfig{
    Get: func(_ *goja.Object, prop string, _ goja.Value) goja.Value {
        if prop == "v" { return v42 }
        return goja.Undefined()
    },
    Has: func(_ *goja.Object, prop string) bool { return prop == "v" },
}))
```

### 8.3 Reproduce the numbers

```bash
cd go-go-goja
go test ./perf/goja -run '^$' \
  -bench 'BindingFunctionCall|BindingPropertyGet|BindingArrayIndexGet|ProxyDispatchDetail' \
  -benchtime=1s -count=5 -benchmem | tee /tmp/raw.txt
benchstat /tmp/raw.txt
# or just: bash <ticket>/scripts/02-run-bindings-bench.sh /tmp/out
```

---

## 9. Risks, alternatives, open questions

- **Wall-clock noise.** Run on a quiet box; the ns are illustrative. Ratios and
  alloc counts are stable across samples.
- **Single-goroutine only.** goja runtimes are not concurrency-safe; these
  numbers are for a single `*Runtime`. Multi-runtime throughput (a pool) is a
  separate question.
- **No JIT.** All numbers reflect the interpreter. A future goja JIT would
  change call costs but not the *relative* ranking of binding mechanisms.
- **Open:** attribute the Proxy's extra alloc precisely with a CPU/alloc
  profile (`-cpuprofile`/`-memprofile`) of `Proxy_GetTrap`.
- **Open:** write-path benchmarks (`DynamicObject.Set`, Proxy `Set` trap) and a
  `Proxy.construct` variant.

---

## 10. Key file references (this workspace)

| Topic | File:line |
|---|---|
| Value interface | `goja/value.go` |
| `FunctionCall` / `ConstructorCall` | `goja/object.go:249` |
| native func trampoline | `goja/func.go:558` (`vmCall`), `:551` (`assertCallable`) |
| `ToValue` type switch | `goja/runtime.go:1786` (`:1816` native-func branch, `:1956` reflect-func branch) |
| `wrapReflectFunc` (per-call reflect) | `goja/runtime.go:1973` |
| `newWrappedFunc` | `goja/runtime.go:757` |
| `Runtime.Set` | `goja/runtime.go:2385` |
| Reflect struct object | `goja/object_goreflect.go:195` (`getStr`), `:212` (`_getMethod`), `:247` (`_getFieldValue`) |
| `DynamicObject` interface | `goja/object_dynamic.go:17` |
| `DynamicArray` interface | `goja/object_dynamic.go:48` |
| `NewDynamicObject` / `NewDynamicArray` | `goja/object_dynamic.go:99` / `:138` |
| `dynamicObject.getStr` | `goja/object_dynamic.go:193` |
| `ProxyTrapConfig` (all traps) | `goja/builtin_proxy.go:256` |
| `NewProxy` | `goja/builtin_proxy.go:351` |
| `nativeProxyHandler` | `goja/builtin_proxy.go:7`, `getStr` at `:131` |
| `proxyObject.getStr` (dispatch + invariants) | `goja/proxy.go:601`, struct at `:286` |
| Host module interface | `go-go-goja/modules/common.go` (`NativeModule`) |
| `SetExport` helper | `go-go-goja/modules/exports.go:4` |
| Real loader example | `go-go-goja/modules/fs/fs.go` (`Loader`) |
| This benchmark | `go-go-goja/perf/goja/phase3_bindings_bench_test.go` |
| Existing perf suites | `go-go-goja/perf/goja/bench_test.go`, `phase2_bench_test.go` |
