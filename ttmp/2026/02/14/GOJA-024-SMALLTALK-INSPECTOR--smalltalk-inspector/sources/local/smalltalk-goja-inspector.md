Let me read the task document first.Now I have a solid picture. Here's a full set of TUI mockups with flow explanations.

---

## Screen 1 — Startup / Load File

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

---

## Screen 2 — File Loaded: Global Scope Overview

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

**Legend:** `ƒ` = function, `C` = class, `●` = value/object. Arrow `← Animal` shows extends.

---

## Screen 3 — Select a Class: Member List Populates

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

---

## Screen 4 — Select a Method: Source + Symbols Appear

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

---

## Screen 5 — REPL Evaluation: Live Object Inspection

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

---

## Screen 6 — Drill Into Prototype / Deep Inspect

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

---

## Screen 7 — Symbol-Keyed Properties & Cross-References

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

---

## Screen 8 — Error / Stack Trace Inspection

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

---

## Flow Summary

```
┌──────────────┐     :load file.js     ┌──────────────────┐
│   Startup    │ ────────────────────▸  │  Global Overview  │
│  (Screen 1)  │                       │    (Screen 2)     │
└──────────────┘                       └────────┬─────────┘
                                                │
                                     select class/object
                                                │
                                                ▾
                                       ┌──────────────────┐
                                       │  Member List +    │
                                       │  Source + Symbols │
                                       │  (Screens 3–4)   │
                                       └───────┬──────────┘
                                               │
                              ┌────────────────┼────────────────┐
                              │                │                │
                     select [[Proto]]    select symbol     REPL eval
                              │           property              │
                              ▾                │                ▾
                     ┌────────────────┐        ▾       ┌────────────────┐
                     │ Prototype Walk │  ┌──────────┐  │  Live Object   │
                     │  (Screen 6)    │  │ Sym keys │  │  Inspection    │
                     └────────┬───────┘  │(Screen 7)│  │  (Screen 5)   │
                              │          └──────────┘  └───────┬────────┘
                              │                                │
                              └──────────┬─────────────────────┘
                                         │
                                   error thrown?
                                    ┌────┴─────┐
                                    │  no  yes │
                                    │      │   │
                                    │      ▾   │
                                    │ ┌──────────────┐
                                    │ │  Stack Trace  │
                                    │ │  (Screen 8)   │
                                    │ └──────┬───────┘
                                    │        │
                                    └────────┘
                                         │
                                         ▾
                              ┌─────────────────────┐
                              │  inspect any value   │──┐
                              │  from any context    │  │ ← recursive:
                              │  (back to Screen 5)  │◂─┘   "explore forever"
                              └─────────────────────┘
```

### Core Navigation Principles

**Three-pane Smalltalk layout:** Globals (or breadcrumb target) → Members → Source+Symbols. This mirrors the classic Class → Method → Code browser.

**REPL is always present** at the bottom. Any expression result becomes an inspectable object, bridging static browsing and live runtime exploration.

**Breadcrumb navigation** tracks where you've been. `Esc` always goes back. Drilling into `[[Proto]]`, a member value, a stack local, or a cross-reference target all push onto the breadcrumb stack.

**Layers are invisible to the user** — the inspector silently uses Layer A (public API) for runtime values, Layer B (inspector package) for symbols/usages/go-to-definition, and Layer C (private internals) for deep source mapping. The user just sees a unified browsing experience.
