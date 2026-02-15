---
Title: Smalltalk Goja Inspector Interface And Component Design
Ticket: GOJA-024-SMALLTALK-INSPECTOR
Status: active
Topics:
    - go
    - goja
    - tui
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/ast-parse-editor/app/model.go
      Note: AST/CST dual-pane and async parse sequencing reference
    - Path: go-go-goja/cmd/inspector/app/drawer.go
      Note: Reusable drawer editor and completion popup behavior
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Baseline Bubble Tea sync patterns and existing inspector behavior
    - Path: go-go-goja/pkg/jsparse/analyze.go
      Note: High-level analysis facade used for proposed architecture
    - Path: go-go-goja/pkg/jsparse/completion.go
      Note: Completion context and candidate generation model
    - Path: go-go-goja/pkg/jsparse/index.go
      Note: AST node index and navigation primitives
    - Path: go-go-goja/pkg/jsparse/resolve.go
      Note: Scope/binding resolution and usages model
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/scripts/goja-runtime-probe.go
      Note: Runtime behavior probe for symbols
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/scripts/jsparse-index-probe.go
      Note: Static index/resolution probe for global bindings and class nodes
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/sources/local/smalltalk-goja-inspector.md
      Note: Primary screen specification used verbatim in analysis
ExternalSources:
    - local:smalltalk-goja-inspector.md
Summary: Implementation-oriented screen analysis and file/component blueprint for a Smalltalk-style Goja inspector TUI.
LastUpdated: 2026-02-14T18:05:00Z
WhatFor: Guide implementation of a reusable Bubble Tea Smalltalk inspector using current ast-parse-editor, inspector, jsparse, and runtime probes.
WhenToUse: Use before and during implementation of GOJA-024 to map screens to concrete models, messages, and files.
---


# Smalltalk Goja Inspector Interface And Component Design

## Goal

Provide an implementation-ready blueprint for the Smalltalk-style inspector, including verbatim screen specs, per-screen behavior mapping, reusable Bubble Tea model decomposition, and a file-by-file code plan.

## Context

The ticket source was imported from `sources/local/smalltalk-goja-inspector.md` and defines eight target screens plus navigation principles.

Current code baseline used for this design:

- `cmd/ast-parse-editor`: strong live editing + CST/AST sync patterns and async parse sequencing.
- `cmd/inspector`: now refactored to include reusable Bubbles components from `GOJA-025`.
- `pkg/jsparse`: reusable index, resolution, completion context/candidate infrastructure.

GOJA-025 refactor updates now available and should be treated as mandatory reuse points:

- `go-go-goja/cmd/inspector/app/keymap.go`
  - mode-aware key definitions using `bobatea/pkg/mode-keymap`.
  - standardized `bubbles/help` integration.
- `go-go-goja/cmd/inspector/app/tree_list.go`
  - `bubbles/list` adapter for AST-node tree rows.
- `go-go-goja/cmd/inspector/app/model.go` (updated)
  - `bubbles/viewport` source rendering.
  - `bubbles/spinner` status signaling.
  - `bubbles/textinput` command mode (`:`).
  - `bubbles/table` metadata panel.

GOJA-025 implementation commits (for code archaeology):

- `3339aa86b12bbe450ebb01e184241fe2ff47a541` — keymap/help/spinner
- `13d7bbfcecd801d6bd75743826ac5e360f13062b` — viewport/list
- `8e1e1ce4b5c2f4caf3e202fb521bf3b4d2919f99` — textinput/table

Runtime probes were added in this ticket:

- `scripts/goja-runtime-probe.go`
- `scripts/jsparse-index-probe.go`

These probes validate that Goja and jsparse already provide enough primitives for prototype walking, symbol-key listing, descriptors, global binding discovery, and stack-trace text capture.

## Quick Reference

### Baseline Files And What They Already Solve

`go-go-goja/cmd/ast-parse-editor/main.go`
Entry point that opens file or scratch buffer, launches Bubble Tea alt screen.

`go-go-goja/cmd/ast-parse-editor/app/model.go`
Single large model with async AST parse (`parseSeq`, `pendingSeq`), CST parse, AST select mode, go-to-def, find usages, syntax highlighting, multi-pane rendering.

`go-go-goja/cmd/inspector/main.go`
Reads JS file, parses with goja parser, builds `jsparse.Index` and `Resolve`, launches inspector model.

`go-go-goja/cmd/inspector/app/model.go`
Two-pane static inspector plus drawer. Now already includes reusable Bubbles primitives (`help`, `spinner`, `viewport`, `list`, `table`, `textinput`) and mode-gated key handling.

