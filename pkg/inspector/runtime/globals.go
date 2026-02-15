package runtime

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

var builtinGlobals = map[string]bool{
	"Object": true, "Array": true, "Math": true, "JSON": true,
	"String": true, "Number": true, "Boolean": true, "RegExp": true,
	"Date": true, "Error": true, "TypeError": true, "RangeError": true,
	"ReferenceError": true, "SyntaxError": true, "URIError": true, "EvalError": true,
	"Symbol": true, "Map": true, "Set": true, "WeakMap": true, "WeakSet": true,
	"Promise": true, "Proxy": true, "Reflect": true, "ArrayBuffer": true,
	"DataView": true, "Float32Array": true, "Float64Array": true,
	"Int8Array": true, "Int16Array": true, "Int32Array": true,
	"Uint8Array": true, "Uint16Array": true, "Uint32Array": true, "Uint8ClampedArray": true,
	"parseInt": true, "parseFloat": true, "isNaN": true, "isFinite": true,
	"undefined": true, "NaN": true, "Infinity": true, "eval": true,
	"encodeURI": true, "encodeURIComponent": true, "decodeURI": true, "decodeURIComponent": true,
	"escape": true, "unescape": true,
	"console": true, "globalThis": true, "require": true,
	"Function": true, "Iterator": true, "AggregateError": true,
	"SharedArrayBuffer": true, "Atomics": true, "WeakRef": true, "FinalizationRegistry": true,
}

// IsBuiltinGlobal reports whether a name is a standard JavaScript/goja builtin.
func IsBuiltinGlobal(name string) bool {
	return builtinGlobals[name]
}

// BindingKindFromValue infers a binding kind from a runtime value.
// Runtime-discovered values can only be inferred as function or variable here.
func BindingKindFromValue(val goja.Value) jsparse.BindingKind {
	if _, ok := goja.AssertFunction(val); ok {
		return jsparse.BindingFunction
	}
	return jsparse.BindingVar
}

// RuntimeGlobalKinds returns non-builtin global names mapped to inferred kinds.
func RuntimeGlobalKinds(vm *goja.Runtime) map[string]jsparse.BindingKind {
	if vm == nil {
		return nil
	}
	out := map[string]jsparse.BindingKind{}
	global := vm.GlobalObject()
	for _, key := range global.Keys() {
		if IsBuiltinGlobal(key) {
			continue
		}
		out[key] = BindingKindFromValue(global.Get(key))
	}
	return out
}
