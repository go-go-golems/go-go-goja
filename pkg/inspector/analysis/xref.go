package analysis

import (
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// XRefEntry represents a single cross-reference usage of a binding.
type XRefEntry struct {
	Line    int // 1-based source line
	Col     int // 1-based source column
	NodeID  jsparse.NodeID
	Context string // enclosing function/class name if available
}

// CrossReferences finds all usages of a named binding in the analysis result.
func CrossReferences(res *jsparse.Resolution, idx *jsparse.Index, bindingName string) []XRefEntry {
	if res == nil || idx == nil {
		return nil
	}

	// Find binding in any scope
	var binding *jsparse.BindingRecord
	for _, scope := range res.Scopes {
		if b, ok := scope.Bindings[bindingName]; ok {
			binding = b
			break
		}
	}
	if binding == nil {
		return nil
	}

	return CrossReferencesForBinding(binding, idx)
}

// CrossReferencesForBinding returns xref entries for a specific binding record.
func CrossReferencesForBinding(binding *jsparse.BindingRecord, idx *jsparse.Index) []XRefEntry {
	if binding == nil || idx == nil {
		return nil
	}

	var entries []XRefEntry
	for _, nodeID := range binding.AllUsages() {
		node := idx.Nodes[nodeID]
		if node == nil {
			continue
		}

		// Find enclosing context
		context := findEnclosingContext(idx, nodeID)

		entries = append(entries, XRefEntry{
			Line:    node.StartLine,
			Col:     node.StartCol,
			NodeID:  nodeID,
			Context: context,
		})
	}

	return entries
}

// findEnclosingContext finds the name of the nearest enclosing function or class.
func findEnclosingContext(idx *jsparse.Index, nodeID jsparse.NodeID) string {
	path := idx.AncestorPath(nodeID)
	// Walk from node toward root, find first named function/class
	for i := len(path) - 1; i >= 0; i-- {
		ancestor := idx.Nodes[path[i]]
		if ancestor == nil {
			continue
		}
		//exhaustive:ignore
		switch ancestor.Kind {
		case "FunctionDeclaration", "FunctionLiteral", "MethodDefinition":
			if ancestor.Label != "" {
				return ancestor.Label
			}
		case "ClassDeclaration", "ClassLiteral":
			if ancestor.Label != "" {
				return ancestor.Label
			}
		}
	}
	return "(global)"
}