`go-go-goja/cmd/inspector/app/keymap.go`
Central mode-aware key bindings and help model contract.

`go-go-goja/cmd/inspector/app/tree_list.go`
List adapter for AST tree rows, selection, and scope hint formatting.

`go-go-goja/cmd/inspector/app/drawer.go`
Reusable mini editor + CST panel + completion popup logic.

`go-go-goja/cmd/inspector/app/jsparse_bridge.go`
Type aliases and thin API bridge to `pkg/jsparse`.

`go-go-goja/pkg/jsparse/index.go`
AST node index, offset mapping, tree visibility and expand/collapse primitives.

`go-go-goja/pkg/jsparse/resolve.go`
Lexical scope graph and binding/reference resolution (`BindingForNode`, `AllUsages`).

`go-go-goja/pkg/jsparse/completion.go`
Completion context extraction from tree-sitter plus candidate synthesis.

`go-go-goja/pkg/jsparse/analyze.go`
Facade returning parse/index/resolution bundle.

### New Developer Architecture Map

If you are new to this codebase, start with this map before writing code.

Repository-level map:

- `go-go-goja/`
  - Primary project for this ticket.
  - Contains CLI commands and reusable analysis packages.
- `goja/`
  - JavaScript runtime implementation details (upstream-ish engine internals).
  - Usually not modified for GOJA-024 unless runtime API gaps are proven.
- `bobatea/`
  - Shared TUI primitives; GOJA-024 currently depends on `mode-keymap`.
- `glazed/`
  - CLI/documentation framework and tooling integration.

GOJA-024 implementation zones:

- Entry command:
  - `go-go-goja/cmd/smalltalk-inspector/` (new command to create)
- Existing reusable UI baseline:
  - `go-go-goja/cmd/inspector/app/`
- Static analysis engine:
  - `go-go-goja/pkg/jsparse/`
- New runtime/introspection domain logic (to add):
  - `go-go-goja/pkg/inspector/runtime/`
- New UI/domain orchestration (to add):
  - `go-go-goja/pkg/inspector/...`

Where to look for key behaviors:

- Source ↔ AST sync: `go-go-goja/cmd/inspector/app/model.go`
- AST/CST parse and async parse sequence: `go-go-goja/cmd/ast-parse-editor/app/model.go`
- Binding/usages logic: `go-go-goja/pkg/jsparse/resolve.go`
- Completion context/candidates: `go-go-goja/pkg/jsparse/completion.go`
- Runtime object/prototype/symbol APIs: `goja/value.go` (`Symbols`, `Prototype`, `GetOwnPropertyNames`)
- Stack capture and exception shape: `goja/runtime.go`

### Probe Outputs (Implementation Constraints)

From `scripts/goja-runtime-probe.go`:

```text
== Dog instance own keys ==
GetOwnPropertyNames=[name alive energy breed]
Keys=[name alive energy breed]
Symbols=[]
...
== config keys and symbols ==
GetOwnPropertyNames=[apiUrl timeout retries debug]
Symbols=[Symbol.iterator Symbol.toPrimitive custom]
...
== thrown error stack string ==
TypeError: Cannot read property 'calories' of undefined
	at eat (<eval>:25:25(13))
	at fetch (<eval>:46:20(4))
	at <eval>:1:21(4)
```

From `scripts/jsparse-index-probe.go`:

```text
== Global Bindings ==
- Animal kind=class ...
- Dog kind=class ...
- greet kind=function ...
- main kind=function ...
== Unresolved Identifiers ==
- "Error" ...
```

Design implication:

- Screen 7 symbol-key section is fully feasible with `Object.Symbols()`.
- Screen 6 prototype walk is fully feasible with `Object.Prototype()`.
- Screen 8 stack trace can be built from `*goja.Exception` text immediately, and upgraded later to richer frame objects.
- Screen 2 global binding list maps directly to `Resolution` root scope bindings.

## Interface And Behavior By Screen

### Screen 1 — Startup / Load File

The user launches the inspector and is greeted with a file loader + empty workspace. The REPL is always at the bottom.

