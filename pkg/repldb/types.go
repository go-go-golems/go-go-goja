package repldb

import (
	"encoding/json"
	"time"
)

// SessionRecord is the durable representation of a REPL session.
type SessionRecord struct {
	SessionID    string          `json:"sessionId"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	DeletedAt    *time.Time      `json:"deletedAt,omitempty"`
	EngineKind   string          `json:"engineKind"`
	MetadataJSON json.RawMessage `json:"metadataJson"`
}

// SessionExport is the structured export/readback payload for one session.
type SessionExport struct {
	Session     SessionRecord      `json:"session"`
	Evaluations []EvaluationRecord `json:"evaluations"`
}

// EvaluationRecord is the durable representation of one evaluated cell.
type EvaluationRecord struct {
	EvaluationID      int64             `json:"evaluationId"`
	SessionID         string            `json:"sessionId"`
	CellID            int               `json:"cellId"`
	CreatedAt         time.Time         `json:"createdAt"`
	RawSource         string            `json:"rawSource"`
	RewrittenSource   string            `json:"rewrittenSource"`
	OK                bool              `json:"ok"`
	ResultJSON        json.RawMessage   `json:"resultJson"`
	ErrorText         string            `json:"errorText"`
	AnalysisJSON      json.RawMessage   `json:"analysisJson"`
	GlobalsBeforeJSON json.RawMessage   `json:"globalsBeforeJson"`
	GlobalsAfterJSON  json.RawMessage   `json:"globalsAfterJson"`
	ConsoleEvents     []ConsoleEventRecord   `json:"consoleEvents"`
	BindingVersions   []BindingVersionRecord `json:"bindingVersions"`
	BindingDocs       []BindingDocRecord     `json:"bindingDocs"`
}

// ConsoleEventRecord is the durable representation of a console emission.
type ConsoleEventRecord struct {
	Stream string `json:"stream"`
	Seq    int    `json:"seq"`
	Text   string `json:"text"`
}

// BindingVersionRecord stores one version of a binding at a given evaluation.
type BindingVersionRecord struct {
	Name         string          `json:"name"`
	CreatedAt    time.Time       `json:"createdAt"`
	CellID       int             `json:"cellId"`
	Action       string          `json:"action"`
	RuntimeType  string          `json:"runtimeType"`
	DisplayValue string          `json:"displayValue"`
	SummaryJSON  json.RawMessage `json:"summaryJson"`
	ExportKind   string          `json:"exportKind"`
	ExportJSON   json.RawMessage `json:"exportJson"`
	DocDigest    string          `json:"docDigest"`
}

// BindingDocRecord stores REPL-authored documentation associated with a binding.
type BindingDocRecord struct {
	SymbolName     string          `json:"symbolName"`
	CellID         int             `json:"cellId"`
	SourceKind     string          `json:"sourceKind"`
	RawDoc         string          `json:"rawDoc"`
	NormalizedJSON json.RawMessage `json:"normalizedJson"`
}
