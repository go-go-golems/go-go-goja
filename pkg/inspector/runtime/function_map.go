package runtime

import (
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// FunctionSourceMapping maps a runtime function/method back to its AST source span.
type FunctionSourceMapping struct {
	Name      string
	ClassName string // empty for standalone functions
	StartLine int    // 1-based
	StartCol  int    // 1-based
	EndLine   int
	EndCol    int
	NodeID    jsparse.NodeID
}

// MapFunctionToSource attempts to find the source location of a runtime function
// by matching its name against AST declarations.
func MapFunctionToSource(val goja.Value, vm *goja.Runtime, analysis *jsparse.AnalysisResult) *FunctionSourceMapping {
	if analysis == nil || analysis.Program == nil || analysis.Index == nil {
		return nil
	}

	// Get function name from runtime
	_, ok := goja.AssertFunction(val)
	if !ok {
		return nil
	}
	obj := val.ToObject(vm)
	nameVal := obj.Get("name")
	if nameVal == nil || goja.IsUndefined(nameVal) {
		return nil
	}
	funcName := nameVal.String()
	if funcName == "" {
		return nil
	}

	runtimeSource := normalizeFunctionSource(val.String())
	sourceText := analysis.Index.Source()
	candidates := make([]functionSourceCandidate, 0, 4)

	// Search top-level function declarations.
	for _, stmt := range analysis.Program.Body {
		if fd, ok := stmt.(*ast.FunctionDeclaration); ok {
			if fd.Function != nil && fd.Function.Name != nil {
				if string(fd.Function.Name.Name) == funcName {
					candidates = append(candidates, buildFunctionSourceCandidate(
						analysis,
						funcName,
						"",
						int(fd.Idx0()),
						int(fd.Idx1()),
						sourceText,
					))
				}
			}
		}

		// Search class methods.
		if cd, ok := stmt.(*ast.ClassDeclaration); ok {
			if cd.Class != nil {
				className := ""
				if cd.Class.Name != nil {
					className = string(cd.Class.Name.Name)
				}
				for _, elem := range cd.Class.Body {
					if md, ok := elem.(*ast.MethodDefinition); ok {
						name := methodKeyName(md)
						if name == funcName {
							candidates = append(candidates, buildFunctionSourceCandidate(
								analysis,
								funcName,
								className,
								int(md.Idx0()),
								int(md.Idx1()),
								sourceText,
							))
						}
					}
				}
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		m := candidates[0].mapping
		return &m
	}

	if runtimeSource != "" {
		for _, c := range candidates {
			if c.normalizedSource == runtimeSource {
				m := c.mapping
				return &m
			}
		}
		for _, c := range candidates {
			if strings.Contains(c.normalizedSource, runtimeSource) || strings.Contains(runtimeSource, c.normalizedSource) {
				m := c.mapping
				return &m
			}
		}
	}

	// Deterministic fallback: earliest source offset.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].startOffset < candidates[j].startOffset
	})
	m := candidates[0].mapping
	return &m
}

func methodKeyName(md *ast.MethodDefinition) string {
	if md == nil {
		return ""
	}
	switch k := md.Key.(type) {
	case *ast.Identifier:
		return string(k.Name)
	case *ast.StringLiteral:
		return k.Literal
	}
	return ""
}

type functionSourceCandidate struct {
	mapping          FunctionSourceMapping
	normalizedSource string
	startOffset      int
}

func buildFunctionSourceCandidate(
	analysis *jsparse.AnalysisResult,
	funcName string,
	className string,
	startOffset int,
	endOffset int,
	sourceText string,
) functionSourceCandidate {
	startLine, startCol := analysis.Index.OffsetToLineCol(startOffset)
	endLine, endCol := analysis.Index.OffsetToLineCol(endOffset)

	nodeID := jsparse.NodeID(-1)
	if node := analysis.Index.NodeAtOffset(startOffset); node != nil {
		nodeID = node.ID
	}

	return functionSourceCandidate{
		mapping: FunctionSourceMapping{
			Name:      funcName,
			ClassName: className,
			StartLine: startLine,
			StartCol:  startCol,
			EndLine:   endLine,
			EndCol:    endCol,
			NodeID:    nodeID,
		},
		normalizedSource: normalizedSourceSnippet(sourceText, startOffset, endOffset),
		startOffset:      startOffset,
	}
}

func normalizedSourceSnippet(source string, startOffset, endOffset int) string {
	if source == "" {
		return ""
	}
	start := startOffset - 1 // 1-based -> 0-based
	end := endOffset - 1

	if start < 0 {
		start = 0
	}
	if start > len(source) {
		start = len(source)
	}
	if end < start {
		end = start
	}
	if end > len(source) {
		end = len(source)
	}
	return normalizeFunctionSource(source[start:end])
}

func normalizeFunctionSource(src string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return ""
	}
	return strings.Join(strings.Fields(src), " ")
}
