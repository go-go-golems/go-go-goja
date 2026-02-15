---
title: "Fix Plan"
doc_type: design
status: active
intent: long-term
topics:
  - smalltalk-inspector
  - bugs
  - debugging
created: 2026-02-14T13:10:00-05:00
---

# Fix Plan — Smalltalk Inspector Bugs

## Historical Status (Updated 2026-02-15)

This document captures the tactical fix plan used during GOJA-026. It is preserved for implementation history, but parts of the orchestration guidance are superseded by later architectural work that introduced `pkg/inspectorapi` and extracted reusable inspector packages.

Use this page to understand why fixes were made, not as the primary integration guide for current code.

## Overview

Analysis uncovered **6 bugs** (3 original + 3 found during analysis). They share a common root cause: **the globals/members layer is static-only** and doesn't leverage the runtime session. The fix strategy is to bridge runtime introspection into the existing static data structures, preserving the Smalltalk-style class browser for classes while adding runtime-aware display for everything else.

## Complete Bug Inventory

| # | Bug | Severity | Category |
|---|-----|----------|----------|
| 1 | Non-class globals show "(no members)" | Medium | Static-only members |
| 2 | REPL definitions don't refresh globals list | Medium | No post-eval refresh |
| 3 | Enter on value globals does nothing | Low-Medium | Missing interaction |
| 4 | Runtime session created after buildMembers() on load | Low | Init ordering |
| 5 | Proto chain footer only shows static `Extends` | Low | Static-only view |
| 6 | navStack not cleared on new REPL eval | Medium | Stale state |

### Bug 4 (NEW): Init ordering

In `MsgFileLoaded` handler, `buildGlobals()` and `buildMembers()` run on lines 39–40, but `m.rtSession` is created on lines 43–44. Any runtime-aware `buildMembers()` would get a nil session on initial load.

**Fix:** Move runtime session initialization *before* `buildGlobals()`/`buildMembers()`.

### Bug 5 (NEW): Proto chain footer only uses static `Extends`

The `renderMembersPane` footer shows `proto: Shape → Object` but only for classes (via `g.Extends` from jsparse). For `myCircle` it shows nothing, even though the runtime knows the full chain `Circle → Shape → Object`.

**Fix:** When displaying a non-class global, query the runtime for the prototype chain and display it in the footer.

### Bug 6 (NEW): navStack not cleared on new eval

When a new REPL expression is evaluated, the handler replaces `inspectObj`/`inspectProps` but doesn't clear `navStack`. If you'd drilled 3 levels into a previous result and then eval something new, pressing Esc tries to restore stale old frames.

**Fix:** Add `m.navStack = nil` in both success and error paths of `MsgEvalResult`.

## Critical Discovery: const/let vs var in goja

`goja.Runtime.GlobalObject().Keys()` only returns `var` and `function` declarations. `const` and `let` declarations are lexical — they're NOT on the global object. However, `vm.Get(name)` (used by `Session.GlobalValue()`) works for all binding types.

**Implications for Bug 2 fix:**
- Can't discover REPL-defined `const`/`let` names just by enumerating the global object
- Must parse the REPL input to extract declared names (simple regex/AST)
- `var`/`function` definitions CAN be discovered via `GlobalObject().Keys()`
- Best approach: combine both strategies — scan `GlobalObject` for new `var`/`function` names, AND parse REPL input for `const`/`let`/`class` names

## Fix Plan — Execution Order

### Step 1: Fix Bug 6 — navStack leak (trivial, unblocks safe testing)

**File:** `cmd/smalltalk-inspector/app/update.go`

Add `m.navStack = nil` in both the error and success branches of `MsgEvalResult`:

```go
// In error branch (after m.inspectProps = nil):
m.navStack = nil

// In success branch (after m.inspectLabel = ...):
m.navStack = nil
```

**Risk:** None. **Test:** Eval `settings`, drill into `[[Proto]]`, eval `myCircle` — Esc should clear cleanly, not restore stale settings proto state.

### Step 2: Fix Bug 4 — Init ordering (trivial, unblocks Bug 1 fix)

**File:** `cmd/smalltalk-inspector/app/update.go`

Move the runtime session initialization block *before* `buildGlobals()` and `buildMembers()`:

```go
case MsgFileLoaded:
    m.filename = msg.Filename
    m.source = msg.Source
    m.analysis = msg.Analysis
    m.sourceLines = strings.Split(msg.Source, "\n")
    m.loaded = true

    // Initialize runtime session FIRST
    m.rtSession = runtime.NewSession()
    if err := m.rtSession.Load(msg.Source); err != nil {
        m.statusMsg = fmt.Sprintf("... ⚠ runtime: %v", ...)
    }

    // Now build globals and members (can use runtime)
    m.buildGlobals()
    m.buildMembers()
    ...
```

**Risk:** None — just reordering. **Test:** Load a file, verify globals still appear correctly.

### Step 3: Fix Bug 1 — Runtime members for value globals (main fix)

**File:** `cmd/smalltalk-inspector/app/model.go`

