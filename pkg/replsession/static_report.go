package replsession

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dop251/goja/ast"
	inspectoranalysis "github.com/go-go-golems/go-go-goja/pkg/inspector/analysis"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// rangeFromNode converts an AST node's source position into a RangeView.
func rangeFromNode(result *jsparse.AnalysisResult, n ast.Node) *RangeView {
	if result == nil || result.Index == nil || n == nil {
		return nil
	}
	startLine, startCol := result.Index.OffsetToLineCol(int(n.Idx0()))
	endLine, endCol := result.Index.OffsetToLineCol(int(n.Idx1()))
	return &RangeView{
		StartLine: startLine,
		StartCol:  startCol,
		EndLine:   endLine,
		EndCol:    endCol,
	}
}

// buildStaticReport produces the StaticReport view from analysis and optional CST data.
func buildStaticReport(result *jsparse.AnalysisResult, cstRoot *jsparse.TSNode, maxASTRows, maxCSTRows int) StaticReport {
	report := StaticReport{
		Diagnostics:      []DiagnosticView{},
		TopLevelBindings: []TopLevelBindingView{},
		Unresolved:       []IdentifierUseView{},
		References:       []BindingReferenceGroup{},
		AST:              []ASTRowView{},
		CST:              []CSTNodeView{},
		Summary:          []StaticSummaryFact{},
	}
	if result == nil {
		return report
	}
	for _, d := range result.Diagnostics() {
		report.Diagnostics = append(report.Diagnostics, DiagnosticView{
			Severity: d.Severity,
			Message:  d.Message,
		})
	}

	session := inspectoranalysis.NewSessionFromResult(result)
	globals := session.Globals()
	for _, g := range globals {
		line, _ := session.BindingDeclLine(g.Name)
		snippet := declarationSnippet(result, g.Name)
		refs := bindingReferences(result, g.Name)
		report.TopLevelBindings = append(report.TopLevelBindings, TopLevelBindingView{
			Name:           g.Name,
			Kind:           g.Kind.String(),
			Line:           line,
			Snippet:        snippet,
			Extends:        g.Extends,
			ReferenceCount: len(refs),
		})
		report.References = append(report.References, BindingReferenceGroup{
			Name:      g.Name,
			Kind:      g.Kind.String(),
			Locations: refs,
		})
	}

	if result.Resolution != nil && result.Index != nil {
		for _, nodeID := range result.Resolution.Unresolved {
			node := result.Index.Nodes[nodeID]
			if node == nil {
				continue
			}
			report.Unresolved = append(report.Unresolved, IdentifierUseView{
				Line:    node.StartLine,
				Col:     node.StartCol,
				NodeID:  int(node.ID),
				Snippet: node.Snippet,
			})
		}
		report.Scope = buildScopeView(result.Resolution, result.Resolution.RootScopeID)
	}

	if result.Index != nil {
		expandAllNodes(result.Index, result.Index.RootID)
		rows := inspectorRows(result.Index)
		report.ASTNodeCount = len(rows)
		if maxASTRows > 0 && len(rows) > maxASTRows {
			report.ASTTruncated = true
			rows = rows[:maxASTRows]
		}
		report.AST = append(report.AST, rows...)
	}

	if exprStmt, _, ok := finalExpressionStatement(result, result.Source); ok {
		report.FinalExpression = rangeFromNode(result, exprStmt.Expression)
	}

	cstRows, totalCST := flattenCST(cstRoot, maxCSTRows)
	report.CST = cstRows
	report.CSTNodeCount = totalCST
	report.CSTTruncated = maxCSTRows > 0 && totalCST > maxCSTRows
	report.Summary = []StaticSummaryFact{
		{Label: "diagnostics", Value: fmt.Sprintf("%d", len(report.Diagnostics))},
		{Label: "top-level bindings", Value: fmt.Sprintf("%d", len(report.TopLevelBindings))},
		{Label: "unresolved identifiers", Value: fmt.Sprintf("%d", len(report.Unresolved))},
		{Label: "AST nodes", Value: fmt.Sprintf("%d", report.ASTNodeCount)},
		{Label: "CST nodes", Value: fmt.Sprintf("%d", report.CSTNodeCount)},
	}
	return report
}

func declarationSnippet(result *jsparse.AnalysisResult, name string) string {
	if result == nil || result.Index == nil || result.Resolution == nil {
		return ""
	}
	root := result.Resolution.Scopes[result.Resolution.RootScopeID]
	if root == nil {
		return ""
	}
	binding := root.Bindings[name]
	if binding == nil {
		return ""
	}
	node := result.Index.Nodes[binding.DeclNodeID]
	if node == nil {
		return ""
	}
	return node.Snippet
}

