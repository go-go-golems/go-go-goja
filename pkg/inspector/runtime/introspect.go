package runtime

import (
	"fmt"

	"github.com/dop251/goja"
)

// PropertyInfo describes a single property of an object.
type PropertyInfo struct {
	Name     string
	Value    goja.Value
	Kind     string // "function", "string", "number", "boolean", "object", "undefined", "null", "symbol"
	Preview  string // short value preview
	IsSymbol bool   // true if this is a symbol-keyed property
}

// PrototypeLevel represents one level of the prototype chain.
type PrototypeLevel struct {
	Name       string // constructor name or "<anonymous>"
	Properties []PropertyInfo
}

// InspectObject returns all own properties of an object, grouped by string and symbol keys.
func InspectObject(obj *goja.Object, vm *goja.Runtime) []PropertyInfo {
	if obj == nil {
		return nil
	}

	var props []PropertyInfo

	// String-keyed properties
	for _, key := range obj.GetOwnPropertyNames() {
		val := obj.Get(key)
		props = append(props, PropertyInfo{
			Name:    key,
			Value:   val,
			Kind:    valueKind(val, vm),
			Preview: ValuePreview(val, vm, 30),
		})
	}

	// Symbol-keyed properties
	for _, sym := range obj.Symbols() {
		val := obj.GetSymbol(sym)
		props = append(props, PropertyInfo{
			Name:     sym.String(),
			Value:    val,
			Kind:     valueKind(val, vm),
			Preview:  ValuePreview(val, vm, 30),
			IsSymbol: true,
		})
	}

	return props
}

// WalkPrototypeChain returns the full prototype chain for an object.
func WalkPrototypeChain(obj *goja.Object, vm *goja.Runtime) []PrototypeLevel {
	if obj == nil {
		return nil
	}

	var chain []PrototypeLevel
	for p := obj.Prototype(); p != nil; p = p.Prototype() {
		name := protoName(p, vm)
		level := PrototypeLevel{
			Name:       name,
			Properties: InspectObject(p, vm),
		}
		chain = append(chain, level)
	}
	return chain
}

// Descriptor holds property descriptor information.
type Descriptor struct {
	Writable     bool
	Enumerable   bool
	Configurable bool
	HasGetter    bool
	HasSetter    bool
}

// GetDescriptor retrieves the property descriptor for a string-keyed property.
func GetDescriptor(obj *goja.Object, vm *goja.Runtime, key string) (*Descriptor, error) {
	script := fmt.Sprintf(
		`(function() { var d = Object.getOwnPropertyDescriptor(this, %q); if (!d) return null; return {w: !!d.writable, e: !!d.enumerable, c: !!d.configurable, g: !!d.get, s: !!d.set}; })`,
		key,
	)
	fn, err := vm.RunString(script)
	if err != nil {
		return nil, err
	}
	callable, ok := goja.AssertFunction(fn)
	if !ok {
		return nil, fmt.Errorf("descriptor helper is not callable")
	}
	result, err := callable(obj)
	if err != nil {
		return nil, err
	}
	if goja.IsNull(result) || goja.IsUndefined(result) {
		return nil, nil
	}
	dObj := result.ToObject(vm)
	return &Descriptor{
		Writable:     dObj.Get("w").ToBoolean(),
		Enumerable:   dObj.Get("e").ToBoolean(),
		Configurable: dObj.Get("c").ToBoolean(),
		HasGetter:    dObj.Get("g").ToBoolean(),
		HasSetter:    dObj.Get("s").ToBoolean(),
	}, nil
}

func valueKind(val goja.Value, vm *goja.Runtime) string {
	if val == nil || goja.IsUndefined(val) {
		return "undefined"
	}
	if goja.IsNull(val) {
		return "null"
	}
	if _, ok := goja.AssertFunction(val); ok {
		return "function"
	}
	exported := val.Export()
	switch exported.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case int64, float64:
		return "number"
	default:
		if _, ok := val.(*goja.Object); ok {
			return "object"
		}
		return "unknown"
	}
}

func protoName(obj *goja.Object, vm *goja.Runtime) string {
	ctor := obj.Get("constructor")
	if ctor == nil || goja.IsUndefined(ctor) {
		return "<anonymous>"
	}
	ctorObj, ok := ctor.(*goja.Object)
	if !ok {
		return "<anonymous>"
	}
	name := ctorObj.Get("name")
	if name == nil || goja.IsUndefined(name) {
		return "<anonymous>"
	}
	s := name.String()
	if s == "" {
		return "<anonymous>"
	}
	return s
}
