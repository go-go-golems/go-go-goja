package main

import "github.com/dop251/goja"

// accessLog records which Go-provider keys JS actually read — proves laziness.
var accessLog []string

func reset() { accessLog = nil }

// keySet returns true if needle is in hay.
func keySet(hay []string, needle string) bool {
	for _, k := range hay {
		if k == needle {
			return true
		}
	}
	return false
}

// lazyGet returns the (memoised) JS value for prop, calling provider once.
func lazyGet(vm *goja.Runtime, cache map[string]goja.Value, provider func(string) any, prop string) goja.Value {
	if v, ok := cache[prop]; ok {
		return v
	}
	accessLog = append(accessLog, prop)
	raw := provider(prop)
	if raw == nil {
		return goja.Undefined()
	}
	v := vm.ToValue(raw)
	cache[prop] = v
	return v
}

// dataProp is a shorthand for a normal enumerable data-property descriptor.
// Used by GetOwnPropertyDescriptor traps so JSON.stringify / Object.assign work.
func dataProp(v goja.Value) goja.PropertyDescriptor {
	return goja.PropertyDescriptor{
		Value:        v,
		Writable:     goja.FLAG_FALSE, // lazily-backed values are read-only from JS
		Configurable: goja.FLAG_TRUE,
		Enumerable:   goja.FLAG_TRUE,
	}
}
