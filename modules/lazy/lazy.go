// Package lazy exposes lazily-materialised Go data structures to JavaScript
// through goja Proxies. A lazy value looks like an ordinary JS object/array but
// produces each value on demand from a Go provider, only when JS actually reads
// that key or index.
//
// The central design rule (see the GOJA-LAZY-DS design doc, §6): the
// GetOwnPropertyDescriptor trap returns a *flag-only* descriptor (no Value),
// so Object.keys() / for...in enumeration stays O(keys) and never calls the
// provider. Real values are produced only on the Get trap and are memoised.
package lazy

import (
	"github.com/dop251/goja"
)

// flagOnlyDescriptor returns a data-property descriptor carrying only flags.
// It intentionally leaves Value and Writable unset so that:
//   - enumeration (Object.keys / for..in) treats the key as enumerable without
//     materialising a value, and
//   - the descriptor does NOT trip goja's "Writable+Configurable+Enumerable all
//     true" fast path (which would report an undefined value and make the key
//     vanish from Object.keys — see design doc §6.3).
//
// Callers that need the real value must read it via the object ([[Get]]), not
// via Object.getOwnPropertyDescriptor(k).value. This is a documented tradeoff:
// Object.assign / spread / destructuring all use [[Get]] and therefore receive
// the lazy value; only explicit getOwnPropertyDescriptor(k).value is undefined.
func flagOnlyDescriptor(enumerable bool) goja.PropertyDescriptor {
	return goja.PropertyDescriptor{
		Enumerable:   goja.ToFlag(enumerable),
		Configurable: goja.FLAG_TRUE,
	}
}

// absentDescriptor signals "no own property" for keys the lazy structure does
// not own. goja turns an empty PropertyDescriptor into undefined, and the proxy
// then defers to the (empty) target — i.e. the key is not own.
func absentDescriptor() goja.PropertyDescriptor {
	return goja.PropertyDescriptor{}
}

// toValueAny converts a provider result to a goja.Value. If the provider
// already returns a goja.Value (e.g. a JS-backed getter called from Go), it is
// returned as-is to preserve identity and avoid double-wrapping.
func toValueAny(vm *goja.Runtime, v any) goja.Value {
	if v == nil {
		return goja.Undefined()
	}
	if gv, ok := v.(goja.Value); ok {
		return gv
	}
	return vm.ToValue(v)
}

// lazyGetStr returns the (memoised) JS value for key, calling provider once.
// The cache stores goja.Value so repeated reads return identical objects
// (device.k === device.k) and ToValue runs at most once per key.
func lazyGetStr(vm *goja.Runtime, cache map[string]goja.Value, provider func(string) any, key string) goja.Value {
	if v, ok := cache[key]; ok {
		return v
	}
	v := toValueAny(vm, provider(key))
	cache[key] = v
	return v
}

// lazyGetInt is the integer-indexed analogue of lazyGetStr.
func lazyGetInt(vm *goja.Runtime, cache map[int]goja.Value, provider func(int) any, idx int) goja.Value {
	if v, ok := cache[idx]; ok {
		return v
	}
	v := toValueAny(vm, provider(idx))
	cache[idx] = v
	return v
}