```
┌─── goja-inspector ──────────────────────────────────────────────────────────┐
│                                                                             │
│                                                                             │
│                         No program loaded.                                  │
│                                                                             │
│              :load <file.js>    Load a JavaScript file                      │
│              :run  <expr>       Evaluate an expression                      │
│              :help              Show all commands                           │
│                                                                             │
│                                                                             │
│                                                                             │
│                                                                             │
│                                                                             │
│                                                                             │
│                                                                             │
│                                                                             │
│                                                                             │
│                                                                             │
├─── REPL ────────────────────────────────────────────────────────────────────┤
│ » :load app.js                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

Implementation analysis:

- Root model starts in `ModeEmpty` with only command palette + REPL enabled.
- `:load` dispatches `LoadFileCmd(path)` that performs parse/index/resolution build and runtime bootstrap.
- Reuse: file loading and parse bootstrap logic from `cmd/inspector/main.go`, but move into model command path so loading can happen in-app.
- Minimal required state is `currentFile`, `source`, `analysis`, `runtimeSession`, `replHistory`, and `statusLine`.

### Screen 2 — File Loaded: Global Scope Overview

After loading, the left pane shows **all top-level bindings** (globals, classes, functions). The right pane is empty until something is selected. This is the "Class Browser" root.

```
┌─── goja-inspector ─── app.js ───────────────────────────────────────────────┐
│ ┌─ Globals ──────────────┐  ┌─ Members ─────────────┐  ┌─ Source ────────┐ │
│ │                        │  │                        │  │                 │ │
│ │  ƒ  greet              │  │   (select a global)    │  │  (no source)    │ │
│ │  C  Animal             │  │                        │  │                 │ │
│ │  C  Dog  ← Animal      │  │                        │  │                 │ │
│ │  ƒ  main               │  │                        │  │                 │ │
│ │  ●  config      {…}    │  │                        │  │                 │ │
│ │  ●  API_KEY     "sk-…" │  │                        │  │                 │ │
│ │  ●  version     3      │  │                        │  │                 │ │
│ │                        │  │                        │  │                 │ │
│ │                        │  │                        │  │                 │ │
│ │                        │  │                        │  │                 │ │
│ │  8 bindings found      │  │                        │  │                 │ │
│ └────────────────────────┘  └────────────────────────┘  └─────────────────┘ │
├─── REPL ────────────────────────────────────────────────────────────────────┤
│ ✓ Loaded app.js (245 lines, 8 globals)                                      │
│ » ▌                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

Implementation analysis:

- `GlobalsModel` data comes from `analysis.Resolution.Scopes[RootScopeID].Bindings` plus runtime global object snapshots for values not statically declared.
- For each item show kind icon (`function`, `class`, `value`) and optional extends info.
- Selection event publishes `MsgGlobalSelected{BindingName, DeclNodeID, RuntimeValueRef}`.
- Source pane stays empty until selected member/function maps to source span.

### Screen 3 — Select a Class: Member List Populates

User highlights `Dog` in the left pane. The middle pane fills with its **own members + inherited** members. Prototype chain is shown.

