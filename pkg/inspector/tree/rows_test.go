package tree

import (
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func TestBuildRowsFromIndex(t *testing.T) {
	src := "const x = 1;\nfunction f(y){return x+y}\n"
	res := jsparse.Analyze("test.js", src, nil)
	if res == nil || res.Index == nil {
		t.Fatalf("expected analysis with index")
	}

	rows := BuildRowsFromIndex(res.Index, nil)
	if len(rows) == 0 {
		t.Fatalf("expected non-empty rows")
	}
	if rows[0].Title == "" || rows[0].Description == "" {
		t.Fatalf("expected title/description, got %+v", rows[0])
	}
}

func TestBuildRowUsageHint(t *testing.T) {
	src := "const x = 1;\nconsole.log(x)\n"
	res := jsparse.Analyze("test.js", src, nil)
	if res == nil || res.Index == nil {
		t.Fatalf("expected analysis with index")
	}

	var target jsparse.NodeID = -1
	for id, n := range res.Index.Nodes {
		if n != nil && n.Kind == "Identifier" && strings.Contains(n.DisplayLabel(), "x") {
			target = id
			break
		}
	}
	if target < 0 {
		t.Fatalf("failed to find identifier node")
	}

	row := BuildRow(res.Index.Nodes[target], []jsparse.NodeID{target}, res.Resolution)
	if !strings.Contains(row.Description, "★usage") {
		t.Fatalf("expected usage hint in description: %q", row.Description)
	}
}

func TestBuildRowScopeHints(t *testing.T) {
	src := "const x = 1;\nconsole.log(x)\n"
	res := jsparse.Analyze("test.js", src, nil)
	if res == nil || res.Index == nil || res.Resolution == nil {
		t.Fatalf("expected analysis with resolution")
	}

	foundDecl := false
	foundRef := false
	for _, n := range res.Index.Nodes {
		if n == nil || n.Kind != "Identifier" || !strings.Contains(n.DisplayLabel(), "x") {
			continue
		}
		row := BuildRow(n, nil, res.Resolution)
		if strings.Contains(row.Description, "decl") {
			foundDecl = true
		}
		if strings.Contains(row.Description, "[ref]") {
			foundRef = true
		}
	}
	if !foundDecl {
		t.Fatalf("expected declaration hint for x")
	}
	if !foundRef {
		t.Fatalf("expected reference hint for x")
	}
}
