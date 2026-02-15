package inspectorapi

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"
	inspectoranalysis "github.com/go-go-golems/go-go-goja/pkg/inspector/analysis"
	inspectorcore "github.com/go-go-golems/go-go-goja/pkg/inspector/core"
	inspectornavigation "github.com/go-go-golems/go-go-goja/pkg/inspector/navigation"
	inspectorruntime "github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	inspectortree "github.com/go-go-golems/go-go-goja/pkg/inspector/tree"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

type documentSession struct {
	id       DocumentID
	filename string
	source   string
	lines    []string
	analysis *jsparse.AnalysisResult
	session  *inspectoranalysis.Session
}

// Service exposes task-oriented inspector operations over extracted analysis/runtime packages.
type Service struct {
	mu     sync.RWMutex
	nextID int
	docs   map[DocumentID]*documentSession
}

// NewService creates an empty inspector API service.
func NewService() *Service {
	return &Service{docs: map[DocumentID]*documentSession{}}
}

// OpenDocument parses source and creates a new document session.
func (s *Service) OpenDocument(req OpenDocumentRequest) (OpenDocumentResponse, error) {
	result := jsparse.Analyze(req.Filename, req.Source, nil)
	return s.openFromAnalysis(OpenDocumentFromAnalysisRequest{
		Filename: req.Filename,
		Source:   req.Source,
		Analysis: result,
	})
}

// OpenDocumentFromAnalysis registers an existing analysis result as a new document session.
func (s *Service) OpenDocumentFromAnalysis(req OpenDocumentFromAnalysisRequest) (OpenDocumentResponse, error) {
	return s.openFromAnalysis(req)
}

func (s *Service) openFromAnalysis(req OpenDocumentFromAnalysisRequest) (OpenDocumentResponse, error) {
	result := req.Analysis
	if result == nil {
		result = jsparse.Analyze(req.Filename, req.Source, nil)
	}

	doc := &documentSession{
		filename: req.Filename,
		source:   req.Source,
		lines:    strings.Split(req.Source, "\n"),
		analysis: result,
		session:  inspectoranalysis.NewSessionFromResult(result),
	}

	s.mu.Lock()
	s.nextID++
	doc.id = DocumentID(fmt.Sprintf("doc-%d", s.nextID))
	s.docs[doc.id] = doc
	s.mu.Unlock()

	return OpenDocumentResponse{
		DocumentID:  doc.id,
		Analysis:    result,
		Diagnostics: result.Diagnostics(),
	}, nil
}

// UpdateDocument reparses source for an existing document session.
func (s *Service) UpdateDocument(req UpdateDocumentRequest) (UpdateDocumentResponse, error) {
	s.mu.Lock()
	doc, ok := s.docs[req.DocumentID]
	if !ok {
		s.mu.Unlock()
		return UpdateDocumentResponse{}, ErrDocumentNotFound
	}
	filename := doc.filename
	result := jsparse.Analyze(filename, req.Source, nil)
	doc.source = req.Source
	doc.lines = strings.Split(req.Source, "\n")
	doc.analysis = result
	doc.session = inspectoranalysis.NewSessionFromResult(result)
	s.mu.Unlock()

	return UpdateDocumentResponse{
		Analysis:    result,
		Diagnostics: result.Diagnostics(),
	}, nil
}

// CloseDocument removes an existing document session.
func (s *Service) CloseDocument(req CloseDocumentRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.docs[req.DocumentID]; !ok {
		return ErrDocumentNotFound
	}
	delete(s.docs, req.DocumentID)
	return nil
}

// Analysis returns the analysis result for a document.
func (s *Service) Analysis(documentID DocumentID) (*jsparse.AnalysisResult, error) {
	doc, err := s.getDocument(documentID)
	if err != nil {
		return nil, err
	}
	return doc.analysis, nil
}

// ListGlobals returns static globals for a document.
func (s *Service) ListGlobals(req ListGlobalsRequest) (ListGlobalsResponse, error) {
	doc, err := s.getDocument(req.DocumentID)
	if err != nil {
		return ListGlobalsResponse{}, err
	}
	globals := doc.session.Globals()
	return ListGlobalsResponse{Globals: mapAnalysisGlobals(globals)}, nil
}