```
┌─── goja-inspector ─── app.js ───────────────────────────────────────────────┐
│ ┌─ Globals ──────────────┐  ┌─ Dog: Members ────────┐  ┌─ Source ────────┐ │
│ │                        │  │                        │  │                 │ │
│ │   ƒ  greet             │  │ ── own ──────────────  │  │  (select a      │ │
│ │   C  Animal            │  │  ƒ  constructor(name)  │  │   member)       │ │
│ │ ▸ C  Dog  ← Animal     │  │  ƒ  bark()            │  │                 │ │
│ │   ƒ  main              │  │  ƒ  fetch(item)       │  │                 │ │
│ │   ●  config      {…}   │  │  ●  breed    "lab"     │  │                 │ │
│ │   ●  API_KEY     "sk-… │  │                        │  │                 │ │
│ │   ●  version     3     │  │ ── inherited (Animal)  │  │                 │ │
│ │                        │  │  ƒ  eat(food)          │  │                 │ │
│ │                        │  │  ƒ  sleep()            │  │                 │ │
│ │                        │  │  ●  alive     true     │  │                 │ │
│ │                        │  │                        │  │                 │ │
│ │                        │  │ proto: Animal → Object │  │                 │ │
│ └────────────────────────┘  └────────────────────────┘  └─────────────────┘ │
├─── REPL ────────────────────────────────────────────────────────────────────┤
│ » ▌                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key:** The `▸` marker shows the currently selected item. Prototype chain is rendered at the bottom of the members pane.

Implementation analysis:

- `MembersModel` receives selected global class/object and builds grouped rows: own, inherited, metadata.
- Own/inherited separation requires runtime reflection walk: inspect object own properties, then walk `Prototype()` and diff by first occurrence.
- Static method signatures come from AST class declaration in `jsparse.Index` if available; fallback to runtime property labels if no static mapping.
- Reuse: AST span lookup patterns from `inspector` for def jumps.

### Screen 4 — Select a Method: Source + Symbols Appear

User selects `bark()`. The right pane shows **source code**, and a **symbols sub-panel** appears below it with local bindings.

```
┌─── goja-inspector ─── app.js ───────────────────────────────────────────────┐
│ ┌─ Globals ──────────┐ ┌─ Dog: Members ──────┐ ┌─ Source: bark() ────────┐ │
│ │                    │ │                      │ │                         │ │
│ │  ƒ  greet          │ │ ── own ────────────  │ │  43│ bark() {            │ │
│ │  C  Animal         │ │  ƒ  constructor()    │ │  44│   const sound =     │ │
│ │▸ C  Dog ← Animal   │ │ ▸ƒ  bark()          │ │  45│     this.breed ===  │ │
│ │  ƒ  main           │ │  ƒ  fetch(item)     │ │  46│     "husky"         │ │
│ │  ●  config    {…}  │ │  ●  breed   "lab"   │ │  47│     ? "awoo" : "wo… │ │
│ │  ●  API_KEY   "sk… │ │                      │ │  48│   console.log(      │ │
│ │  ●  version   3    │ │ ── inherited ──────  │ │  49│     `${this.name}:  │ │
│ │                    │ │  ƒ  eat(food)        │ │  50│      ${sound}`);    │ │
│ │                    │ │  ƒ  sleep()          │ │  51│   return sound;     │ │
│ │                    │ │  ●  alive    true    │ │  52│ }                   │ │
│ │                    │ │                      │ ├─ Symbols ──────────────┤ │
│ │                    │ │                      │ │ sound  const  L44  2u  │ │
│ │                    │ │                      │ │ this   recv   —    4u  │ │
│ └────────────────────┘ └──────────────────────┘ └─────────────────────────┘ │
├─── REPL ────────────────────────────────────────────────────────────────────┤
│ » ▌                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Symbols sub-panel:** `L44` = defined on line 44, `2u` = 2 usages found. This comes from Layer B (`BuildIndex` + `Resolve`).

Implementation analysis:

- `SourcePanelModel` renders selected span with syntax style and optional target line marker.
- `SymbolTableModel` computes symbols limited to selected method subtree: filter `Resolve` bindings to descendant node IDs of selected method node.
- Usage count is direct from `BindingRecord.AllUsages()` count constrained by method span.
- Row action `Enter` on symbol triggers highlight overlays in source pane.

### Screen 5 — REPL Evaluation: Live Object Inspection

User types an expression in the REPL. The result becomes an **inspectable object** that replaces or augments the Globals pane (shown here as a dedicated result panel).

