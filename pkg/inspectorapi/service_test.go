package inspectorapi

import (
	"testing"

	inspectorruntime "github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
)

func openDoc(t *testing.T, svc *Service, src string) DocumentID {
	t.Helper()
	resp, err := svc.OpenDocument(OpenDocumentRequest{Filename: "test.js", Source: src})
	if err != nil {
		t.Fatalf("open document: %v", err)
	}
	if resp.DocumentID == "" {
		t.Fatalf("expected document id")
	}
	return resp.DocumentID
}

func TestListGlobalsAndMembers(t *testing.T) {
	svc := NewService()
	src := `
class Foo {
  bar(x) { return x }
}
function greet(name) { return name }
const cfg = { answer: 42 }
`
	docID := openDoc(t, svc, src)

	globals, err := svc.ListGlobals(ListGlobalsRequest{DocumentID: docID})
	if err != nil {
		t.Fatalf("list globals: %v", err)
	}
	if len(globals.Globals) < 3 {
		t.Fatalf("expected globals, got %d", len(globals.Globals))
	}

	classMembers, err := svc.ListMembers(ListMembersRequest{DocumentID: docID, GlobalName: "Foo"}, nil)
	if err != nil {
		t.Fatalf("list class members: %v", err)
	}
	if len(classMembers.Members) == 0 {
		t.Fatalf("expected class members")
	}

	fnMembers, err := svc.ListMembers(ListMembersRequest{DocumentID: docID, GlobalName: "greet"}, nil)
	if err != nil {
		t.Fatalf("list function members: %v", err)
	}
	if len(fnMembers.Members) == 0 {
		t.Fatalf("expected function members")
	}
}

func TestDeclarationLookups(t *testing.T) {
	svc := NewService()
	src := `
class Foo {
  bar(x) { return x }
}
`
	docID := openDoc(t, svc, src)

	binding, err := svc.BindingDeclarationLine(BindingDeclarationRequest{
		DocumentID: docID,
		Name:       "Foo",
	})
	if err != nil {
		t.Fatalf("binding declaration line: %v", err)
	}
	if !binding.Found || binding.Line <= 0 {
		t.Fatalf("expected binding declaration line, got %+v", binding)
	}

	member, err := svc.MemberDeclarationLine(MemberDeclarationRequest{
		DocumentID: docID,
		ClassName:  "Foo",
		MemberName: "bar",
	})
	if err != nil {
		t.Fatalf("member declaration line: %v", err)
	}
	if !member.Found || member.Line <= 0 {
		t.Fatalf("expected member declaration line, got %+v", member)
	}
}

func TestRuntimeMergeAndValueMembers(t *testing.T) {
	svc := NewService()
	src := `
const cfg = { answer: 42 }
`
	docID := openDoc(t, svc, src)

	rt := inspectorruntime.NewSession()
	if err := rt.Load(src); err != nil {
		t.Fatalf("runtime load: %v", err)
	}
	if err := rt.Load(`const replAdded = 1`); err != nil {
		t.Fatalf("runtime load repl: %v", err)
	}

	members, err := svc.ListMembers(ListMembersRequest{DocumentID: docID, GlobalName: "cfg"}, rt)
	if err != nil {
		t.Fatalf("list runtime value members: %v", err)
	}
	if len(members.Members) == 0 {
		t.Fatalf("expected runtime value members")
	}

	declared := DeclaredBindingsFromSource(`const replAdded = 1`)
	merged, err := svc.MergeRuntimeGlobals(MergeRuntimeGlobalsRequest{
		DocumentID: docID,
		Declared:   declared,
	}, rt)
	if err != nil {
		t.Fatalf("merge runtime globals: %v", err)
	}

	found := false
	for _, g := range merged.Globals {
		if g.Name == "replAdded" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected merged globals to include replAdded")
	}
}

func TestTreeAndSyncFacade(t *testing.T) {
	svc := NewService()
	src := `
const greeting = "hello"
console.log(greeting)
`
	docID := openDoc(t, svc, src)

	rows, err := svc.BuildTreeRows(BuildTreeRowsRequest{DocumentID: docID})
	if err != nil {
		t.Fatalf("build tree rows: %v", err)
	}
	if len(rows.Rows) == 0 {
		t.Fatalf("expected tree rows")
	}

	syncFromSource, err := svc.SyncSourceToTree(SyncSourceToTreeRequest{
		DocumentID: docID,
		CursorLine: 2,
		CursorCol:  13,
	})
	if err != nil {
		t.Fatalf("sync source to tree: %v", err)
	}
	if !syncFromSource.Found {
		t.Fatalf("expected source->tree selection")
	}

	syncFromTree, err := svc.SyncTreeToSource(SyncTreeToSourceRequest{
		DocumentID:    docID,
		SelectedIndex: syncFromSource.VisibleIndex,
	})
	if err != nil {
		t.Fatalf("sync tree to source: %v", err)
	}
	if !syncFromTree.Found {
		t.Fatalf("expected tree->source selection")
	}
}

func TestUpdateAndCloseDocument(t *testing.T) {
	svc := NewService()
	docID := openDoc(t, svc, "const a = 1")

	if _, err := svc.UpdateDocument(UpdateDocumentRequest{DocumentID: docID, Source: "const b = 2"}); err != nil {
		t.Fatalf("update document: %v", err)
	}

	if err := svc.CloseDocument(CloseDocumentRequest{DocumentID: docID}); err != nil {
		t.Fatalf("close document: %v", err)
	}

	if _, err := svc.ListGlobals(ListGlobalsRequest{DocumentID: docID}); err == nil {
		t.Fatalf("expected document not found after close")
	}
}
