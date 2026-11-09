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
// keys, slices, scalars) and returns a copy in which every Provider is
// replaced by replace(n, p), plus the distinct providers in walk order. n is
// the 1-based provider number; the same comparable provider reached twice
// keeps its number. Map keys are visited in sorted order, so two walks of the
// same value number providers identically. Depth and property counts are
// bounded by limits; values beyond them are kept as they are.
//
// MapExported does not touch goja and can run on any goroutine.
func MapExported(value any, limits Limits, replace func(n int, p Provider) any) (any, []Provider) {
	w := &exportedWalker{limits: limits.Normalize(), replace: replace, ids: map[Provider]int{}}
	out := w.walk(value, 0)
	return out, w.providers
}

// ExtractExported replaces each provider in an exported value with its
// numbered placeholder string ("[content N: ...]").
func ExtractExported(value any, limits Limits) (any, []Provider) {
	return MapExported(value, limits, func(n int, p Provider) any { return Placeholder(n, p) })
}

type exportedWalker struct {
	limits    Limits
	replace   func(int, Provider) any
	props     int
	providers []Provider
	ids       map[Provider]int
}

func (w *exportedWalker) number(p Provider) int {
	if reflect.TypeOf(p).Comparable() {
		if n, ok := w.ids[p]; ok {
			return n
		}
	}
	w.providers = append(w.providers, p)
	n := len(w.providers)
	if reflect.TypeOf(p).Comparable() {
		w.ids[p] = n
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
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(v))
		for _, key := range keys {
			out[key] = w.child(v[key], depth)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = w.child(item, depth)
		}
		return out
	default:
		return value
	}
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
