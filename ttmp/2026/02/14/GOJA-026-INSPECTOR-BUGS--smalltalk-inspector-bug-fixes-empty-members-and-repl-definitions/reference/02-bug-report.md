---
title: "Bug Report"
doc_type: reference
status: active
intent: long-term
topics:
  - smalltalk-inspector
  - bugs
  - debugging
created: 2026-02-14T12:50:00-05:00
---

# Smalltalk Inspector — Bug Report

## Test File

`testdata/inspector-test.js` — contains classes (Shape, Circle, Rectangle with inheritance), functions (createShape, totalArea, main), constants (VERSION, MAX_ITEMS), object literals (settings), and instantiated objects (myCircle, myRect, shapes).

## Bug 1: Non-Class Globals Show "(no members)" in Members Pane

### Severity

Medium — degrades usability for value-type globals.

### Steps to Reproduce

1. `go run ./cmd/smalltalk-inspector ./testdata/inspector-test.js`
2. Navigate globals pane with arrow keys to `settings` (object literal), `myCircle` (class instance), `myRect`, `shapes` (array), `VERSION` (string), or `MAX_ITEMS` (number)
3. Observe members pane shows "(no members)" for all value-type globals

### Expected Behavior

- `settings` should show `theme`, `fontSize`, `lineNumbers` as members
- `myCircle` should show `color`, `visible`, `radius` plus Circle/Shape methods
- `shapes` should show `0`, `1`, `length` (array properties)
- `VERSION` should show the string value "1.0.0"
- `MAX_ITEMS` should show the number value 100

### Actual Behavior

All value globals (`●` icon) show "(no members)". Only class declarations (`C`) and function declarations (`ƒ`) populate the members pane.

### Root Cause

`buildMembers()` in `model.go` only handles two cases:

```go
switch selected.Kind {
case jsparse.BindingClass:
    m.buildClassMembers(selected.Name)
case jsparse.BindingFunction:
    m.buildFunctionMembers(selected.Name)
}
```

There is no case for value-type bindings (`jsparse.BindingConst`, `jsparse.BindingLet`, `jsparse.BindingVar`). The method uses only static AST analysis (`jsparse`), not the runtime session. The runtime session is available (`m.rtSession`) and can introspect any global value via `m.rtSession.GlobalValue(name)`.

### Suggested Fix

Add a default case to `buildMembers()` that uses the runtime session to introspect the value:

```go
default:
    m.buildValueMembers(selected.Name)
```

Where `buildValueMembers` does:
1. `val := m.rtSession.GlobalValue(name)` — get the runtime value
2. If it's an object: `runtime.InspectObject(obj, m.rtSession.VM)` — show properties
3. If it's a primitive: show a single entry with the value preview
4. Convert the `PropertyInfo` list to `MemberItem` entries

### Files Involved

- `cmd/smalltalk-inspector/app/model.go` — `buildMembers()`, lines 194-213
- `pkg/inspector/runtime/session.go` — `GlobalValue()` already exists
- `pkg/inspector/runtime/introspect.go` — `InspectObject()` already exists

---

## Bug 2: REPL Variable Definitions Don't Refresh Globals List

### Severity

Medium — runtime state diverges from displayed state.

### Steps to Reproduce

1. `go run ./cmd/smalltalk-inspector ./testdata/inspector-test.js`
2. Tab to REPL
3. Type `var myVar = 42` → Enter
4. Result shows `→ undefined` (correct for `var` declaration)
5. Globals pane still shows "12 bindings" — `myVar` is not listed
6. Type `myVar` → Enter → shows `→ 42` (value IS in runtime)

### Expected Behavior

After evaluating `var myVar = 42`, the globals list should refresh to show 13 bindings including `myVar`.

### Actual Behavior

The globals list is never refreshed after REPL evaluation. It only changes on `:load`.

### Root Cause

