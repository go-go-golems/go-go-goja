package analysis

import (
	"sort"

	"github.com/dop251/goja/ast"
	inspectorcore "github.com/go-go-golems/go-go-goja/pkg/inspector/core"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// GlobalBinding describes one top-level binding for inspector list rendering.
type GlobalBinding struct {
	Name    string
	Kind    jsparse.BindingKind
	Extends string
}

// ParseError returns the parse error for the current analysis result, if any.
func (s *Session) ParseError() error {
	if s == nil || s.Result == nil {
		return nil
	}
	return s.Result.ParseErr
}

// Program exposes the parsed program from the session result.
func (s *Session) Program() *ast.Program {
	if s == nil || s.Result == nil {
		return nil
	}
	return s.Result.Program
}

// Index exposes the node index from the session result.
func (s *Session) Index() *jsparse.Index {
	if s == nil || s.Result == nil {
		return nil
	}
	return s.Result.Index
}

// Globals returns sorted top-level bindings with optional class extends info.
func (s *Session) Globals() []GlobalBinding {
	bindings := s.GlobalBindings()
	if len(bindings) == 0 {
		return nil
	}

	globals := make([]GlobalBinding, 0, len(bindings))
	program := s.Program()
	for name, b := range bindings {
		g := GlobalBinding{
			Name: name,
			Kind: b.Kind,
		}
		if program != nil && b.Kind == jsparse.BindingClass {
			g.Extends = inspectorcore.ClassExtends(program, name)
		}
		globals = append(globals, g)
	}

	sort.Slice(globals, func(i, j int) bool {
		oi := bindingSortOrder(globals[i].Kind)
		oj := bindingSortOrder(globals[j].Kind)
		if oi != oj {
			return oi < oj
		}
		return globals[i].Name < globals[j].Name
	})

	return globals
}

// ClassMembers returns own + inherited class members for className.
func (s *Session) ClassMembers(className string) []inspectorcore.Member {
	if className == "" {
		return nil
	}
	return inspectorcore.BuildClassMembers(s.Program(), className)
}

// FunctionMembers returns function parameter members for funcName.
func (s *Session) FunctionMembers(funcName string) []inspectorcore.Member {
	if funcName == "" {
		return nil
	}
	return inspectorcore.BuildFunctionMembers(s.Program(), funcName)
}

// BindingDeclLine returns the declaration line (1-based) for a global binding.
func (s *Session) BindingDeclLine(name string) (int, bool) {
	if name == "" {
		return 0, false
	}
	root := s.rootScope()
	idx := s.Index()
	if root == nil || idx == nil {
		return 0, false
	}
	b, ok := root.Bindings[name]
	if !ok || b == nil {
		return 0, false
	}
	declNode := idx.Nodes[b.DeclNodeID]
	if declNode == nil || declNode.StartLine <= 0 {
		return 0, false
	}
	return declNode.StartLine, true
}

// MemberDeclLine returns the declaration line (1-based) for memberName in className.
// sourceClass is optional and can be used for inherited members.
func (s *Session) MemberDeclLine(className, sourceClass, memberName string) (int, bool) {
	if className == "" || memberName == "" {
		return 0, false
	}
	program := s.Program()
	idx := s.Index()
	if program == nil || idx == nil {
		return 0, false
	}

	targetClass := className
	if sourceClass != "" {
		targetClass = sourceClass
	}

	for _, stmt := range program.Body {
		cd, ok := stmt.(*ast.ClassDeclaration)
		if !ok || cd.Class == nil || cd.Class.Name == nil {
			continue
		}
		if string(cd.Class.Name.Name) != targetClass {
			continue
		}

		for _, elem := range cd.Class.Body {
			md, ok := elem.(*ast.MethodDefinition)
			if !ok {
				continue
			}
			if methodName(md) != memberName {
				continue
			}
			line, _ := idx.OffsetToLineCol(int(md.Idx0()))
			if line <= 0 {
				return 0, false
			}
			return line, true
		}
		break
	}

	return 0, false
}

func bindingSortOrder(k jsparse.BindingKind) int {
	//exhaustive:ignore
	switch k {
	case jsparse.BindingClass:
		return 0
	case jsparse.BindingFunction:
		return 1
	default:
		return 2
	}
}

func methodName(md *ast.MethodDefinition) string {
	if md == nil {
		return "<unknown>"
	}
	switch k := md.Key.(type) {
	case *ast.Identifier:
		return string(k.Name)
	case *ast.StringLiteral:
		return k.Literal
	}
	return "<computed>"
}