```
┌─── goja-inspector ─── app.js ───────────────────────────────────────────────┐
│ ┌─ REPL Result ──────────────────────────────┐  ┌─ Source ────────────────┐ │
│ │                                            │  │                         │ │
│ │  new Dog("Rex")                            │  │                         │ │
│ │  → Dog {                                   │  │  (select member to      │ │
│ │      name      : "Rex"           string    │  │   view source)          │ │
│ │      breed     : "lab"           string    │  │                         │ │
│ │    ▸ bark      : ƒ bark()        function  │  │                         │ │
│ │      fetch     : ƒ fetch(item)   function  │  │                         │ │
│ │      eat       : ƒ eat(food)     function  │  │                         │ │
│ │      sleep     : ƒ sleep()       function  │  │                         │ │
│ │      alive     : true            boolean   │  │                         │ │
│ │      [[Proto]] : Animal.prototype          │  │                         │ │
│ │                                            │  │                         │ │
│ │  [Tab: expand]  [Enter: inspect]  [s: src] │  │                         │ │
│ └────────────────────────────────────────────┘  └─────────────────────────┘ │
├─── REPL ────────────────────────────────────────────────────────────────────┤
│   new Dog("Rex")                                                            │
│ » ▌                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Flow:** The REPL result is a **live runtime object**. Selecting `bark` here would show the same source as Screen 4. Selecting `[[Proto]]` would drill into `Animal.prototype`.

Implementation analysis:

- `ReplModel` owns input buffer, history, eval command parsing, and emits `MsgEvalResult` or `MsgEvalError`.
- `ObjectBrowserModel` renders exported preview rows for objects and functions.
- Source jump for runtime function rows requires function-to-AST mapping strategy.
- Mapping strategy v1 should be name-based + class context and then span fallback via declaration lookup.

### Screen 6 — Drill Into Prototype / Deep Inspect

User selects `[[Proto]]` to walk the prototype chain. The inspector now shows `Animal.prototype` as the inspected target.

```
┌─── goja-inspector ─── app.js ───────────────────────────────────────────────┐
│ ┌─ Breadcrumb: Dog instance → [[Proto]] ─────────────────────────────────┐  │
│ └────────────────────────────────────────────────────────────────────────┘  │
│ ┌─ Animal.prototype ─────────────────────────┐  ┌─ Source: eat(food) ────┐ │
│ │                                            │  │                        │ │
│ │   constructor : ƒ Animal(name)   function  │  │ 12│ eat(food) {        │ │
│ │ ▸ eat         : ƒ eat(food)      function  │  │ 13│   if (!this.alive) │ │
│ │   sleep       : ƒ sleep()        function  │  │ 14│     throw new      │ │
│ │   alive       : true             boolean   │  │ 15│       Error("…");  │ │
│ │   [[Proto]]   : Object.prototype           │  │ 16│   this.energy +=   │ │
│ │                                            │  │ 17│     food.calories; │ │
│ │                                            │  │ 18│   return this;     │ │
│ │                                            │  │ 19│ }                  │ │
│ │                                            │  ├─ Symbols ─────────────┤ │
│ │                                            │  │ food    param L12  3u │ │
│ │                                            │  │ this    recv  —    2u │ │
│ │  [Esc: back to Dog]  [Enter: inspect]      │  │                      │ │
│ └────────────────────────────────────────────┘  └────────────────────────┘ │
├─── REPL ────────────────────────────────────────────────────────────────────┤
│ » ▌                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key:** The **breadcrumb bar** at the top tracks navigation history. `Esc` goes back. This is the "continue exploring forever" Smalltalk behavior.

Implementation analysis:

- `NavigationStackModel` stores `InspectTarget` frames and supports push/pop/replace.
- Each frame carries enough identity to restore selection and scroll state.
- `InspectTarget` union must support runtime object refs, AST node refs, stack frame locals, and cross-reference rows.
- Back navigation must be lossless, including previous source highlight and selected member row.

### Screen 7 — Symbol-Keyed Properties & Cross-References

User inspects an object with Symbol keys (Layer A: `obj.Symbols()`). Also shows the cross-reference / usages panel.

```
┌─── goja-inspector ─── app.js ───────────────────────────────────────────────┐
│ ┌─ Breadcrumb: config ───────────────────────────────────────────────────┐  │
│ └────────────────────────────────────────────────────────────────────────┘  │
│ ┌─ config: Members ──────────────────────────┐  ┌─ Cross-References ────┐ │
│ │                                            │  │                        │ │
│ │ ── string keys ──────────────────────────  │  │ config.apiUrl          │ │
│ │   apiUrl    : "https://…"        string    │  │ ── usages (4 found) ── │ │
│ │   timeout   : 3000               number    │  │                        │ │
│ │   retries   : 5                  number    │  │  L67  main()           │ │
│ │   debug     : false              boolean   │  │  L89  Animal.eat()     │ │
│ │                                            │  │  L134 fetchData()      │ │
│ │ ── symbol keys ──────────────────────────  │  │  L201 init()           │ │
│ │   [Symbol.iterator]  : ƒ        function   │  │                        │ │
│ │   [Symbol.toPrim…]   : ƒ        function   │  │ [Enter: jump to usage] │ │
│ │   [Symbol(custom)]   : "meta"   string     │  │                        │ │
│ │                                            │  │                        │ │
│ │ ── descriptors ──────────────────────────  │  │                        │ │
│ │   apiUrl: {writable:T enum:T config:T}     │  │                        │ │
│ └────────────────────────────────────────────┘  └────────────────────────┘ │
├─── REPL ────────────────────────────────────────────────────────────────────┤
│ » config.apiUrl                                                             │
│ → "https://api.example.com/v3"                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

Implementation analysis:

- String keys from `GetOwnPropertyNames()`, symbols from `Symbols()`.
- Descriptor details should be read through JS helper call `Object.getOwnPropertyDescriptor` for both string and symbol keys.
- `CrossRefModel` should consume `jsparse.Resolve` references when object maps to static binding.
- For runtime-only values with no static anchor, cross-ref panel should explicitly render `no static references available`.

### Screen 8 — Error / Stack Trace Inspection

When an expression throws, the REPL catches it and shows the **call stack** as an inspectable entity (Layer A: `CaptureCallStack`).

```
┌─── goja-inspector ─── app.js ───────────────────────────────────────────────┐
│ ┌─ Error ────────────────────────────────────────────────────────────────┐  │
│ │  TypeError: Cannot read property 'calories' of undefined               │  │
│ └────────────────────────────────────────────────────────────────────────┘  │
│ ┌─ Call Stack ───────────────────────────────┐  ┌─ Source: eat() ────────┐ │
│ │                                            │  │                        │ │
│ │ ▸ #0  Animal.eat        app.js:17:5        │  │  15│     Error("…");   │ │
│ │   #1  Dog.fetch         app.js:56:12       │  │  16│   this.energy +=  │ │
│ │   #2  main              app.js:72:3        │  │ →17│     food.calories │ │
│ │   #3  (global)          app.js:80:1        │  │  18│   return this;    │ │
│ │                                            │  │  19│ }                 │ │
│ │                                            │  │                        │ │
│ │ ── locals at #0 ───────────────────────    │  │  → marks error line    │ │
│ │   this   : Dog { name: "Rex" }             │  │                        │ │
│ │   food   : undefined                       │  │                        │ │
│ │                                            │  │                        │ │
│ │  [Enter: inspect value]  [↑↓: walk stack]  │  │                        │ │
│ └────────────────────────────────────────────┘  └────────────────────────┘ │
├─── REPL ────────────────────────────────────────────────────────────────────┤
│ ✗ new Dog("Rex").fetch()                                                    │
│ » ▌                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Flow:** Arrow keys walk the stack frames. Selecting `this` or `food` in the locals section opens them as inspectable objects (back to Screen 5 style). Selecting a different stack frame updates the source pane.

