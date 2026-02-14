package runtime

import (
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

	// Search top-level function declarations
	for _, stmt := range analysis.Program.Body {
		if fd, ok := stmt.(*ast.FunctionDeclaration); ok {
			if fd.Function != nil && fd.Function.Name != nil {
				if string(fd.Function.Name.Name) == funcName {
					offset := int(fd.Idx0())
					sl, sc := analysis.Index.OffsetToLineCol(offset)
					el, ec := analysis.Index.OffsetToLineCol(int(fd.Idx1()))
					return &FunctionSourceMapping{
						Name:      funcName,
						StartLine: sl,
						StartCol:  sc,
						EndLine:   el,
						EndCol:    ec,
					}
				}
			}
		}

		// Search class methods
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
							offset := int(md.Idx0())
							sl, sc := analysis.Index.OffsetToLineCol(offset)
							el, ec := analysis.Index.OffsetToLineCol(int(md.Idx1()))
							return &FunctionSourceMapping{
								Name:      funcName,
								ClassName: className,
								StartLine: sl,
								StartCol:  sc,
								EndLine:   el,
								EndCol:    ec,
							}
						}
					}
				}
			}
		}
	}

	return nil
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
