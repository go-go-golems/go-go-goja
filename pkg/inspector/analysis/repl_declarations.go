package analysis

import (
	"sort"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// DeclaredBinding describes a binding declared by a REPL/source snippet.
type DeclaredBinding struct {
	Name string
	Kind jsparse.BindingKind
}

// DeclaredBindingsFromSource parses a snippet and returns top-level declared bindings.
// It is parser-backed (jsparse) and therefore safer than token/word heuristics.
func DeclaredBindingsFromSource(source string) []DeclaredBinding {
	result := jsparse.Analyze("<repl>", source, nil)
	if result == nil || result.Resolution == nil {
		return nil
	}
	root := result.Resolution.Scopes[result.Resolution.RootScopeID]
	if root == nil || len(root.Bindings) == 0 {
		return nil
	}

	out := make([]DeclaredBinding, 0, len(root.Bindings))
	for name, b := range root.Bindings {
		if name == "" || b == nil {
			continue
		}
		out = append(out, DeclaredBinding{Name: name, Kind: b.Kind})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return bindingSortOrder(out[i].Kind) < bindingSortOrder(out[j].Kind)
		}
		return out[i].Name < out[j].Name
	})

	return out
}
