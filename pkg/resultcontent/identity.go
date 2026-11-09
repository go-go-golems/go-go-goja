package resultcontent

import "reflect"

type pointerIdentity struct {
	t   reflect.Type
	ptr uintptr
}

// IdentityKey returns a key that is safe to use in a map for deduplicating
// providers. Pointer-like providers (pointers, maps, channels, funcs) are
// keyed by type and address; other values are keyed by themselves only when
// the value (not just its type) is comparable, so a struct whose interface
// field holds a slice never panics as a map key. ok is false when the
// provider cannot be deduplicated; callers then treat every occurrence as
// distinct.
func IdentityKey(p Provider) (any, bool) {
	if p == nil {
		return nil, false
	}
	rv := reflect.ValueOf(p)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return pointerIdentity{t: rv.Type(), ptr: rv.Pointer()}, true
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array,
		reflect.Interface, reflect.Slice, reflect.String, reflect.Struct:
		if rv.Comparable() {
			return p, true
		}
		return nil, false
	default:
		return nil, false
	}
}

// containerIdentity identifies a Go map or slice backing array for cycle
// detection. Empty slices and nil maps have no identity.
func containerIdentity(rv reflect.Value) (pointerIdentity, bool) {
	switch rv.Kind() {
	case reflect.Map:
		if rv.IsNil() {
			return pointerIdentity{}, false
		}
		return pointerIdentity{t: rv.Type(), ptr: rv.Pointer()}, true
	case reflect.Slice:
		if rv.Len() == 0 {
			return pointerIdentity{}, false
		}
		return pointerIdentity{t: rv.Type(), ptr: rv.Pointer()}, true
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan,
		reflect.Func, reflect.Interface, reflect.Pointer, reflect.String, reflect.Struct, reflect.UnsafePointer:
		return pointerIdentity{}, false
	default:
		return pointerIdentity{}, false
	}
}

// addressKey is IdentityKey restricted to pointer-like providers.
func addressKey(p Provider) (any, bool) {
	if p == nil {
		return nil, false
	}
	rv := reflect.ValueOf(p)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return pointerIdentity{t: rv.Type(), ptr: rv.Pointer()}, true
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array,
		reflect.Interface, reflect.Slice, reflect.String, reflect.Struct:
		return nil, false
	default:
		return nil, false
	}
}
