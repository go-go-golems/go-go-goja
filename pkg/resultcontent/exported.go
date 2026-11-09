package resultcontent

import (
	"reflect"
	"sort"
)

// InlinePlaceholder names a provider without a number, for places that are
// not part of a numbered result (console output, binding snapshots):
// "[content: <desc>]" or "[content]".
func InlinePlaceholder(p Provider) string {
	if desc := Describe(p); desc != "" {
		return "[content: " + desc + "]"
	}
	return "[content]"
}

// MapExported walks a Go value as produced by goja's Export (maps with string
// keys and slices or arrays, including typed Go collections such as
// map[string]*Image or []*Image, plus scalars) and returns a copy in which every Provider is
// replaced by replace(n, p), plus the distinct providers in walk order. n is
// the 1-based provider number; the same provider reached twice keeps its
// number when it has an IdentityKey. Typed collections are copied into
// map[string]any / []any. A map or slice already being walked further up
// (a cycle, detected by backing pointer) becomes "[Circular]". Map keys are visited in sorted order, so two walks of the
// same value number providers identically. Depth and property counts are
// bounded by limits; values beyond them are kept as they are.
//
// MapExported does not touch goja and can run on any goroutine.
func MapExported(value any, limits Limits, replace func(n int, p Provider) any) (any, []Provider) {
	w := &exportedWalker{limits: limits.Normalize(), replace: replace, ids: map[any]int{}, inProgress: map[pointerIdentity]struct{}{}}
	out := w.walk(value, 0)
	return out, w.providers
}

// ExtractExported replaces each provider in an exported value with its
// numbered placeholder string ("[content N: ...]").
func ExtractExported(value any, limits Limits) (any, []Provider) {
	return MapExported(value, limits, func(n int, p Provider) any { return Placeholder(n, p) })
}

type exportedWalker struct {
	limits     Limits
	replace    func(int, Provider) any
	props      int
	providers  []Provider
	ids        map[any]int
	inProgress map[pointerIdentity]struct{}
}

func (w *exportedWalker) number(p Provider) int {
	key, dedupable := IdentityKey(p)
	if dedupable {
		if n, ok := w.ids[key]; ok {
			return n
		}
	}
	w.providers = append(w.providers, p)
	n := len(w.providers)
	if dedupable {
		w.ids[key] = n
	}
	return n
}

func (w *exportedWalker) walk(value any, depth int) any {
	if value == nil {
		return nil
	}
	if p, ok := value.(Provider); ok {
		n := w.number(p)
		if w.replace == nil {
			return Placeholder(n, p)
		}
		return w.replace(n, p)
	}
	if depth >= w.limits.MaxDepth {
		return truncatedContainer(value)
	}
	rv := reflect.ValueOf(value)
	switch {
	case rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String:
		return w.cycleGuard(rv, func() any { return w.walkMap(rv, depth) })
	case (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Type().Elem().Kind() != reflect.Uint8:
		return w.cycleGuard(rv, func() any { return w.walkSlice(rv, depth) })
	default:
		return value
	}
}

// cycleGuard returns "[Circular]" when rv (a map or slice) is already being
// walked further up, using its backing pointer as identity.
func (w *exportedWalker) cycleGuard(rv reflect.Value, fn func() any) any {
	id, ok := containerIdentity(rv)
	if !ok {
		return fn()
	}
	if _, cyclic := w.inProgress[id]; cyclic {
		return "[Circular]"
	}
	w.inProgress[id] = struct{}{}
	defer delete(w.inProgress, id)
	return fn()
}

// walkMap copies a string-keyed map (map[string]any or a typed Go map) into
// a map[string]any, visiting keys in sorted order.
func (w *exportedWalker) walkMap(rv reflect.Value, depth int) any {
	keys := make([]string, 0, rv.Len())
	values := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		keys = append(keys, key)
		values[key] = iter.Value().Interface()
	}
	sort.Strings(keys)
	out := make(map[string]any, len(keys))
	for _, key := range keys {
		out[key] = w.child(values[key], depth)
	}
	return out
}

// walkSlice copies a slice or array ([]any or typed) into a []any.
func (w *exportedWalker) walkSlice(rv reflect.Value, depth int) any {
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = w.child(rv.Index(i).Interface(), depth)
	}
	return out
}

func (w *exportedWalker) child(value any, depth int) any {
	w.props++
	if w.props > w.limits.MaxProperties {
		if p, ok := value.(Provider); ok {
			return w.walk(p, depth+1)
		}
		return truncatedContainer(value)
	}
	return w.walk(value, depth+1)
}

func truncatedContainer(value any) any {
	switch value.(type) {
	case map[string]any, []any:
		return "…"
	}
	rv := reflect.ValueOf(value)
	kind := rv.Kind()
	if kind == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
		return value
	}
	if kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array || kind == reflect.Struct || kind == reflect.Pointer || kind == reflect.Interface {
		return "…"
	}
	return value
}