Implementation analysis:

- Immediate implementation uses `*goja.Exception` string parse to extract message and frame locations.
- Upgrade path uses runtime-side helper wrappers around `CaptureCallStack` where possible during function calls.
- Frame selection publishes `MsgStackFrameSelected` to source panel.
- Locals inspection requires runtime instrumentation wrappers. This should be marked `phase-2` with graceful placeholder in phase-1.

## Component System Design

### Model Decomposition

`RootModel`
Orchestrates focus, layout, active mode, shared stores, and command routing.

`GlobalsModel`
Shows static root bindings and global runtime values.

`MembersModel`
Shows selected target members grouped into own/inherited/symbol/descriptors.

`SourcePanelModel`
Displays source slice, current highlight range, error line, and optional symbol table subsection.

`SymbolTableModel`
Displays local symbols for selected method with kind and usage counts.

`CrossRefModel`
Displays usages and supports jump-to-usage.

`ReplModel`
Maintains input/history/result/error and emits eval messages.

`ObjectBrowserModel`
Renders inspectable values and member rows.

`NavigationStackModel`
Maintains breadcrumb frames and `Esc` back behavior.

`StackTraceModel`
Renders error stack entries and selected frame.

`StatusModel`
Renders contextual status + transient notices.

### Reusable UI Components

`PaneFrame`
Title bar, focused border style, body clipping.

`SelectableList`
Keyboard selection, scroll windowing, optional sections.

`CodeView`
Gutter, line highlight, char-range highlight.

`InlineTable`
Two-to-four column compact row display for symbols/descriptors.

`BreadcrumbBar`
Truncated path segments with active item styling.

`CommandLine`
REPL input with prompt, history, transient result line.

### Reuse And Refactor Matrix (GOJA-025 Baseline)

`mode-keymap` + `help`
Reuse `cmd/inspector/app/keymap.go` patterns directly for mode-scoped bindings (`global`, `members`, `source`, `repl`, `stack`) and keep a single help footer contract.

`list`
Reuse the approach in `cmd/inspector/app/tree_list.go`: domain row -> `list.Item` adapter, centralized `FilterValue`, and single owner for selected-row mapping.

`viewport`
Reuse source-pane behavior from `cmd/inspector/app/model.go` (`viewport.Model` ownership in panel model, not in root). Refactor only line-highlight handling into `pkg/inspector/ui/components/code_view.go`.

`table`
Reuse metadata panel/table shape from `cmd/inspector/app/model.go` for descriptors, locals, and symbol-key rows. Refactor column factories into one shared helper to avoid per-pane column drift.

`spinner`
Reuse spinner lifecycle/status messaging from `cmd/inspector/app/model.go` for async load/eval states (`loading file`, `rebuilding analysis`, `evaluating`).