// ListMembers returns members for the selected global. Value members are runtime-derived when runtime is provided.
func (s *Service) ListMembers(req ListMembersRequest, rt *inspectorruntime.Session) (ListMembersResponse, error) {
	doc, err := s.getDocument(req.DocumentID)
	if err != nil {
		return ListMembersResponse{}, err
	}

	globals := doc.session.Globals()
	selected, ok := findGlobalByName(globals, req.GlobalName)
	if !ok {
		return ListMembersResponse{}, ErrGlobalNotFound
	}

	//exhaustive:ignore
	switch selected.Kind {
	case jsparse.BindingClass:
		members := doc.session.ClassMembers(selected.Name)
		return ListMembersResponse{Members: mapClassMembers(members)}, nil
	case jsparse.BindingFunction:
		members := doc.session.FunctionMembers(selected.Name)
		return ListMembersResponse{Members: mapFunctionMembers(members)}, nil
	default:
		if rt == nil {
			return ListMembersResponse{}, nil
		}
		return ListMembersResponse{Members: runtimeValueMembers(rt, selected.Name)}, nil
	}
}

// BindingDeclarationLine finds the 1-based declaration line for a global binding.
func (s *Service) BindingDeclarationLine(req BindingDeclarationRequest) (BindingDeclarationResponse, error) {
	doc, err := s.getDocument(req.DocumentID)
	if err != nil {
		return BindingDeclarationResponse{}, err
	}
	line, ok := doc.session.BindingDeclLine(req.Name)
	return BindingDeclarationResponse{Line: line, Found: ok}, nil
}

// MemberDeclarationLine finds the 1-based declaration line for a class member.
func (s *Service) MemberDeclarationLine(req MemberDeclarationRequest) (MemberDeclarationResponse, error) {
	doc, err := s.getDocument(req.DocumentID)
	if err != nil {
		return MemberDeclarationResponse{}, err
	}
	line, ok := doc.session.MemberDeclLine(req.ClassName, req.SourceClass, req.MemberName)
	return MemberDeclarationResponse{Line: line, Found: ok}, nil
}

// DeclaredBindingsFromSource extracts top-level declarations from snippet source.
func DeclaredBindingsFromSource(source string) []DeclaredBinding {
	declared := inspectoranalysis.DeclaredBindingsFromSource(source)
	out := make([]DeclaredBinding, 0, len(declared))
	for _, d := range declared {
		out = append(out, DeclaredBinding{Name: d.Name, Kind: d.Kind})
	}
	return out
}

// MergeRuntimeGlobals merges static globals with runtime globals and REPL declarations.
func (s *Service) MergeRuntimeGlobals(req MergeRuntimeGlobalsRequest, rt *inspectorruntime.Session) (MergeRuntimeGlobalsResponse, error) {
	doc, err := s.getDocument(req.DocumentID)
	if err != nil {
		return MergeRuntimeGlobalsResponse{}, err
	}

	existing := req.Existing
	if len(existing) == 0 {
		existing = mapAnalysisGlobals(doc.session.Globals())
	}
	if rt == nil {
		return MergeRuntimeGlobalsResponse{Globals: existing}, nil
	}

	runtimeKinds := inspectorruntime.RuntimeGlobalKinds(rt.VM)
	hasRuntimeValue := func(name string) bool {
		val := rt.GlobalValue(name)
		return val != nil && !goja.IsUndefined(val)
	}

	merged := inspectoranalysis.MergeGlobals(
		toAnalysisGlobals(existing),
		runtimeKinds,
		toAnalysisDeclaredBindings(req.Declared),
		hasRuntimeValue,
	)
	return MergeRuntimeGlobalsResponse{Globals: mapAnalysisGlobals(merged)}, nil
}

// BuildTreeRows returns current tree rows for a document index.
func (s *Service) BuildTreeRows(req BuildTreeRowsRequest) (BuildTreeRowsResponse, error) {
	doc, err := s.getDocument(req.DocumentID)
	if err != nil {
		return BuildTreeRowsResponse{}, err
	}
	if doc.analysis == nil || doc.analysis.Index == nil {
		return BuildTreeRowsResponse{}, nil
	}
	rows := inspectortree.BuildRowsFromIndex(doc.analysis.Index, req.UsageHighlights)
	return BuildTreeRowsResponse{Rows: rows}, nil
}

