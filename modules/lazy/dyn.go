package lazy

import "github.com/dop251/goja"

// DynMap is a lazily-valued string-keyed object backed by a goja DynamicObject
// instead of an ES6 Proxy. It is the same shape as Map (Keys/Provider/Mutable)
// but uses the DynamicObject interface (Get/Set/Has/Delete/Keys).
//
// Tradeoffs vs Map (Proxy-backed), measured (see GOJA-LAZY-BENCH):
//   - cheaper reads: DynamicObject lands on the 1-allocation floor (~124 ns)
//     vs the Proxy's 2 allocations (~191 ns), because there are no ES6
//     invariant checks and no target object.
//   - cheaper enumeration: Object.keys/for..in call Keys() once and never call
//     Get, so there is no per-key descriptor to construct and no §6 sharp edge.
//   - simpler model: every key is writable/enumerable/configurable; the object
//     is always extensible. Get returning nil delegates to the prototype
//     automatically (no manual fallthrough as in the Proxy Get trap).
//   - limitations: no ES6-only traps (construct, getPrototypeOf, defineProperty
//     interception), no Symbol own properties, no per-key enumerability control.
//     Use Map (Proxy) when an ES6-only trap is genuinely required.
type DynMap struct {
	// Keys are the enumerable own keys surfaced to JS (Object.keys / for..in).
	Keys []string
	// Provider computes a value for a key on demand. Return nil for "absent".
	Provider func(key string) any
	// Mutable, when true, makes Set and Delete capture writes into the cache
	// and key list.
	Mutable bool
}

// dynMapHandler implements goja.DynamicObject. It holds a per-runtime cache so
// repeated reads return identical goja.Value (identity + single ToValue).
type dynMapHandler struct {
	m     *DynMap
	vm    *goja.Runtime
	cache map[string]goja.Value
}

// Get returns the (memoised) value for key, or nil to signal "absent" (goja
// then delegates to the prototype, so inherited properties like toString work).
func (h *dynMapHandler) Get(key string) goja.Value {
	if !containsString(h.m.Keys, key) {
		return nil
	}
	return lazyGetStr(h.vm, h.cache, h.m.Provider, key)
}

func (h *dynMapHandler) Has(key string) bool { return containsString(h.m.Keys, key) }

func (h *dynMapHandler) Keys() []string { return h.m.Keys }

func (h *dynMapHandler) Set(key string, val goja.Value) bool {
	h.cache[key] = val
	if !containsString(h.m.Keys, key) {
		h.m.Keys = append(h.m.Keys, key)
	}
	return true
}

func (h *dynMapHandler) Delete(key string) bool {
	delete(h.cache, key)
	h.m.Keys = removeString(h.m.Keys, key)
	return true
}

// ToValue builds the DynamicObject-backed object for this runtime.
func (m *DynMap) ToValue(vm *goja.Runtime) goja.Value {
	h := &dynMapHandler{m: m, vm: vm, cache: map[string]goja.Value{}}
	// DynamicObject has no Set/Delete toggle; the interface always requires
	// them. When not Mutable, the handlers are still present but JS writes would
	// mutate the cache/keys. To enforce read-only semantics when Mutable is
	// false, wrap with a read-only handler.
	if !m.Mutable {
		return vm.NewDynamicObject(&readOnlyDynMap{h: h})
	}
	return vm.NewDynamicObject(h)
}

// DynArray is a lazily-valued integer-indexed object backed by a goja
// DynamicArray. Element i is produced on demand by Provider(i). It mirrors
// Array but uses the DynamicArray interface (Len/Get/Set/SetLen), so it shares
// DynMap's efficiency advantages and the same lack of ES6 traps.
//
// Note: like Array, indices are surfaced via Get (indexing) and Len (length),
// not as enumerable own properties in the Proxy sense; Object.keys on a
// DynamicArray returns the index strings because the dynamic array reports
// 0..Len-1 as own properties.
type DynArray struct {
	// Len is the reported .length. Elements are produced on demand.
	Len int
	// Provider computes the value at index i. Return nil for absent.
	Provider func(i int) any
}

type dynArrHandler struct {
	a     *DynArray
	vm    *goja.Runtime
	cache map[int]goja.Value
}

func (h *dynArrHandler) Len() int { return h.a.Len }

// Get returns the (memoised) value at idx, or nil for out-of-bounds (absent).
func (h *dynArrHandler) Get(idx int) goja.Value {
	if idx < 0 || idx >= h.a.Len {
		return nil
	}
	return lazyGetInt(h.vm, h.cache, h.a.Provider, idx)
}

func (h *dynArrHandler) Set(idx int, val goja.Value) bool {
	if idx < 0 {
		return false
	}
	h.cache[idx] = val
	return true
}

func (h *dynArrHandler) SetLen(n int) bool { return false } // not supported in v1

// ToValue builds the DynamicArray-backed object for this runtime.
func (a *DynArray) ToValue(vm *goja.Runtime) goja.Value {
	return vm.NewDynamicArray(&dynArrHandler{a: a, vm: vm, cache: map[int]goja.Value{}})
}

// readOnlyDynMap wraps a dynMapHandler to make Set/Delete no-ops (return false),
// enforcing read-only semantics when DynMap.Mutable is false.
type readOnlyDynMap struct {
	h *dynMapHandler
}

func (r *readOnlyDynMap) Get(key string) goja.Value           { return r.h.Get(key) }
func (r *readOnlyDynMap) Has(key string) bool                 { return r.h.Has(key) }
func (r *readOnlyDynMap) Keys() []string                      { return r.h.Keys() }
func (r *readOnlyDynMap) Set(key string, val goja.Value) bool { return false }
func (r *readOnlyDynMap) Delete(key string) bool              { return false }