func containsString(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func removeString(s []string, v string) []string {
	out := s[:0]
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// strArray builds a JS array of strings from a Go slice.
func strArray(vm *goja.Runtime, keys []string) *goja.Object {
	items := make([]any, len(keys))
	for i, k := range keys {
		items[i] = k
	}
	return vm.NewArray(items...)
}

// Map is a lazily-valued string-keyed object whose keys are known up front.
// Values are produced on demand by Provider the first time JS reads a key.
//
// Configure once, then call ToValue per *goja.Runtime (each call builds a fresh
// proxy with its own cache; a goja.Value is bound to a single runtime).
type Map struct {
	// Keys are the enumerable own keys surfaced to JS (Object.keys / for..in).
	Keys []string
	// Provider computes a value for a key on demand. Return nil for "absent".
	Provider func(key string) any
	// Mutable, when true, installs Set and deleteProperty traps so JS writes
	// (obj.k = v, delete obj.k) are captured into the cache and key list.
	Mutable bool
}

// Build constructs the lazy Proxy for this runtime. The returned goja.Proxy is a
// valueContainer: pass it to vm.Set / return it from Go functions, or call
// ToValue for a goja.Value. Revoke it with .Revoke() to tear the proxy down.
func (m *Map) Build(vm *goja.Runtime) goja.Proxy {
	cache := map[string]goja.Value{}
	target := vm.NewObject()
	cfg := &goja.ProxyTrapConfig{
		Get: func(t *goja.Object, p string, receiver goja.Value) goja.Value {
			if containsString(m.Keys, p) {
				return lazyGetStr(vm, cache, m.Provider, p)
			}
			// Inherited properties (toString, valueOf, ...) fall through.
			return t.Get(p)
		},
		Has: func(t *goja.Object, p string) bool {
			if containsString(m.Keys, p) {
				return true
			}
			v := t.Get(p)
			return v != nil && !goja.IsUndefined(v) && !goja.IsNull(v)
		},
		GetOwnPropertyDescriptor: func(t *goja.Object, p string) goja.PropertyDescriptor {
			if !containsString(m.Keys, p) {
				return absentDescriptor()
			}
			return flagOnlyDescriptor(true)
		},
		OwnKeys: func(t *goja.Object) *goja.Object {
			return strArray(vm, m.Keys)
		},
	}
	if m.Mutable {
		cfg.Set = func(t *goja.Object, p string, value goja.Value, receiver goja.Value) bool {
			cache[p] = value
			if !containsString(m.Keys, p) {
				m.Keys = append(m.Keys, p)
			}
			return true
		}
		cfg.DeleteProperty = func(t *goja.Object, p string) bool {
			delete(cache, p)
			m.Keys = removeString(m.Keys, p)
			return true
		}
	}
	return vm.NewProxy(target, cfg)
}

// ToValue is a convenience wrapper returning the proxy as a goja.Value.
func (m *Map) ToValue(vm *goja.Runtime) goja.Value {
	return vm.ToValue(m.Build(vm))
}

// Array is a lazily-valued integer-indexed object with a reported length.
// Element i is produced on demand by Provider(i) when JS indexes it.
//
// Note: indices are surfaced via [[Get]]/[[Has]] (indexing and `in`), not as
// enumerable own properties. Object.keys() therefore returns [] — use indexing
// or a for..of loop driven by .length. This keeps even "infinite" arrays cheap.
type Array struct {
	// Len is the reported .length. May be large; elements are not materialised.
	Len int
	// Provider computes the value at index i on demand. Return nil for absent.
	Provider func(i int) any
}

// Build constructs the lazy Array proxy for this runtime.
func (a *Array) Build(vm *goja.Runtime) goja.Proxy {
	cache := map[int]goja.Value{}
	target := vm.NewArray() // Array target -> Array.isArray(proxy) === true
	cfg := &goja.ProxyTrapConfig{
		GetIdx: func(t *goja.Object, i int, receiver goja.Value) goja.Value {
			if i < 0 || i >= a.Len {
				return goja.Undefined()
			}
			return lazyGetInt(vm, cache, a.Provider, i)
		},
		HasIdx: func(t *goja.Object, i int) bool { return i >= 0 && i < a.Len },
		Get: func(t *goja.Object, p string, receiver goja.Value) goja.Value {
			if p == "length" {
				return vm.ToValue(a.Len)
			}
			return t.Get(p)
		},
	}
	return vm.NewProxy(target, cfg)
}

// ToValue is a convenience wrapper returning the proxy as a goja.Value.
func (a *Array) ToValue(vm *goja.Runtime) goja.Value {
	return vm.ToValue(a.Build(vm))
}

// Record is a lazily-valued object whose key SET is itself produced on demand
// (e.g. environment variables, a live registry, a paged KV store). Keys is a
// cheap listing query; Provider fetches a single value.
//
// If Has is nil, membership is derived by scanning Keys() on each Has/Get call.
// Provide Has for O(1) membership when Keys() is expensive.
type Record struct {
	// Keys returns the enumerable own keys (called on Object.keys / for..in).
	// It must be cheap; it must NOT materialise values.
	Keys func() []string
	// Has reports whether key exists. Optional; defaults to scanning Keys().
	Has func(key string) bool
	// Provider computes a value for a key on demand. Return nil for absent.
	Provider func(key string) any
	// Mutable, when true, installs Set/deleteProperty traps. Added keys are
	// kept in an internal extra set and unioned with Keys() for enumeration.
	Mutable bool

	// extra tracks keys added via Set when Mutable. Per-runtime in practice
	// (a Record is configured once and ToValue'd per runtime); not guarded for
	// cross-runtime sharing — don't share a Mutable Record across runtimes.
	extra []string
}

func (r *Record) has(key string) bool {
	// Keys added via Set (Mutable) are always considered present, even if the
	// backing Has/Keys functions don't know about them yet.
	if containsString(r.extra, key) {
		return true
	}
	if r.Has != nil {
		return r.Has(key)
	}
	return containsString(r.Keys(), key)
}

func (r *Record) ownKeys() []string {
	base := r.Keys()
	if len(r.extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(r.extra))
	out := make([]string, 0, len(base)+len(r.extra))
	for _, k := range base {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	for _, k := range r.extra {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	return out
}

// Build constructs the lazy Record proxy for this runtime.
func (r *Record) Build(vm *goja.Runtime) goja.Proxy {
	cache := map[string]goja.Value{}
	target := vm.NewObject()
	cfg := &goja.ProxyTrapConfig{
		Get: func(t *goja.Object, p string, receiver goja.Value) goja.Value {
			if r.has(p) {
				return lazyGetStr(vm, cache, r.Provider, p)
			}
			return t.Get(p)
		},
		Has: func(t *goja.Object, p string) bool {
			if r.has(p) {
				return true
			}
			v := t.Get(p)
			return v != nil && !goja.IsUndefined(v) && !goja.IsNull(v)
		},
		GetOwnPropertyDescriptor: func(t *goja.Object, p string) goja.PropertyDescriptor {
			if !r.has(p) {
				return absentDescriptor()
			}
			return flagOnlyDescriptor(true)
		},
		OwnKeys: func(t *goja.Object) *goja.Object {
			return strArray(vm, r.ownKeys())
		},
	}
	if r.Mutable {
		cfg.Set = func(t *goja.Object, p string, value goja.Value, receiver goja.Value) bool {
			cache[p] = value
			if !containsString(r.extra, p) && !containsString(r.Keys(), p) {
				r.extra = append(r.extra, p)
			}
			return true
		}
		cfg.DeleteProperty = func(t *goja.Object, p string) bool {
			delete(cache, p)
			r.extra = removeString(r.extra, p)
			return true
		}
	}
	return vm.NewProxy(target, cfg)
}

// ToValue is a convenience wrapper returning the proxy as a goja.Value.
func (r *Record) ToValue(vm *goja.Runtime) goja.Value {
	return vm.ToValue(r.Build(vm))
}
