package replsession

import "time"

// SessionSummary is the top-level state returned to the web UI.
type SessionSummary struct {
	ID             string             `json:"id"`
	CreatedAt      time.Time          `json:"createdAt"`
	CellCount      int                `json:"cellCount"`
	BindingCount   int                `json:"bindingCount"`
	Bindings       []BindingView      `json:"bindings"`
	History        []HistoryEntry     `json:"history"`
	CurrentGlobals []GlobalStateView  `json:"currentGlobals"`
	Provenance     []ProvenanceRecord `json:"provenance"`
}

// EvaluateRequest is the JSON payload for evaluating one REPL cell.
type EvaluateRequest struct {
	Source string `json:"source"`
}

// EvaluateResponse is the JSON payload returned after one evaluation.
type EvaluateResponse struct {
	Session *SessionSummary `json:"session"`
	Cell    *CellReport     `json:"cell"`
}

// CellReport captures one evaluation pipeline end to end.
type CellReport struct {
	ID         int                `json:"id"`
	CreatedAt  time.Time          `json:"createdAt"`
	Source     string             `json:"source"`
	Static     StaticReport       `json:"static"`
	Rewrite    RewriteReport      `json:"rewrite"`
	Execution  ExecutionReport    `json:"execution"`
	Runtime    RuntimeReport      `json:"runtime"`
	Provenance []ProvenanceRecord `json:"provenance"`
}

// ExecutionReport describes the actual runtime evaluation outcome.
type ExecutionReport struct {
	Status      string         `json:"status"`
	Result      string         `json:"result"`
	Error       string         `json:"error,omitempty"`
	DurationMS  int64          `json:"durationMs"`
	Awaited     bool           `json:"awaited"`
	Console     []ConsoleEvent `json:"console"`
	HadSideFX   bool           `json:"hadSideEffects"`
	HelperError bool           `json:"helperError"`
}

// ConsoleEvent captures one console.* emission.
type ConsoleEvent struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// StaticReport contains parser- and index-derived information.
type StaticReport struct {
	Diagnostics      []DiagnosticView        `json:"diagnostics"`
	TopLevelBindings []TopLevelBindingView   `json:"topLevelBindings"`
	Unresolved       []IdentifierUseView     `json:"unresolved"`
	References       []BindingReferenceGroup `json:"references"`
	Scope            *ScopeView              `json:"scope,omitempty"`
	AST              []ASTRowView            `json:"ast"`
	ASTNodeCount     int                     `json:"astNodeCount"`
	ASTTruncated     bool                    `json:"astTruncated"`
	CST              []CSTNodeView           `json:"cst"`
	CSTNodeCount     int                     `json:"cstNodeCount"`
	CSTTruncated     bool                    `json:"cstTruncated"`
	FinalExpression  *RangeView              `json:"finalExpression,omitempty"`
	Summary          []StaticSummaryFact     `json:"summary"`
}

// StaticSummaryFact is a compact text fact shown in the UI before the raw JSON.
type StaticSummaryFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// RewriteReport shows how the submitted cell was transformed before execution.
type RewriteReport struct {
	Mode               string        `json:"mode"`
	DeclaredNames      []string      `json:"declaredNames"`
	HelperNames        []string      `json:"helperNames"`
	LastHelperName     string        `json:"lastHelperName"`
	BindingHelperName  string        `json:"bindingHelperName"`
	CapturedLastExpr   bool          `json:"capturedLastExpr"`
	TransformedSource  string        `json:"transformedSource"`
	Operations         []RewriteStep `json:"operations"`
	Warnings           []string      `json:"warnings,omitempty"`
	FinalExpressionSrc string        `json:"finalExpressionSource,omitempty"`
}

// RewriteStep is one explicit transformation step.
type RewriteStep struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// RuntimeReport contains runtime diffs and changed symbol information.
type RuntimeReport struct {
	BeforeGlobals    []GlobalStateView `json:"beforeGlobals"`
	AfterGlobals     []GlobalStateView `json:"afterGlobals"`
	Diffs            []GlobalDiffView  `json:"diffs"`
	NewBindings      []string          `json:"newBindings"`
	UpdatedBindings  []string          `json:"updatedBindings"`
	RemovedBindings  []string          `json:"removedBindings"`
	LeakedGlobals    []string          `json:"leakedGlobals"`
	PersistedByWrap  []string          `json:"persistedByWrap"`
	CurrentCellValue string            `json:"currentCellValue"`
}

// ProvenanceRecord shows how a section was obtained.
type ProvenanceRecord struct {
	Section string   `json:"section"`
	Source  string   `json:"source"`
	Notes   []string `json:"notes,omitempty"`
}

// HistoryEntry is the compact per-cell history shown in the session panel.
type HistoryEntry struct {
	CellID        int       `json:"cellId"`
	CreatedAt     time.Time `json:"createdAt"`
	SourcePreview string    `json:"sourcePreview"`
	ResultPreview string    `json:"resultPreview"`
	Status        string    `json:"status"`
}

