package inspectorapi

import (
	"errors"

	inspectornavigation "github.com/go-go-golems/go-go-goja/pkg/inspector/navigation"
	inspectortree "github.com/go-go-golems/go-go-goja/pkg/inspector/tree"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// DocumentID identifies one analysis document session in the service.
type DocumentID string

var (
	ErrDocumentNotFound = errors.New("inspectorapi: document not found")
	ErrGlobalNotFound   = errors.New("inspectorapi: global not found")
)

// Global is the user-facing global binding DTO.
type Global struct {
	Name    string
	Kind    jsparse.BindingKind
	Extends string
}

// Member is the user-facing member DTO.
type Member struct {
	Name           string
	Kind           string
	Preview        string
	Inherited      bool
	Source         string
	RuntimeDerived bool
}

// DeclaredBinding is the user-facing declaration DTO for REPL snippets.
type DeclaredBinding struct {
	Name string
	Kind jsparse.BindingKind
}

// OpenDocumentRequest opens/analyzes source into a service document session.
type OpenDocumentRequest struct {
	Filename string
	Source   string
}

// OpenDocumentFromAnalysisRequest registers an existing analysis result.
type OpenDocumentFromAnalysisRequest struct {
	Filename string
	Source   string
	Analysis *jsparse.AnalysisResult
}

// OpenDocumentResponse describes one opened document session.
type OpenDocumentResponse struct {
	DocumentID  DocumentID
	Analysis    *jsparse.AnalysisResult
	Diagnostics []jsparse.Diagnostic
}

// UpdateDocumentRequest reparses and refreshes an existing document session.
type UpdateDocumentRequest struct {
	DocumentID DocumentID
	Source     string
}

// UpdateDocumentResponse includes the updated analysis and diagnostics.
type UpdateDocumentResponse struct {
	Analysis    *jsparse.AnalysisResult
	Diagnostics []jsparse.Diagnostic
}

// CloseDocumentRequest closes and removes a document session.
type CloseDocumentRequest struct {
	DocumentID DocumentID
}

// ListGlobalsRequest requests globals for one document.
type ListGlobalsRequest struct {
	DocumentID DocumentID
}

// ListGlobalsResponse includes all global bindings for a document.
type ListGlobalsResponse struct {
	Globals []Global
}

// ListMembersRequest requests members for one selected global.
type ListMembersRequest struct {
	DocumentID DocumentID
	GlobalName string
}

// ListMembersResponse includes all members for the selected global.
type ListMembersResponse struct {
	Members []Member
}

// BindingDeclarationRequest requests declaration line lookup for a global binding.
type BindingDeclarationRequest struct {
	DocumentID DocumentID
	Name       string
}

// BindingDeclarationResponse returns a 1-based declaration line when found.
type BindingDeclarationResponse struct {
	Line  int
	Found bool
}

// MemberDeclarationRequest requests declaration line lookup for a member.
type MemberDeclarationRequest struct {
	DocumentID  DocumentID
	ClassName   string
	SourceClass string
	MemberName  string
}

// MemberDeclarationResponse returns a 1-based declaration line when found.
type MemberDeclarationResponse struct {
	Line  int
	Found bool
}

// MergeRuntimeGlobalsRequest merges static globals with runtime and REPL declarations.
type MergeRuntimeGlobalsRequest struct {
	DocumentID DocumentID
	Existing   []Global
	Declared   []DeclaredBinding
}

// MergeRuntimeGlobalsResponse returns merged/sorted globals.
type MergeRuntimeGlobalsResponse struct {
	Globals []Global
}

// BuildTreeRowsRequest builds tree row DTOs for a document index.
type BuildTreeRowsRequest struct {
	DocumentID      DocumentID
	UsageHighlights []jsparse.NodeID
}

// BuildTreeRowsResponse includes UI-agnostic tree rows.
type BuildTreeRowsResponse struct {
	Rows []inspectortree.Row
}

// SyncSourceToTreeRequest maps a source cursor to tree selection.
// CursorLine/CursorCol are 0-based, mirroring Bubble Tea cursor conventions.
type SyncSourceToTreeRequest struct {
	DocumentID DocumentID
	CursorLine int
	CursorCol  int
}

// SyncSourceToTreeResponse includes selected node and visible-tree index.
type SyncSourceToTreeResponse struct {
	Selection    inspectornavigation.SourceSelection
	VisibleIndex int
	Found        bool
}

// SyncTreeToSourceRequest maps selected tree row index to source cursor.
type SyncTreeToSourceRequest struct {
	DocumentID    DocumentID
	SelectedIndex int
}

// SyncTreeToSourceResponse includes source cursor/highlight selection.
type SyncTreeToSourceResponse struct {
	Selection inspectornavigation.TreeSelection
	Found     bool
}