The eval result handler in `update.go` (`case MsgEvalResult:`) updates the inspect/error state but never calls `m.buildGlobals()`. Even if it did, `buildGlobals()` only reads from `m.analysis.Resolution` (the static AST scope), not from the runtime's actual global object.

Two issues compound:
1. `buildGlobals()` is purely static — it reads `jsparse.AnalysisResult`, not `goja.Runtime`
2. No call to refresh globals after REPL eval

### Suggested Fix

After successful REPL eval, enumerate the runtime's global object and merge new bindings:

```go
// In MsgEvalResult handler, after successful eval:
m.refreshRuntimeGlobals()
```

Where `refreshRuntimeGlobals()`:
1. Gets the runtime global object: `m.rtSession.VM.GlobalObject()`
2. Enumerates its keys
3. For any key not already in `m.globals`, adds it as a `●` value binding
4. Optionally sorts the combined list

Alternatively, maintain a separate "REPL-defined globals" list that appends to the static list.

### Files Involved

- `cmd/smalltalk-inspector/app/update.go` — `MsgEvalResult` handler, lines 73-104
- `cmd/smalltalk-inspector/app/model.go` — `buildGlobals()`, lines 154-191

---

## Bug 3: Enter on Value Globals Does Nothing

### Severity

Low-Medium — missed interaction that would make the inspector more discoverable.

### Steps to Reproduce

1. Navigate to `myCircle` in globals pane
2. Press Enter
3. Nothing happens — no inspect view, no navigation

### Expected Behavior

Pressing Enter on a value global should trigger runtime inspection, equivalent to typing the name in the REPL. For `myCircle` this would show `{color, visible, radius}` with `[[Proto]]: Circle.prototype`.

### Actual Behavior

The Enter handler in `handleGlobalsKey()` only moves focus to the members pane if there are members:

```go
if key.Matches(msg, m.keyMap.Select) {
    if len(m.members) > 0 {
        m.focus = FocusMembers
        m.updateMode()
    }
    return m, nil
}
```

Since value globals have no members (Bug 1), Enter does nothing.

### Suggested Fix

When Enter is pressed on a global and either there are no members or the binding is a value type, trigger runtime inspection:

```go
if key.Matches(msg, m.keyMap.Select) {
    selected := m.globals[m.globalIdx]
    if selected.Kind != jsparse.BindingClass && selected.Kind != jsparse.BindingFunction {
        // Trigger runtime inspection
        result := m.rtSession.EvalWithCapture(selected.Name)
        return m, func() tea.Msg { return MsgEvalResult{Result: result} }
    }
    if len(m.members) > 0 {
        m.focus = FocusMembers
        m.updateMode()
    }
    return m, nil
}
```

### Files Involved

- `cmd/smalltalk-inspector/app/update.go` — `handleGlobalsKey()`, Select handler

---

## Summary

| # | Bug | Severity | Root Cause | Fix Complexity |
|---|-----|----------|-----------|---------------|
| 1 | Non-class globals show no members | Medium | `buildMembers()` only handles class/function via static AST | Small — add runtime introspection fallback |
| 2 | REPL definitions don't refresh globals | Medium | `buildGlobals()` is static-only, no refresh after eval | Medium — need runtime global enumeration + merge |
| 3 | Enter on value globals does nothing | Low-Medium | Enter handler requires non-empty members list | Small — add runtime eval fallback for value bindings |

### Reproduction Command

```bash
go run ./cmd/smalltalk-inspector ./testdata/inspector-test.js
```

### Test Sequence

1. Arrow to `settings` → members pane should show properties (Bug 1)
2. Arrow to `myCircle` → members pane should show instance properties (Bug 1)
3. Enter on `myCircle` → should open inspect view (Bug 3)
4. Tab to REPL → `var x = {a:1}` → Enter → globals should show `x` (Bug 2)
5. Type `x` → Enter → inspect view shows `{a}` (this works — REPL eval is fine)