// SyncSourceToTree maps source cursor (0-based line/col) to tree selection.
func (s *Service) SyncSourceToTree(req SyncSourceToTreeRequest) (SyncSourceToTreeResponse, error) {
	doc, err := s.getDocument(req.DocumentID)
	if err != nil {
		return SyncSourceToTreeResponse{}, err
	}
	if doc.analysis == nil || doc.analysis.Index == nil {
		return SyncSourceToTreeResponse{}, nil
	}
	selection, ok := inspectornavigation.SelectionAtSourceCursor(
		doc.analysis.Index,
		doc.lines,
		req.CursorLine,
		req.CursorCol,
	)
	if !ok {
		return SyncSourceToTreeResponse{}, nil
	}

	doc.analysis.Index.ExpandTo(selection.NodeID)
	visible := doc.analysis.Index.VisibleNodes()
	visibleIdx := inspectornavigation.FindVisibleNodeIndex(visible, selection.NodeID)

	return SyncSourceToTreeResponse{
		Selection:    selection,
		VisibleIndex: visibleIdx,
		Found:        true,
	}, nil
}

// SyncTreeToSource maps selected tree row index to source cursor/highlight.
func (s *Service) SyncTreeToSource(req SyncTreeToSourceRequest) (SyncTreeToSourceResponse, error) {
	doc, err := s.getDocument(req.DocumentID)
	if err != nil {
		return SyncTreeToSourceResponse{}, err
	}
	if doc.analysis == nil || doc.analysis.Index == nil {
		return SyncTreeToSourceResponse{}, nil
	}
	visible := doc.analysis.Index.VisibleNodes()
	selection, ok := inspectornavigation.SelectionFromVisibleTree(doc.analysis.Index, visible, req.SelectedIndex)
	if !ok {
		return SyncTreeToSourceResponse{}, nil
	}
	return SyncTreeToSourceResponse{Selection: selection, Found: true}, nil
}

func (s *Service) getDocument(documentID DocumentID) (*documentSession, error) {
	s.mu.RLock()
	doc, ok := s.docs[documentID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrDocumentNotFound
	}
	return doc, nil
}

func mapAnalysisGlobals(globals []inspectoranalysis.GlobalBinding) []Global {
	out := make([]Global, 0, len(globals))
	for _, g := range globals {
		out = append(out, Global{Name: g.Name, Kind: g.Kind, Extends: g.Extends})
	}
	return out
}

func toAnalysisGlobals(globals []Global) []inspectoranalysis.GlobalBinding {
	out := make([]inspectoranalysis.GlobalBinding, 0, len(globals))
	for _, g := range globals {
		out = append(out, inspectoranalysis.GlobalBinding{Name: g.Name, Kind: g.Kind, Extends: g.Extends})
	}
	return out
}

func toAnalysisDeclaredBindings(declared []DeclaredBinding) []inspectoranalysis.DeclaredBinding {
	out := make([]inspectoranalysis.DeclaredBinding, 0, len(declared))
	for _, d := range declared {
		out = append(out, inspectoranalysis.DeclaredBinding{Name: d.Name, Kind: d.Kind})
	}
	return out
}

func mapClassMembers(members []inspectorcore.Member) []Member {
	out := make([]Member, 0, len(members))
	for _, m := range members {
		out = append(out, Member{
			Name:      m.Name,
			Kind:      m.Kind,
			Preview:   m.Preview,
			Inherited: m.Inherited,
			Source:    m.Source,
		})
	}
	return out
}

func mapFunctionMembers(members []inspectorcore.Member) []Member {
	out := make([]Member, 0, len(members))
	for _, m := range members {
		out = append(out, Member{Name: m.Name, Kind: m.Kind, Preview: m.Preview})
	}
	return out
}

func findGlobalByName(globals []inspectoranalysis.GlobalBinding, name string) (inspectoranalysis.GlobalBinding, bool) {
	for _, g := range globals {
		if g.Name == name {
			return g, true
		}
	}
	return inspectoranalysis.GlobalBinding{}, false
}

func runtimeValueMembers(rt *inspectorruntime.Session, name string) []Member {
	val := rt.GlobalValue(name)
	if val == nil || goja.IsUndefined(val) {
		return []Member{{Name: "(value)", Kind: "value", Preview: " : undefined", RuntimeDerived: true}}
	}
	if goja.IsNull(val) {
		return []Member{{Name: "(value)", Kind: "value", Preview: " : null", RuntimeDerived: true}}
	}
	if obj, ok := val.(*goja.Object); ok {
		props := inspectorruntime.InspectObject(obj, rt.VM)
		out := make([]Member, 0, len(props))
		for _, p := range props {
			out = append(out, Member{
				Name:           p.Name,
				Kind:           p.Kind,
				Preview:        " : " + p.Preview,
				RuntimeDerived: true,
			})
		}
		return out
	}
	return []Member{{
		Name:           "(value)",
		Kind:           "value",
		Preview:        " : " + inspectorruntime.ValuePreview(val, rt.VM, 40),
		RuntimeDerived: true,
	}}
}
