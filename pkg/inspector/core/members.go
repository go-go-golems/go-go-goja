// Package core provides UI-independent inspector logic that can be reused
// by Bubble Tea, CLI, or REST frontends.
package core

import (
	"strings"

	"github.com/dop251/goja/ast"
)

// Member describes one member entry returned by AST-based analysis.
type Member struct {
	Name      string
	Kind      string
	Preview   string
	Inherited bool
	Source    string
}

// ClassExtends returns the direct superclass identifier name, if any.
func ClassExtends(program *ast.Program, className string) string {
	if program == nil {
		return ""
	}
	for _, stmt := range program.Body {
		cd, ok := stmt.(*ast.ClassDeclaration)
		if !ok || cd.Class == nil || cd.Class.Name == nil {
			continue
		}
		if string(cd.Class.Name.Name) != className {
			continue
		}
		if cd.Class.SuperClass != nil {
			if ident, ok := cd.Class.SuperClass.(*ast.Identifier); ok {
				return string(ident.Name)
			}
		}
		return ""
	}
	return ""
}

// BuildClassMembers returns own + inherited members for a class declaration.
// Inherited members include methods only (constructor excluded), matching
// existing smalltalk-inspector semantics.
func BuildClassMembers(program *ast.Program, className string) []Member {
	if program == nil {
		return nil
	}

	classes := indexClasses(program)
	root := classes[className]
	if root == nil || root.Class == nil {
		return nil
	}

	var members []Member
	seenNames := map[string]bool{}

	// Own members first.
	for _, elem := range root.Class.Body {
		switch e := elem.(type) {
		case *ast.MethodDefinition:
			name := methodName(e)
			preview := "()"
			if e.Body != nil && e.Body.ParameterList != nil {
				params := paramNames(e.Body.ParameterList)
				preview = "(" + strings.Join(params, ", ") + ")"
			}
			members = append(members, Member{
				Name:    name,
				Kind:    "function",
				Preview: preview,
			})
			seenNames[name] = true
		case *ast.FieldDefinition:
			name := fieldName(e)
			members = append(members, Member{
				Name: name,
				Kind: "value",
			})
			seenNames[name] = true
		}
	}

	visited := map[string]bool{className: true}
	addInheritedMethods(classes, superClassName(root), &members, seenNames, visited)

	return members
}

// BuildFunctionMembers returns function parameter members for a top-level function.
func BuildFunctionMembers(program *ast.Program, funcName string) []Member {
	if program == nil {
		return nil
	}
	for _, stmt := range program.Body {
		fd, ok := stmt.(*ast.FunctionDeclaration)
		if !ok || fd.Function == nil || fd.Function.Name == nil {
			continue
		}
		if string(fd.Function.Name.Name) != funcName {
			continue
		}
		var members []Member
		if fd.Function.ParameterList != nil {
			for _, p := range paramNames(fd.Function.ParameterList) {
				members = append(members, Member{
					Name: p,
					Kind: "param",
				})
			}
		}
		return members
	}
	return nil
}

func addInheritedMethods(
	classes map[string]*ast.ClassDeclaration,
	className string,
	out *[]Member,
	seenNames map[string]bool,
	visited map[string]bool,
) {
	if className == "" || visited[className] {
		return
	}
	visited[className] = true

	cd := classes[className]
	if cd == nil || cd.Class == nil {
		return
	}

	for _, elem := range cd.Class.Body {
		md, ok := elem.(*ast.MethodDefinition)
		if !ok {
			continue
		}
		name := methodName(md)
		if name == "constructor" || seenNames[name] {
			continue
		}
		preview := "()"
		if md.Body != nil && md.Body.ParameterList != nil {
			params := paramNames(md.Body.ParameterList)
			preview = "(" + strings.Join(params, ", ") + ")"
		}
		*out = append(*out, Member{
			Name:      name,
			Kind:      "function",
			Preview:   preview,
			Inherited: true,
			Source:    className,
		})
		seenNames[name] = true
	}

	addInheritedMethods(classes, superClassName(cd), out, seenNames, visited)
}

func indexClasses(program *ast.Program) map[string]*ast.ClassDeclaration {
	classes := map[string]*ast.ClassDeclaration{}
	for _, stmt := range program.Body {
		cd, ok := stmt.(*ast.ClassDeclaration)
		if !ok || cd.Class == nil || cd.Class.Name == nil {
			continue
		}
		classes[string(cd.Class.Name.Name)] = cd
	}
	return classes
}

func superClassName(cd *ast.ClassDeclaration) string {
	if cd == nil || cd.Class == nil || cd.Class.SuperClass == nil {
		return ""
	}
	if ident, ok := cd.Class.SuperClass.(*ast.Identifier); ok {
		return string(ident.Name)
	}
	return ""
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

func fieldName(fd *ast.FieldDefinition) string {
	if fd == nil {
		return "<unknown>"
	}
	switch k := fd.Key.(type) {
	case *ast.Identifier:
		return string(k.Name)
	case *ast.StringLiteral:
		return k.Literal
	}
	return "<computed>"
}

func paramNames(pl *ast.ParameterList) []string {
	var names []string
	if pl == nil {
		return names
	}
	for _, p := range pl.List {
		if p == nil || p.Target == nil {
			continue
		}
		if ident, ok := p.Target.(*ast.Identifier); ok {
			names = append(names, string(ident.Name))
		}
	}
	return names
}
