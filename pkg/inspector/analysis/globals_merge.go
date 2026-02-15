package analysis

import (
	"sort"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// SortGlobals sorts globals by kind-group then name.
func SortGlobals(globals []GlobalBinding) {
	sort.Slice(globals, func(i, j int) bool {
		oi := bindingSortOrder(globals[i].Kind)
		oj := bindingSortOrder(globals[j].Kind)
		if oi != oj {
			return oi < oj
		}
		return globals[i].Name < globals[j].Name
	})
}

// MergeGlobals appends runtime/discovered bindings to an existing global list.
// It keeps existing entries, de-duplicates by name, and returns a sorted list.
func MergeGlobals(
	existing []GlobalBinding,
	runtimeKinds map[string]jsparse.BindingKind,
	declared []DeclaredBinding,
	hasRuntimeValue func(name string) bool,
) []GlobalBinding {
	out := make([]GlobalBinding, 0, len(existing)+len(runtimeKinds)+len(declared))
	out = append(out, existing...)

	known := make(map[string]bool, len(out))
	for _, g := range out {
		known[g.Name] = true
	}

	for name, kind := range runtimeKinds {
		if known[name] {
			continue
		}
		out = append(out, GlobalBinding{
			Name: name,
			Kind: kind,
		})
		known[name] = true
	}

	for _, decl := range declared {
		if decl.Name == "" || known[decl.Name] {
			continue
		}
		if hasRuntimeValue != nil && !hasRuntimeValue(decl.Name) {
			continue
		}
		out = append(out, GlobalBinding{
			Name: decl.Name,
			Kind: decl.Kind,
		})
		known[decl.Name] = true
	}

	SortGlobals(out)
	return out
}
