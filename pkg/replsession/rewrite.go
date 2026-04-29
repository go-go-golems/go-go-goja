package replsession

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dop251/goja/ast"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func buildRewrite(source string, result *jsparse.AnalysisResult, cellID int) RewriteReport {
	report := RewriteReport{
		Mode:          "async-iife-with-binding-capture",
		DeclaredNames: []string{},
		HelperNames:   []string{},
		Operations:    []RewriteStep{},
		Warnings:      []string{},
	}
	if result == nil {
		report.TransformedSource = source
		report.Warnings = append(report.Warnings, "analysis result missing; source left unchanged")
		return report
	}

	report.DeclaredNames = declaredNamesFromResult(result)
	helperLast := fmt.Sprintf("__ggg_repl_last_%d__", cellID)
	helperBindings := fmt.Sprintf("__ggg_repl_bindings_%d__", cellID)
	report.HelperNames = []string{helperLast, helperBindings}
	report.LastHelperName = helperLast
	report.BindingHelperName = helperBindings
	report.Operations = append(report.Operations,
		RewriteStep{Kind: "wrap", Detail: "wrap cell source in an async IIFE so lexical declarations become cell-local"},
		RewriteStep{Kind: "capture-bindings", Detail: "return top-level declarations as an object so they can be persisted back into the session"},
	)

	body := source
	if exprStmt, exprSnippet, ok := finalExpressionStatement(result, source); ok {
		report.CapturedLastExpr = true
		report.FinalExpressionSrc = exprSnippet
		report.FinalExpressionSrc = strings.TrimSpace(report.FinalExpressionSrc)
		report.Operations = append(report.Operations,
			RewriteStep{Kind: "capture-last-expression", Detail: "replace the final top-level expression statement with an assignment to a hidden helper"},
		)
		report.FinalExpressionSrc = trimForDisplay(report.FinalExpressionSrc, 120)
		replacement := helperLast + " = (" + exprSnippet + ");"
		body = replaceSourceRange(body, int(exprStmt.Idx0()), int(exprStmt.Idx1()), replacement)
	} else {
		report.Operations = append(report.Operations,
			RewriteStep{Kind: "no-final-expression", Detail: "last top-level statement was not an expression; REPL result defaults to undefined unless user assigns explicitly"},
		)
	}

	bindingLines := make([]string, 0, len(report.DeclaredNames))
	for _, name := range report.DeclaredNames {
		bindingLines = append(bindingLines,
			fmt.Sprintf("      %q: (typeof %s === \"undefined\" ? undefined : %s)", name, name, name),
		)
	}
	bindingsBody := ""
	if len(bindingLines) > 0 {
		bindingsBody = strings.Join(bindingLines, ",\n") + "\n"
	}

	var builder strings.Builder
	builder.WriteString("(async function () {\n")
	builder.WriteString("  let ")
	builder.WriteString(helperLast)
	builder.WriteString(";\n")
	builder.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		builder.WriteByte('\n')
	}
	builder.WriteString("  return {\n")
	builder.WriteString("    ")
	fmt.Fprintf(&builder, "%q", helperBindings)
	builder.WriteString(": {\n")
	builder.WriteString(bindingsBody)
	builder.WriteString("    },\n")
	builder.WriteString("    ")
	fmt.Fprintf(&builder, "%q", helperLast)
	builder.WriteString(": (typeof ")
	builder.WriteString(helperLast)
	builder.WriteString(" === \"undefined\" ? undefined : ")
	builder.WriteString(helperLast)
	builder.WriteString(")\n")
	builder.WriteString("  };\n")
	builder.WriteString("})()")
	report.TransformedSource = builder.String()
	return report
}

func declaredNamesFromResult(result *jsparse.AnalysisResult) []string {
	if result == nil || result.Resolution == nil {
		return nil
	}
	root := result.Resolution.Scopes[result.Resolution.RootScopeID]
	if root == nil {
		return nil
	}
	names := make([]string, 0, len(root.Bindings))
	for name := range root.Bindings {
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func finalExpressionStatement(result *jsparse.AnalysisResult, source string) (*ast.ExpressionStatement, string, bool) {
	if result == nil || result.Program == nil || len(result.Program.Body) == 0 {
		return nil, "", false
	}
	last := result.Program.Body[len(result.Program.Body)-1]
	exprStmt, ok := last.(*ast.ExpressionStatement)
	if !ok || exprStmt.Expression == nil {
		return nil, "", false
	}
	exprSource := sourceSlice(source, int(exprStmt.Expression.Idx0()), int(exprStmt.Expression.Idx1()))
	exprSource = strings.TrimSpace(exprSource)
	exprSource = strings.TrimRight(exprSource, "; \t\r\n")
	if strings.TrimSpace(exprSource) == "" {
		return nil, "", false
	}
	return exprStmt, exprSource, true
}

func sourceSlice(source string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end < start {
		end = start
	}
	s := start - 1
	e := end - 1
	if s > len(source) {
		s = len(source)
	}
	if e > len(source) {
		e = len(source)
	}
	return source[s:e]
}

func replaceSourceRange(source string, start, end int, replacement string) string {
	if start < 1 {
		start = 1
	}
	if end < start {
		end = start
	}
	s := start - 1
	e := end - 1
	if s > len(source) {
		s = len(source)
	}
	if e > len(source) {
		e = len(source)
	}
	return source[:s] + replacement + source[e:]
}
