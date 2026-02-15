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

func TestAugmentREPLCandidatesIdentifierDoesNotExecuteThrowingGetter(t *testing.T) {
	r := goja.New()
	_, err := r.RunString(`
let getterHits = 0;
Object.defineProperty(globalThis, "boom", {
	enumerable: true,
	get() {
		getterHits++;
		throw new Error("boom");
	},
});
`)
	if err != nil {
		t.Fatalf("run snippet: %v", err)
	}

	ctx := CompletionContext{
		Kind:        CompletionIdentifier,
		PartialText: "bo",
	}
	got := AugmentREPLCandidates(r, "", ctx, nil, nil)
	labels := make([]string, 0, len(got))
	for _, c := range got {
		labels = append(labels, c.Label)
	}
	if !slices.Contains(labels, "boom") {
		t.Fatalf("expected boom candidate, got %v", labels)
	}

	hits := r.Get("getterHits")
	if hits.ToInteger() != 0 {
		t.Fatalf("expected getter not to execute during identifier completion, got %d hits", hits.ToInteger())
	}
}

func TestAugmentREPLCandidatesIdentifierKeepsUndefinedAndNullHints(t *testing.T) {
	r := goja.New()
	snippet := `
let foo;
const nilish = null;
`
	if _, err := r.RunString(snippet); err != nil {
		t.Fatalf("run snippet: %v", err)
	}

	hints := ExtractTopLevelBindingCandidates(snippet)
	ctx := CompletionContext{
		Kind: CompletionIdentifier,
	}
	got := AugmentREPLCandidates(r, "", ctx, nil, hints)
	labels := make([]string, 0, len(got))
	for _, c := range got {
		labels = append(labels, c.Label)
	}
	if !slices.Contains(labels, "foo") {
		t.Fatalf("expected undefined lexical foo to remain in completions, got %v", labels)
	}
	if !slices.Contains(labels, "nilish") {
		t.Fatalf("expected null lexical nilish to remain in completions, got %v", labels)
	}
}

func TestAugmentREPLCandidatesPropertyHandlesThrowingBaseGetter(t *testing.T) {
	r := goja.New()
	_, err := r.RunString(`
Object.defineProperty(globalThis, "boom", {
	enumerable: true,
	get() {
		throw new Error("boom");
	},
});
`)
	if err != nil {
		t.Fatalf("run snippet: %v", err)
	}

	ctx := CompletionContext{
		Kind:     CompletionProperty,
		BaseExpr: "boom",
	}
	got := AugmentREPLCandidates(r, "", ctx, nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected no property candidates when base getter throws, got %v", got)
	}
}
