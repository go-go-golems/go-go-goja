package repldb

import (
	"encoding/json"
	"time"
)

// SessionRecord is the durable representation of a REPL session.
type SessionRecord struct {
	SessionID    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	EngineKind   string
	MetadataJSON json.RawMessage
}

// SessionExport is the structured export/readback payload for one session.
type SessionExport struct {
	Session     SessionRecord
	Evaluations []EvaluationRecord
}

// EvaluationRecord is the durable representation of one evaluated cell.
type EvaluationRecord struct {
	EvaluationID      int64
	SessionID         string
	CellID            int
	CreatedAt         time.Time
	RawSource         string
	RewrittenSource   string
	OK                bool
	ResultJSON        json.RawMessage
	ErrorText         string
	AnalysisJSON      json.RawMessage
	GlobalsBeforeJSON json.RawMessage
	GlobalsAfterJSON  json.RawMessage
	ConsoleEvents     []ConsoleEventRecord
	BindingVersions   []BindingVersionRecord
	BindingDocs       []BindingDocRecord
}

// ConsoleEventRecord is the durable representation of a console emission.
type ConsoleEventRecord struct {
	Stream string
	Seq    int
	Text   string
}

// BindingVersionRecord stores one version of a binding at a given evaluation.
type BindingVersionRecord struct {
	Name         string
	CreatedAt    time.Time
	CellID       int
	Action       string
	RuntimeType  string
	DisplayValue string
	SummaryJSON  json.RawMessage
	ExportKind   string
	ExportJSON   json.RawMessage
	DocDigest    string
}

// BindingDocRecord stores REPL-authored documentation associated with a binding.
type BindingDocRecord struct {
	SymbolName     string
	CellID         int
	SourceKind     string
	RawDoc         string
	NormalizedJSON json.RawMessage
}
