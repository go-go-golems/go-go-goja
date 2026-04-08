package replsession

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func TestServicePersistsSessionAndEvaluation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openPersistenceTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(store))
	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Evaluate(ctx, session.ID, "const x = 1;\nx"); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	var sessionCount int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE session_id = ?`, session.ID).Scan(&sessionCount); err != nil {
		t.Fatalf("count session rows: %v", err)
	}
	if sessionCount != 1 {
		t.Fatalf("expected 1 session row, got %d", sessionCount)
	}

	var evaluationCount int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM evaluations WHERE session_id = ?`, session.ID).Scan(&evaluationCount); err != nil {
		t.Fatalf("count evaluation rows: %v", err)
	}
	if evaluationCount != 1 {
		t.Fatalf("expected 1 evaluation row, got %d", evaluationCount)
	}
}

func TestServiceDeleteSessionPersistsDeletion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openPersistenceTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(store))
	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := service.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	var deletedAt string
	if err := store.DB().QueryRowContext(ctx, `SELECT deleted_at FROM sessions WHERE session_id = ?`, session.ID).Scan(&deletedAt); err != nil {
		t.Fatalf("query deleted_at: %v", err)
	}
	if deletedAt == "" {
		t.Fatal("expected deleted_at to be set")
	}
}

func TestServiceCreateSessionUsesCollisionResistantDefaultIDsAcrossServices(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openPersistenceTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	service1 := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(store))
	session1, err := service1.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create first session: %v", err)
	}

	service2 := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(store))
	session2, err := service2.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create second session: %v", err)
	}

	if session1.ID == "" || session2.ID == "" {
		t.Fatalf("expected non-empty generated ids, got %q and %q", session1.ID, session2.ID)
	}
	if session1.ID == session2.ID {
		t.Fatalf("expected distinct generated ids, both were %q", session1.ID)
	}
}

func TestServiceCreateSessionHonorsExplicitID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openPersistenceTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(store))
	session, err := service.CreateSessionWithOptions(ctx, SessionOptions{ID: "manual-session-id"})
	if err != nil {
		t.Fatalf("create session with explicit id: %v", err)
	}
	if session.ID != "manual-session-id" {
		t.Fatalf("expected explicit id to be preserved, got %q", session.ID)
	}
}

func TestServicePersistsBindingVersionsAndDocs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openPersistenceTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(store))
	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	source := "__doc__(\"x\", {summary:\"number x\", tags:[\"demo\"]});\ndoc`---\nsymbol: x\n---\nLong-form docs for x.\n`;\nconst x = {answer: 42};\nx"
	response, err := service.Evaluate(ctx, session.ID, source)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if response.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok execution status, got %q", response.Cell.Execution.Status)
	}

	var (
		bindingName string
		exportKind  string
		exportJSON  string
		docDigest   string
	)
	err = store.DB().QueryRowContext(
		ctx,
		`SELECT b.name, bv.export_kind, bv.export_json, bv.doc_digest
		 FROM binding_versions bv
		 JOIN bindings b ON b.binding_id = bv.binding_id
		 ORDER BY bv.binding_version_id ASC
		 LIMIT 1`,
	).Scan(&bindingName, &exportKind, &exportJSON, &docDigest)
	if err != nil {
		t.Fatalf("query binding version: %v", err)
	}
	if bindingName != "x" {
		t.Fatalf("expected binding version for x, got %q", bindingName)
	}
	if exportKind != "json" {
		t.Fatalf("expected export kind json, got %q", exportKind)
	}
	if exportJSON != `{"answer":42}` {
		t.Fatalf("unexpected export json: %q", exportJSON)
	}
	if docDigest == "" {
		t.Fatal("expected doc digest to be set")
	}

	var (
		symbolName string
		rawDoc     string
	)
	err = store.DB().QueryRowContext(
		ctx,
		`SELECT symbol_name, raw_doc
		 FROM binding_docs
		 ORDER BY binding_doc_id ASC
		 LIMIT 1`,
	).Scan(&symbolName, &rawDoc)
	if err != nil {
		t.Fatalf("query binding doc: %v", err)
	}
	if symbolName != "x" {
		t.Fatalf("expected doc symbol x, got %q", symbolName)
	}
	if rawDoc == "" {
		t.Fatal("expected raw_doc to be set")
	}
}

func newPersistenceTestFactory(t *testing.T) *engine.Factory {
	t.Helper()

	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	return factory
}

func openPersistenceTestStore(t *testing.T) *repldb.Store {
	t.Helper()

	store, err := repldb.Open(context.Background(), filepath.Join(t.TempDir(), "repl.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}
