package analysis

import (
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// MethodSymbol represents a local binding within a method scope.
type MethodSymbol struct {
	Name       string
	Kind       jsparse.BindingKind
	DefineLine int // 1-based line
	UsageCount int // total usages within method scope
}

// MethodSymbols extracts local symbols for a method/function body.
// It finds bindings defined in scopes that fall within the given source span.
func MethodSymbols(res *jsparse.Resolution, idx *jsparse.Index, startOffset, endOffset int) []MethodSymbol {
	if res == nil || idx == nil {
		return nil
	}

	var symbols []MethodSymbol
	for _, scope := range res.Scopes {
		// Skip global scope and scopes outside the method
		if scope.Kind == jsparse.ScopeGlobal {
			continue
		}
		if scope.Start < startOffset || scope.End > endOffset {
			continue
		}

		for _, b := range scope.Bindings {
			// Count usages within the method span
			usageCount := 0
			for _, refID := range b.AllUsages() {
				node := idx.Nodes[refID]
				if node != nil && node.Start >= startOffset && node.End <= endOffset {
					usageCount++
				}
			}

			declNode := idx.Nodes[b.DeclNodeID]
			defLine := 0
			if declNode != nil {
				defLine = declNode.StartLine
			}

			symbols = append(symbols, MethodSymbol{
				Name:       b.Name,
				Kind:       b.Kind,
				DefineLine: defLine,
				UsageCount: usageCount,
			})
		}
	}

	return symbols
}