Add a `default` case to `buildMembers()` and a new `buildValueMembers()` method:

```go
//exhaustive:ignore
switch selected.Kind {
case jsparse.BindingClass:
    m.buildClassMembers(selected.Name)
case jsparse.BindingFunction:
    m.buildFunctionMembers(selected.Name)
default:
    m.buildValueMembers(selected.Name)
}
```

`buildValueMembers(name string)`:
1. Guard: `if m.rtSession == nil { return }` (handles pre-runtime state gracefully)
2. `val := m.rtSession.GlobalValue(name)` — get the runtime value
3. If nil/undefined: add a single member `{Name: "(value)", Kind: "value", Preview: "undefined"}`
4. If primitive (string/number/boolean): add a single member `{Name: "(value)", Kind: "value", Preview: preview}`
5. If object: call `runtime.InspectObject(obj, vm)` → convert each `PropertyInfo` to `MemberItem`:
   - `Name` = prop.Name
   - `Kind` = prop.Kind (already "function"/"string"/"number"/etc)
   - `Preview` = " : " + prop.Preview (to match inspect view style)
   - New field `RuntimeDerived: true` to distinguish from static members

Add `RuntimeDerived bool` to `MemberItem` struct. This flag:
- Prevents `jumpToMember()` from trying AST lookup (which would fail)
- Can be used by view to show a subtle visual distinction if desired
- Tells the Enter handler to open runtime inspect view instead of source-jump

**jumpToSource() change:** In `jumpToSource()`, when focus is on Members and the member is `RuntimeDerived`, skip the `jumpToMember()` call (no source location exists for runtime properties).

**Risk:** Low. `InspectObject` is already well-tested. **Test:** Navigate to `settings` → see `theme`, `fontSize`, `lineNumbers`. Navigate to `myCircle` → see `color`, `visible`, `radius`.

### Step 4: Fix Bug 5 — Runtime proto chain in members footer

**File:** `cmd/smalltalk-inspector/app/view.go`

In `renderMembersPane()`, replace the static-only proto chain footer logic:

```go
// Current (static only):
if g.Extends != "" {
    protoInfo = fmt.Sprintf("proto: %s → Object", g.Extends)
}

// New (static + runtime fallback):
if g.Extends != "" {
    protoInfo = fmt.Sprintf("proto: %s → Object", g.Extends)
} else if m.rtSession != nil {
    val := m.rtSession.GlobalValue(g.Name)
    if val != nil && !goja.IsUndefined(val) {
        if obj, ok := val.(*goja.Object); ok {
            chain := runtime.WalkPrototypeChain(obj, m.rtSession.VM)
            if len(chain) > 0 {
                var names []string
                for _, level := range chain {
                    names = append(names, level.Name)
                }
                protoInfo = "proto: " + strings.Join(names, " → ")
            }
        }
    }
}
```

**Risk:** Low — `WalkPrototypeChain` is tested. Might need to cache to avoid re-walking on every render. **Test:** Navigate to `myCircle` → footer shows `proto: Circle → Shape → Object`.

### Step 5: Fix Bug 3 — Enter on value globals opens runtime inspect

**File:** `cmd/smalltalk-inspector/app/update.go`

Modify the Select handler in `handleGlobalsKey()`:

```go
if key.Matches(msg, m.keyMap.Select) {
    if len(m.globals) == 0 || m.globalIdx >= len(m.globals) {
        return m, nil
    }
    selected := m.globals[m.globalIdx]

    // For value-type globals, trigger runtime inspection
    if selected.Kind != jsparse.BindingClass && selected.Kind != jsparse.BindingFunction {
        if m.rtSession != nil {
            result := m.rtSession.EvalWithCapture(selected.Name)
            return m, func() tea.Msg {
                return MsgEvalResult{Result: result}
            }
        }
        return m, nil
    }

    // For class/function, move to members pane
    if len(m.members) > 0 {
        m.focus = FocusMembers
        m.updateMode()
    }
    return m, nil
}
```

**Risk:** Low. **Test:** Navigate to `myCircle` → Enter → inspect view shows `{color, visible, radius}` with `[[Proto]]: Circle.prototype`.

### Step 6: Fix Bug 2 — Refresh globals after REPL eval

**File:** `cmd/smalltalk-inspector/app/model.go` + `update.go`

This is the most nuanced fix because of the `const`/`let` vs `var` semantics.

**Strategy: Two-source merge**

Add a new method `refreshRuntimeGlobals()` that merges runtime-discovered names into the globals list:

