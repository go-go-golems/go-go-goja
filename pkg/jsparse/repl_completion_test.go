package jsparse

import (
	"slices"
	"testing"

	"github.com/dop251/goja"
)

func TestExtractRequireAliases(t *testing.T) {
	input := `
const fs = require("fs");
let path = require('path');
var url = require("url");
const notAlias = make("x");
`
	aliases := ExtractRequireAliases(input)
	if aliases["fs"] != "fs" {
		t.Fatalf("expected fs alias, got %#v", aliases)
	}
	if aliases["path"] != "path" {
		t.Fatalf("expected path alias, got %#v", aliases)
	}
	if aliases["url"] != "url" {
		t.Fatalf("expected url alias, got %#v", aliases)
	}
	if _, ok := aliases["notAlias"]; ok {
		t.Fatalf("did not expect notAlias to be extracted")
	}
}

func TestExtractTopLevelBindingCandidates(t *testing.T) {
	source := `
const dataBucket = { count: 1 };
let total = 10;
var legacy = true;
function greetUser(name) { return name; }
`

	candidates := ExtractTopLevelBindingCandidates(source)
	labels := make([]string, 0, len(candidates))
	for _, c := range candidates {
		labels = append(labels, c.Label)
	}
	if !slices.Contains(labels, "dataBucket") {
		t.Fatalf("expected dataBucket in top-level bindings, got %v", labels)
	}
	if !slices.Contains(labels, "total") {
		t.Fatalf("expected total in top-level bindings, got %v", labels)
	}
	if !slices.Contains(labels, "legacy") {
		t.Fatalf("expected legacy in top-level bindings, got %v", labels)
	}
	if !slices.Contains(labels, "greetUser") {
		t.Fatalf("expected greetUser in top-level bindings, got %v", labels)
	}
}

func TestAugmentREPLCandidatesIdentifierIncludesRuntimeLexicals(t *testing.T) {
	r := goja.New()
	snippet := `const dataBucket = { count: 1, label: "demo" }; function greetUser(name) { return name; }`
	if _, err := r.RunString(snippet); err != nil {
		t.Fatalf("run snippet: %v", err)
	}

	hints := ExtractTopLevelBindingCandidates(snippet)
	ctx := CompletionContext{
		Kind:        CompletionIdentifier,
		PartialText: "dataB",
	}

	got := AugmentREPLCandidates(r, "", ctx, nil, hints)
	labels := make([]string, 0, len(got))
	for _, c := range got {
		labels = append(labels, c.Label)
	}
	if !slices.Contains(labels, "dataBucket") {
		t.Fatalf("expected runtime lexical completion for dataBucket, got %v", labels)
	}
}

func TestAugmentREPLCandidatesPropertyIncludesRuntimeObjectKeys(t *testing.T) {
	r := goja.New()
	snippet := `const dataBucket = { count: 1, label: "demo" };`
	if _, err := r.RunString(snippet); err != nil {
		t.Fatalf("run snippet: %v", err)
	}
	hints := ExtractTopLevelBindingCandidates(snippet)
	ctx := CompletionContext{
		Kind:        CompletionProperty,
		BaseExpr:    "dataBucket",
		PartialText: "",
	}

	got := AugmentREPLCandidates(r, "", ctx, nil, hints)
	labels := make([]string, 0, len(got))
	for _, c := range got {
		labels = append(labels, c.Label)
	}
	if !slices.Contains(labels, "count") {
		t.Fatalf("expected runtime property count, got %v", labels)
	}
	if !slices.Contains(labels, "label") {
		t.Fatalf("expected runtime property label, got %v", labels)
	}
}

func TestAugmentREPLCandidatesKeepsStaticPriorityOnDuplicates(t *testing.T) {
	r := goja.New()
	if _, err := r.RunString(`function console() { return "shadow"; }`); err != nil {
		t.Fatalf("run snippet: %v", err)
	}
	hints := ExtractTopLevelBindingCandidates(`function console() { return "shadow"; }`)
	ctx := CompletionContext{
		Kind:        CompletionIdentifier,
		PartialText: "cons",
	}
	static := []CompletionCandidate{
		{Label: "console", Kind: CandidateVariable, Detail: "global"},
	}

	got := AugmentREPLCandidates(r, "", ctx, static, hints)
	if len(got) == 0 {
		t.Fatalf("expected candidates, got none")
	}
	if got[0].Label != "console" {
		t.Fatalf("expected static console first, got %#v", got[0])
	}
	if got[0].Detail != "global" {
		t.Fatalf("expected static detail to win dedupe, got %q", got[0].Detail)
	}
}