// BindingView is the current session view of one named binding.
type BindingView struct {
	Name            string             `json:"name"`
	Kind            string             `json:"kind"`
	Origin          string             `json:"origin"`
	DeclaredInCell  int                `json:"declaredInCell"`
	LastUpdatedCell int                `json:"lastUpdatedCell"`
	DeclaredLine    int                `json:"declaredLine,omitempty"`
	DeclaredSnippet string             `json:"declaredSnippet,omitempty"`
	Static          *BindingStaticView `json:"static,omitempty"`
	Runtime         BindingRuntimeView `json:"runtime"`
	Provenance      []ProvenanceRecord `json:"provenance,omitempty"`
}

// BindingStaticView stores parser-derived metadata for the binding.
type BindingStaticView struct {
	References []IdentifierUseView `json:"references,omitempty"`
	Parameters []string            `json:"parameters,omitempty"`
	Extends    string              `json:"extends,omitempty"`
	Members    []MemberView        `json:"members,omitempty"`
}

// BindingRuntimeView stores runtime-derived metadata for the binding.
type BindingRuntimeView struct {
	ValueKind       string               `json:"valueKind"`
	Preview         string               `json:"preview"`
	OwnProperties   []PropertyView       `json:"ownProperties,omitempty"`
	PrototypeChain  []PrototypeLevelView `json:"prototypeChain,omitempty"`
	FunctionMapping *FunctionMappingView `json:"functionMapping,omitempty"`
}

// PrototypeLevelView is one prototype chain frame.
type PrototypeLevelView struct {
	Name       string         `json:"name"`
	Properties []PropertyView `json:"properties,omitempty"`
}

// PropertyView is a JSON-safe property descriptor snapshot.
type PropertyView struct {
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	Preview    string          `json:"preview"`
	IsSymbol   bool            `json:"isSymbol"`
	Descriptor *DescriptorView `json:"descriptor,omitempty"`
}

// DescriptorView is a JSON-safe property descriptor.
type DescriptorView struct {
	Writable     bool `json:"writable"`
	Enumerable   bool `json:"enumerable"`
	Configurable bool `json:"configurable"`
	HasGetter    bool `json:"hasGetter"`
	HasSetter    bool `json:"hasSetter"`
}

// FunctionMappingView points from a runtime function back to source.
type FunctionMappingView struct {
	Name      string `json:"name"`
	ClassName string `json:"className,omitempty"`
	StartLine int    `json:"startLine"`
	StartCol  int    `json:"startCol"`
	EndLine   int    `json:"endLine"`
	EndCol    int    `json:"endCol"`
	NodeID    int    `json:"nodeId"`
}

// GlobalStateView is a compact snapshot of a global property.
type GlobalStateView struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Preview       string `json:"preview"`
	Identity      string `json:"identity"`
	PropertyCount int    `json:"propertyCount"`
}

// GlobalDiffView describes a before/after change to one global entry.
type GlobalDiffView struct {
	Name         string `json:"name"`
	Change       string `json:"change"`
	Before       string `json:"before,omitempty"`
	After        string `json:"after,omitempty"`
	BeforeKind   string `json:"beforeKind,omitempty"`
	AfterKind    string `json:"afterKind,omitempty"`
	SessionBound bool   `json:"sessionBound"`
}

// DiagnosticView is a JSON-safe diagnostic.
type DiagnosticView struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// TopLevelBindingView is one parser-derived declaration in the submitted cell.
type TopLevelBindingView struct {
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	Line           int    `json:"line"`
	Snippet        string `json:"snippet"`
	Extends        string `json:"extends,omitempty"`
	ReferenceCount int    `json:"referenceCount"`
}

// BindingReferenceGroup groups identifier usages by binding name.
type BindingReferenceGroup struct {
	Name      string              `json:"name"`
	Kind      string              `json:"kind"`
	Locations []IdentifierUseView `json:"locations"`
}

// IdentifierUseView points at an identifier usage in source.
type IdentifierUseView struct {
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Context string `json:"context,omitempty"`
	NodeID  int    `json:"nodeId"`
	Snippet string `json:"snippet,omitempty"`
}

// ScopeView is a recursive, UI-safe representation of the resolver scope tree.
type ScopeView struct {
	ID       int            `json:"id"`
	Kind     string         `json:"kind"`
	Start    int            `json:"start"`
	End      int            `json:"end"`
	Bindings []ScopeBinding `json:"bindings"`
	Children []*ScopeView   `json:"children,omitempty"`
}

// ScopeBinding is one binding entry within a scope.
type ScopeBinding struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// ASTRowView is one flattened AST row.
type ASTRowView struct {
	NodeID      int    `json:"nodeId"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// CSTNodeView is one flattened CST entry.
type CSTNodeView struct {
	Depth     int    `json:"depth"`
	Kind      string `json:"kind"`
	Text      string `json:"text,omitempty"`
	StartRow  int    `json:"startRow"`
	StartCol  int    `json:"startCol"`
	EndRow    int    `json:"endRow"`
	EndCol    int    `json:"endCol"`
	IsError   bool   `json:"isError"`
	IsMissing bool   `json:"isMissing"`
}

// RangeView points to a span in source.
type RangeView struct {
	StartLine int `json:"startLine"`
	StartCol  int `json:"startCol"`
	EndLine   int `json:"endLine"`
	EndCol    int `json:"endCol"`
}

// MemberView is a JSON-safe view of inspectorcore.Member.
type MemberView struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Preview   string `json:"preview,omitempty"`
	Inherited bool   `json:"inherited"`
	Source    string `json:"source,omitempty"`
}