```go
func (m *Model) refreshRuntimeGlobals() {
    if m.rtSession == nil {
        return
    }

    // Build a set of already-known names
    known := make(map[string]bool)
    for _, g := range m.globals {
        known[g.Name] = true
    }

    // 1. Scan GlobalObject for new var/function names
    global := m.rtSession.VM.GlobalObject()
    for _, key := range global.Keys() {
        if known[key] {
            continue
        }
        // Skip goja builtins (Object, Array, Math, etc.)
        if isBuiltinGlobal(key) {
            continue
        }
        val := global.Get(key)
        kind := jsparse.BindingVar
        if _, ok := goja.AssertFunction(val); ok {
            kind = jsparse.BindingFunction
        }
        m.globals = append(m.globals, GlobalItem{
            Name: key,
            Kind: kind,
        })
        known[key] = true
    }

    // 2. Check tracked REPL names (const/let/class from REPL input)
    for _, name := range m.replDefinedNames {
        if known[name] {
            continue
        }
        m.globals = append(m.globals, GlobalItem{
            Name: name,
            Kind: jsparse.BindingConst, // approximate
        })
        known[name] = true
    }

    // Re-sort
    m.sortGlobals()
}
```

Add `replDefinedNames []string` to Model. In `handleReplKey()`, before eval, extract declared names:

```go
func extractDeclaredNames(expr string) []string {
    // Simple regex/split for: var x, let x, const x, class X, function X
    // Not full parsing — just enough for REPL one-liners
}
```

Add `isBuiltinGlobal(name string) bool` helper that filters out goja builtins (`Object`, `Array`, `Math`, `JSON`, `String`, `Number`, `Boolean`, `RegExp`, `Date`, `Error`, `Symbol`, `Map`, `Set`, `WeakMap`, `WeakSet`, `Promise`, `Proxy`, `Reflect`, `parseInt`, `parseFloat`, `isNaN`, `isFinite`, `undefined`, `NaN`, `Infinity`, `eval`, `encodeURI`, `encodeURIComponent`, `decodeURI`, `decodeURIComponent`, `console`, `globalThis`, `require`).

In `MsgEvalResult` success handler, add: `m.refreshRuntimeGlobals()`.

**Risk:** Medium — builtin filter list needs to be comprehensive enough. REPL name extraction regex can miss complex patterns but covers the common case. **Test:** Tab to REPL → `var z = 99` → globals list updates to include `z`. Also `const w = {x:1}` → `w` appears.

## Execution Order Rationale

1. **Bug 6 first** — trivial one-liner, prevents stale state from confusing subsequent testing
2. **Bug 4 second** — reorder init, prerequisite for Bug 1 fix to work on load
3. **Bug 1 third** — core fix, highest user impact, makes value globals useful
4. **Bug 5 fourth** — view improvement that complements Bug 1 (proto chain for value globals)
5. **Bug 3 fifth** — interaction improvement that now works naturally (Bug 1 populated members, but Enter-to-inspect is still useful for the deep drill-down view)
6. **Bug 2 last** — most complex, requires the name extraction + merge + filtering logic

Steps 1–2 are trivial setup. Steps 3–5 are the core fixes. Step 6 is a standalone enhancement.

## Testing Plan

After each step, build + lint + run in tmux with `testdata/inspector-test.js`:

```bash
go build ./cmd/smalltalk-inspector/... && golangci-lint run ./cmd/smalltalk-inspector/...
tmux new-session -d -s sinspect -x 130 -y 40 \
  "go run ./cmd/smalltalk-inspector ./testdata/inspector-test.js; read"
```

### Test Matrix

| Test | After Step | Expected |
|------|-----------|----------|
| Navigate to `settings` → members pane | 3 | Shows `theme`, `fontSize`, `lineNumbers` |
| Navigate to `myCircle` → members pane | 3 | Shows `color`, `visible`, `radius` |
| Navigate to `VERSION` → members pane | 3 | Shows `(value) : "1.0.0"` |
| Navigate to `shapes` → members pane | 3 | Shows `0`, `1`, `length` |
| `myCircle` footer shows proto chain | 4 | `proto: Circle → Shape → Object` |
| Enter on `myCircle` → inspect view | 5 | Shows `{color, visible, radius}` with `[[Proto]]` |
| Enter on `VERSION` → inspect view | 5 | Shows `→ "1.0.0"` (primitive, no inspect pane) |
| REPL `var z = 99` → globals list | 6 | 13 bindings, `z` visible as `● z` |
| REPL `const w = {x:1}` → globals list | 6 | 14 bindings, `w` visible as `● w` |
| Navigate to `w` → members pane | 6+3 | Shows `x : 1  number` |
| Eval `settings`, drill `[[Proto]]`, eval `myCircle` → Esc | 1 | Clean exit, no stale proto frames |

## Files Changed Summary

| File | Steps | Changes |
|------|-------|---------|
| `cmd/smalltalk-inspector/app/update.go` | 1,2,5,6 | navStack clear, init reorder, Enter handler, eval refresh |
| `cmd/smalltalk-inspector/app/model.go` | 3,6 | `buildValueMembers()`, `refreshRuntimeGlobals()`, `MemberItem.RuntimeDerived` |
| `cmd/smalltalk-inspector/app/view.go` | 4 | Runtime proto chain in members footer |
| `pkg/inspector/runtime/session.go` | — | No changes needed (GlobalValue already exists) |
| `pkg/inspector/runtime/introspect.go` | — | No changes needed (InspectObject already exists) |