func bindingReferences(result *jsparse.AnalysisResult, name string) []IdentifierUseView {
	if result == nil || result.Resolution == nil || result.Index == nil {
		return nil
	}
	entries := inspectoranalysis.CrossReferences(result.Resolution, result.Index, name)
	out := make([]IdentifierUseView, 0, len(entries))
	for _, entry := range entries {
		node := result.Index.Nodes[entry.NodeID]
		snippet := ""
		if node != nil {
			snippet = node.Snippet
		}
		out = append(out, IdentifierUseView{
			Line:    entry.Line,
			Col:     entry.Col,
			Context: entry.Context,
			NodeID:  int(entry.NodeID),
			Snippet: snippet,
		})
	}
	return out
}

func buildScopeView(res *jsparse.Resolution, scopeID jsparse.ScopeID) *ScopeView {
	if res == nil {
		return nil
	}
	scope := res.Scopes[scopeID]
	if scope == nil {
		return nil
	}
	bindings := make([]ScopeBinding, 0, len(scope.Bindings))
	for name, binding := range scope.Bindings {
		if binding == nil {
			continue
		}
		bindings = append(bindings, ScopeBinding{Name: name, Kind: binding.Kind.String()})
	}
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].Kind != bindings[j].Kind {
			return bindings[i].Kind < bindings[j].Kind
		}
		return bindings[i].Name < bindings[j].Name
	})
	children := make([]*ScopeView, 0, len(scope.Children))
	for _, childID := range scope.Children {
		if child := buildScopeView(res, childID); child != nil {
			children = append(children, child)
		}
	}
	return &ScopeView{
		ID:       int(scope.ID),
		Kind:     scope.Kind.String(),
		Start:    scope.Start,
		End:      scope.End,
		Bindings: bindings,
		Children: children,
	}
}

func expandAllNodes(idx *jsparse.Index, id jsparse.NodeID) {
	if idx == nil || id < 0 {
		return
	}
	node := idx.Nodes[id]
	if node == nil {
		return
	}
	node.Expanded = true
	for _, childID := range node.ChildIDs {
		expandAllNodes(idx, childID)
	}
}

func inspectorRows(idx *jsparse.Index) []ASTRowView {
	rows := make([]ASTRowView, 0)
	if idx == nil {
		return rows
	}
	for _, row := range buildRows(idx) {
		rows = append(rows, ASTRowView{
			NodeID:      int(row.NodeID),
			Title:       row.Title,
			Description: row.Description,
		})
	}
	return rows
}

func buildRows(idx *jsparse.Index) []rowLike {
	rows := make([]rowLike, 0)
	if idx == nil {
		return rows
	}
	for _, id := range idx.VisibleNodes() {
		node := idx.Nodes[id]
		if node == nil {
			continue
		}
		rows = append(rows, rowLike{
			NodeID:      id,
			Title:       fmt.Sprintf("%s%s", strings.Repeat("  ", node.Depth), node.DisplayLabel()),
			Description: fmt.Sprintf("[%d..%d] %s", node.Start, node.End, trimForDisplay(node.Snippet, 80)),
		})
	}
	return rows
}

type rowLike struct {
	NodeID      jsparse.NodeID
	Title       string
	Description string
}

func flattenCST(root *jsparse.TSNode, maxRows int) ([]CSTNodeView, int) {
	if root == nil {
		return nil, 0
	}
	rows := make([]CSTNodeView, 0)
	total := 0
	var walk func(node *jsparse.TSNode, depth int)
	walk = func(node *jsparse.TSNode, depth int) {
		if node == nil {
			return
		}
		total++
		if maxRows <= 0 || len(rows) < maxRows {
			rows = append(rows, CSTNodeView{
				Depth:     depth,
				Kind:      node.Kind,
				Text:      trimForDisplay(strings.ReplaceAll(strings.TrimSpace(node.Text), "\n", "\\n"), 80),
				StartRow:  node.StartRow,
				StartCol:  node.StartCol,
				EndRow:    node.EndRow,
				EndCol:    node.EndCol,
				IsError:   node.IsError,
				IsMissing: node.IsMissing,
			})
		}
		for _, child := range node.Children {
			walk(child, depth+1)
		}
	}
	walk(root, 0)
	return rows, total
}

func trimForDisplay(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen < 2 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}