`textinput`
Reuse command-mode input patterns from `cmd/inspector/app/model.go` for `:load`/`:run`, and keep REPL input as a separate model that shares history/edit key behavior.

Refactor scope to avoid overcoupling:
- Do not copy the existing inspector root model wholesale.
- Extract tiny reusable constructors/helpers first, then compose them in `cmd/smalltalk-inspector/app`.
- Keep `pkg/inspector/*` UI-agnostic where possible (especially runtime and analysis packages).

### Message Contracts

`MsgFileLoaded`
Carries filename, source, analysis bundle, runtime session state.

`MsgGlobalSelected`
Carries selected root binding and optional runtime reference.

`MsgMemberSelected`
Carries member identity and optional source span.

`MsgInspectTargetPushed`
Carries new inspect target frame.

`MsgInspectBack`
Pops one breadcrumb frame.

`MsgEvalResult`
Carries expression text and runtime value reference.

`MsgEvalError`
Carries expression text and parsed stack metadata.

`MsgJumpToSource`
Carries filename + line/column/span.

`MsgHighlightBinding`
Carries binding id and usage ids for source overlay.

### State Stores

`AnalysisStore`
`jsparse` result cache for current file.

`RuntimeStore`
Goja runtime and reference registry for inspectable values.

`NavigationStore`
Breadcrumb frames and per-frame UI snapshot.

`SelectionStore`
Current global/member/symbol/xref/stack selections.

## Developer Onboarding Runbook

### First 30 Minutes (Find Your Marks)

1. Read this document end-to-end once.
2. Open and skim:
   - `go-go-goja/cmd/inspector/app/model.go`
   - `go-go-goja/cmd/inspector/app/keymap.go`
   - `go-go-goja/cmd/inspector/app/tree_list.go`
   - `go-go-goja/pkg/jsparse/index.go`
   - `go-go-goja/pkg/jsparse/resolve.go`
3. Run baseline tests:
   - `cd go-go-goja && go test ./cmd/inspector/... -count=1`
   - `cd go-go-goja && go test ./pkg/jsparse -count=1`
4. Run existing inspector to understand interaction baseline:
   - `cd go-go-goja && go run ./cmd/inspector ../path/to/file.js`

### Fast Code Navigation Commands

Run these from `go-go-goja/` to find key implementation marks quickly:

- `rg -n "type keyMap|ShortHelp|FullHelp" cmd/inspector/app`
- `rg -n "spinner|textinput|viewport|table|list" cmd/inspector/app/model.go`
- `rg -n "BindingForNode|AllUsages|WalkScopeChain" pkg/jsparse`
- `rg -n "Symbols\\(|Prototype\\(|GetOwnPropertyNames\\(" ../goja`
- `rg -n "TODO|FIXME|GOJA-024|smalltalk-inspector" cmd pkg`

### Architectural Boundaries (Do Not Blur)

- `cmd/*` directories:
  - CLI glue and high-level wiring only.
  - Avoid putting deep business logic here.
- `pkg/jsparse`:
  - Static parse/index/resolve/completion logic.
  - Keep this reusable and UI-agnostic.
- `pkg/inspector` (new):
  - Smalltalk-inspector domain and UI orchestration.
  - Prefer this over adding more monolith logic into `cmd/inspector/app`.
- `goja/` runtime internals:
  - Modify only if unavoidable and justified by missing API capability.

### New Developer Checklist Before Opening First PR

- Confirm your changes do not regress existing `cmd/inspector` behavior.
- Keep new Smalltalk features behind the new command path.
- Add/update tests in the same directory as changed logic.
- Update GOJA-024 `tasks.md`, `changelog.md`, and `reference/01-diary.md` at each meaningful milestone.

## File-By-File Implementation Blueprint

### Command Entry And Wiring

`cmd/smalltalk-inspector/main.go`
CLI args, startup config, launches root model with alt screen.

`cmd/smalltalk-inspector/app/model.go`
Root model struct fields for all child models and shared stores.

`cmd/smalltalk-inspector/app/update.go`
Main message routing and command chaining logic.

`cmd/smalltalk-inspector/app/view.go`
Top-level layout assembly and pane switching by mode.

`cmd/smalltalk-inspector/app/keymap.go`
Canonical key definitions and help text per mode.

`cmd/smalltalk-inspector/app/messages.go`
Shared typed message structs used across components.

