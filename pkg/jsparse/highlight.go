package jsparse

import (
	"github.com/charmbracelet/lipgloss"
)

// SyntaxClass identifies the highlighting category for a syntax token.
type SyntaxClass int

const (
	SyntaxNone        SyntaxClass = iota
	SyntaxKeyword                 // const, let, class, function, return, ...
	SyntaxString                  // "...", '...', `...`
	SyntaxNumber                  // 42, 3.14
	SyntaxComment                 // // ..., /* ... */
	SyntaxIdentifier              // variable/property names
	SyntaxOperator                // =, +, -, =>
	SyntaxPunctuation             // (, ), {, }, [, ]
)

// SyntaxSpan represents a highlighted region in source code.
type SyntaxSpan struct {
	StartLine int // 1-based
	StartCol  int // 1-based
	EndLine   int // 1-based
	EndCol    int // 1-based (exclusive)
	Class     SyntaxClass
}

// BuildSyntaxSpans walks a TSNode tree and returns syntax spans for leaf tokens.
func BuildSyntaxSpans(root *TSNode) []SyntaxSpan {
	if root == nil {
		return nil
	}
	var spans []SyntaxSpan
	appendSyntaxSpans(root, &spans)
	return spans
}

func appendSyntaxSpans(n *TSNode, spans *[]SyntaxSpan) {
	if n == nil {
		return
	}
	if len(n.Children) == 0 {
		class := ClassifySyntaxKind(n.Kind)
		if class != SyntaxNone {
			*spans = append(*spans, SyntaxSpan{
				StartLine: n.StartRow + 1,
				StartCol:  n.StartCol + 1,
				EndLine:   n.EndRow + 1,
				EndCol:    n.EndCol + 1,
				Class:     class,
			})
		}
		return
	}
	for _, child := range n.Children {
		appendSyntaxSpans(child, spans)
	}
}

// ClassifySyntaxKind maps a tree-sitter node kind to a SyntaxClass.
func ClassifySyntaxKind(kind string) SyntaxClass {
	switch kind {
	case "comment":
		return SyntaxComment
	case "string", "string_fragment", "template_string", "template_chars":
		return SyntaxString
	case "number":
		return SyntaxNumber
	case "identifier", "property_identifier", "shorthand_property_identifier",
		"shorthand_property_identifier_pattern":
		return SyntaxIdentifier
	case "const", "let", "var", "function", "return", "if", "else", "for", "while", "do",
		"switch", "case", "default", "break", "continue", "new", "class", "import", "export",
		"from", "try", "catch", "finally", "throw", "await", "async", "extends", "this",
		"typeof", "instanceof", "in", "of", "yield", "super", "true", "false", "null",
		"undefined", "void", "delete", "static", "get", "set":
		return SyntaxKeyword
	case ".", ",", ";", ":", "=", "==", "===", "!=", "!==", "+", "-", "*", "/", "%", "=>",
		"&&", "||", "!", ">", "<", ">=", "<=", "?", "++", "--", "+=", "-=", "*=", "/=",
		"**", "??", "?.", "...":
		return SyntaxOperator
	case "(", ")", "{", "}", "[", "]":
		return SyntaxPunctuation
	default:
		return SyntaxNone
	}
}

// SyntaxClassAt returns the syntax class for a given position (1-based line/col).
func SyntaxClassAt(spans []SyntaxSpan, lineNo, colNo int) SyntaxClass {
	for _, span := range spans {
		if inSyntaxRange(lineNo, colNo, span.StartLine, span.StartCol, span.EndLine, span.EndCol) {
			return span.Class
		}
	}
	return SyntaxNone
}

func inSyntaxRange(line, col, startLine, startCol, endLine, endCol int) bool {
	if line < startLine || line > endLine {
		return false
	}
	if line == startLine && col < startCol {
		return false
	}
	if line == endLine && col >= endCol {
		return false
	}
	return true
}

// --- Default color rendering ---

var (
	styleKeyword     = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	styleString      = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	styleNumber      = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	styleComment     = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	styleIdent       = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	styleOperator    = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	stylePunctuation = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
)

// RenderSyntaxChar renders a single character with syntax coloring.
func RenderSyntaxChar(class SyntaxClass, ch string) string {
	switch class {
	case SyntaxNone:
		return ch
	case SyntaxKeyword:
		return styleKeyword.Render(ch)
	case SyntaxString:
		return styleString.Render(ch)
	case SyntaxNumber:
		return styleNumber.Render(ch)
	case SyntaxComment:
		return styleComment.Render(ch)
	case SyntaxIdentifier:
		return styleIdent.Render(ch)
	case SyntaxOperator:
		return styleOperator.Render(ch)
	case SyntaxPunctuation:
		return stylePunctuation.Render(ch)
	}
	return ch
}
