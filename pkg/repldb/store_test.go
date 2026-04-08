package repldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenBootstrapsSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "repl.sqlite")

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	for _, tableName := range []string{
		"repldb_meta",
		"sessions",
		"evaluations",
		"console_events",
		"bindings",
		"binding_versions",
		"binding_docs",
	} {
		if !tableExists(t, store.DB(), tableName) {
			t.Fatalf("expected table %q to exist", tableName)
		}
	}

	var version string
	err = store.DB().QueryRowContext(ctx, `SELECT value FROM repldb_meta WHERE key = 'schema_version'`).Scan(&version)
	if err != nil {
		t.Fatalf("query schema version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("expected schema version %q, got %q", currentSchemaVersion, version)
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	t.Parallel()

	store, err := Open(context.Background(), "")
	if err == nil {
		_ = store.Close()
		t.Fatal("expected error for empty path")
	}
}

func TestCreateSessionAndPersistEvaluation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	createdAt := time.Date(2026, 4, 3, 18, 10, 0, 0, time.UTC)
	if err := store.CreateSession(ctx, SessionRecord{
		SessionID:    "session-1",
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		EngineKind:   "goja",
		MetadataJSON: json.RawMessage(`{"transport":"test"}`),
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := store.PersistEvaluation(ctx, EvaluationRecord{
		SessionID:         "session-1",
		CellID:            1,
		CreatedAt:         createdAt.Add(time.Second),
		RawSource:         "const x = 1",
		RewrittenSource:   "(async()=>{return 1})()",
		OK:                true,
		ResultJSON:        json.RawMessage(`{"status":"ok"}`),
		AnalysisJSON:      json.RawMessage(`{"bindings":["x"]}`),
		GlobalsBeforeJSON: json.RawMessage(`[]`),
		GlobalsAfterJSON:  json.RawMessage(`[{"name":"x"}]`),
		ConsoleEvents: []ConsoleEventRecord{
			{Stream: "log", Seq: 1, Text: "hello"},
		},
	}); err != nil {
		t.Fatalf("persist evaluation: %v", err)
	}

	var (
		engineKind   string
		metadataJSON string
		updatedAt    string
	)
	if err := store.DB().QueryRowContext(
		ctx,
		`SELECT engine_kind, metadata_json, updated_at FROM sessions WHERE session_id = ?`,
		"session-1",
	).Scan(&engineKind, &metadataJSON, &updatedAt); err != nil {
		t.Fatalf("query session row: %v", err)
	}
	if engineKind != "goja" {
		t.Fatalf("expected engine kind goja, got %q", engineKind)
	}
	if metadataJSON != `{"transport":"test"}` {
		t.Fatalf("unexpected metadata json: %q", metadataJSON)
	}
	if updatedAt == "" {
		t.Fatal("expected updated_at to be set")
	}

	var (
		rawSource string
		okValue   int
	)
	if err := store.DB().QueryRowContext(
		ctx,
		`SELECT raw_source, ok FROM evaluations WHERE session_id = ? AND cell_id = ?`,
		"session-1",
		1,
	).Scan(&rawSource, &okValue); err != nil {
		t.Fatalf("query evaluation row: %v", err)
	}
	if rawSource != "const x = 1" {
		t.Fatalf("unexpected raw source: %q", rawSource)
	}
	if okValue != 1 {
		t.Fatalf("expected ok flag 1, got %d", okValue)
	}

	var consoleCount int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM console_events`).Scan(&consoleCount); err != nil {
		t.Fatalf("count console events: %v", err)
	}
	if consoleCount != 1 {
		t.Fatalf("expected 1 console event, got %d", consoleCount)
	}
}

func TestExportSessionAndReplaySource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	createdAt := time.Date(2026, 4, 3, 18, 12, 0, 0, time.UTC)
	if err := store.CreateSession(ctx, SessionRecord{
		SessionID:  "session-export",
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
		EngineKind: "goja",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := store.PersistEvaluation(ctx, EvaluationRecord{
		SessionID:         "session-export",
		CellID:            1,
		CreatedAt:         createdAt.Add(time.Second),
		RawSource:         "const x = {answer: 42}",
		RewrittenSource:   "(async()=>({x:{answer:42}}))()",
		OK:                true,
		ResultJSON:        json.RawMessage(`{"status":"ok"}`),
		AnalysisJSON:      json.RawMessage(`{"bindings":["x"]}`),
		GlobalsBeforeJSON: json.RawMessage(`[]`),
		GlobalsAfterJSON:  json.RawMessage(`[{"name":"x"}]`),
		BindingVersions: []BindingVersionRecord{
			{
				Name:         "x",
				CreatedAt:    createdAt.Add(time.Second),
				CellID:       1,
				Action:       "insert",
				RuntimeType:  "object",
				DisplayValue: "{answer: 42}",
				SummaryJSON:  json.RawMessage(`{"name":"x"}`),
				ExportKind:   "json",
				ExportJSON:   json.RawMessage(`{"answer":42}`),
				DocDigest:    "abc123",
			},
		},
		BindingDocs: []BindingDocRecord{
			{
				SymbolName:     "x",
				CellID:         1,
				SourceKind:     "jsdocex",
				RawDoc:         `{"name":"x","summary":"number x"}`,
				NormalizedJSON: json.RawMessage(`{"name":"x","summary":"number x"}`),
			},
		},
	}); err != nil {
		t.Fatalf("persist evaluation: %v", err)
	}

	exported, err := store.ExportSession(ctx, "session-export")
	if err != nil {
		t.Fatalf("export session: %v", err)
	}
	if exported.Session.SessionID != "session-export" {
		t.Fatalf("unexpected session id: %q", exported.Session.SessionID)
	}
	if len(exported.Evaluations) != 1 {
		t.Fatalf("expected 1 evaluation, got %d", len(exported.Evaluations))
	}
	if len(exported.Evaluations[0].BindingVersions) != 1 {
		t.Fatalf("expected 1 binding version, got %d", len(exported.Evaluations[0].BindingVersions))
	}
	if len(exported.Evaluations[0].BindingDocs) != 1 {
		t.Fatalf("expected 1 binding doc, got %d", len(exported.Evaluations[0].BindingDocs))
	}

	replaySource, err := store.LoadReplaySource(ctx, "session-export")
	if err != nil {
		t.Fatalf("load replay source: %v", err)
	}
	if len(replaySource) != 1 || replaySource[0] != "const x = {answer: 42}" {
		t.Fatalf("unexpected replay source: %#v", replaySource)
	}
}

func TestDeletedSessionsAreHiddenFromNormalReads(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	createdAt := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	sessionID := "session-deleted"
	if err := store.CreateSession(ctx, SessionRecord{
		SessionID:  sessionID,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
		EngineKind: "goja",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := store.PersistEvaluation(ctx, EvaluationRecord{
		SessionID:         sessionID,
		CellID:            1,
		CreatedAt:         createdAt.Add(time.Second),
		RawSource:         "1 + 1",
		RewrittenSource:   "1 + 1",
		OK:                true,
		ResultJSON:        json.RawMessage(`{"result":"2"}`),
		AnalysisJSON:      json.RawMessage(`{}`),
		GlobalsBeforeJSON: json.RawMessage(`[]`),
		GlobalsAfterJSON:  json.RawMessage(`[]`),
	}); err != nil {
		t.Fatalf("persist evaluation: %v", err)
	}
	if err := store.DeleteSession(ctx, sessionID, createdAt.Add(2*time.Second)); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected deleted session to be hidden from list, got %d rows", len(sessions))
	}

	if _, err := store.LoadSession(ctx, sessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected load session to return ErrSessionNotFound, got %v", err)
	}
	if _, err := store.LoadEvaluations(ctx, sessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected load evaluations to return ErrSessionNotFound, got %v", err)
	}
	if _, err := store.ExportSession(ctx, sessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected export session to return ErrSessionNotFound, got %v", err)
	}
	if _, err := store.LoadReplaySource(ctx, sessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected replay source to return ErrSessionNotFound, got %v", err)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "repl.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func tableExists(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()

	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&name)
	switch err {
	case nil:
		return name == tableName
	case sql.ErrNoRows:
		return false
	default:
		t.Fatalf("query sqlite_master for %q: %v", tableName, err)
		return false
	}
}