`cmd/smalltalk-inspector/app/styles.go`
Central lipgloss style definitions.

`cmd/smalltalk-inspector/app/reuse_inspector_components.go` (or equivalent wiring module)
Imports/adapts reusable logic from `cmd/inspector/app` where practical (keymap patterns, list/view/table setup, viewport behaviors).

### Domain Layer

`pkg/inspector/analysis/session.go`
Build/refresh analysis from source using `jsparse.Analyze`.

`pkg/inspector/analysis/method_symbols.go`
Resolve method-local symbol rows and usage counts.

`pkg/inspector/analysis/xref.go`
Binding-to-usage row mapping with jump targets.

`pkg/inspector/runtime/session.go`
Owns `goja.Runtime`, load/run/eval helpers, and value registry.

`pkg/inspector/runtime/introspect.go`
Property, symbol, descriptor, and prototype helpers.

`pkg/inspector/runtime/errors.go`
Normalize `*goja.Exception` into UI stack/error rows.

`pkg/inspector/runtime/function_map.go`
Map runtime function/method selections to static source spans.

`pkg/inspector/navigation/targets.go`
Defines `InspectTarget` union and breadcrumb frame payloads.

### Component Layer

`pkg/inspector/ui/components/selectable_list.go`
Optional thin wrapper around `bubbles/list` for inspector-specific sections and icons.

`pkg/inspector/ui/components/code_view.go`
Reusable source rendering with line/char highlights built on `bubbles/viewport`.

`pkg/inspector/ui/components/pane_frame.go`
Pane wrapper and focus border rendering.

`pkg/inspector/ui/components/breadcrumb.go`
Breadcrumb row rendering and truncation.

`pkg/inspector/ui/components/repl_line.go`
Prompt line, input editing, and history behavior built on `bubbles/textinput`.

### Feature Models

`pkg/inspector/ui/models/globals_model.go`
Build/render/update root binding list.

`pkg/inspector/ui/models/members_model.go`
Build/render/update selected target members.

`pkg/inspector/ui/models/source_model.go`
Render source and manage jump/highlight.

`pkg/inspector/ui/models/symbols_model.go`
Render method-local symbol rows.

`pkg/inspector/ui/models/xref_model.go`
Render xref rows and jump selection.

`pkg/inspector/ui/models/object_model.go`
Render runtime object member table.

`pkg/inspector/ui/models/repl_model.go`
Input/history/eval dispatch/result presentation.

`pkg/inspector/ui/models/stack_model.go`
Error stack list and frame selection.

`pkg/inspector/ui/models/status_model.go`
Status and transient notices.

## Delivery Phases

Phase 0 (completed in GOJA-025)
Reusable inspector component baseline in `cmd/inspector/app`:
- mode-aware keymap (`mode-keymap`)
- `bubbles/help`, `bubbles/spinner`
- `bubbles/viewport`, `bubbles/list`
- `bubbles/textinput`, `bubbles/table`

Phase 1
Implement Smalltalk command skeleton and Screens 1 to 4 on top of the Phase-0 baseline.

Phase 2
Implement Screens 5 to 7 with runtime object browsing, prototype navigation, and descriptor/symbol-key panes.

Phase 3
Implement Screen 8 stack trace browsing, then harden tests/docs and extract stable `pkg/inspector` shared packages.

## Usage Examples

Example implementation loop:

```bash
cd go-go-goja

go test ./cmd/inspector/... -count=1

go test ./pkg/jsparse -count=1

go run ./ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/scripts/goja-runtime-probe.go

go run ./ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/scripts/jsparse-index-probe.go
```

Example design-to-code mapping:

- Screen 2 globals list -> `globals_model.go` + `analysis/session.go`
- Screen 5 REPL object inspect -> `repl_model.go` + `runtime/session.go` + `object_model.go`
- Screen 8 stack trace -> `runtime/errors.go` + `stack_model.go` + `source_model.go`

## Related

- `sources/local/smalltalk-goja-inspector.md`
- `reference/01-diary.md`
- `go-go-goja/cmd/inspector/app/model.go`
- `go-go-goja/cmd/inspector/app/keymap.go`
- `go-go-goja/cmd/inspector/app/tree_list.go`
- `go-go-goja/cmd/ast-parse-editor/app/model.go`
- `go-go-goja/pkg/jsparse/index.go`
- `go-go-goja/pkg/jsparse/resolve.go`
- `go-go-goja/pkg/jsparse/completion.go`
- `go-go-goja/ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/reference/01-inspector-refactor-design-guide.md`
