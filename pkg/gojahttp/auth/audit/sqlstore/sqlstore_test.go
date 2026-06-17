package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/audittest"
)

func TestSQLiteStoreContract(t *testing.T) {
	audittest.RunStoreContract(t, func(t testing.TB) audittest.Harness {
		store := newSQLiteStore(t)
		return audittest.Harness{
			Store: store,
			Snapshot: func() []audit.Record {
				t.Helper()
				records, err := store.Snapshot(context.Background())
				if err != nil {
					t.Fatalf("snapshot: %v", err)
				}
				return records
			},
		}
	})
}

func TestSinkRedactsBeforeInsert(t *testing.T) {
	store := newSQLiteStore(t)
	now := time.Date(2026, 6, 12, 15, 0, 0, 0, time.UTC)
	sink := audit.Sink{Store: store, Normalizer: audit.Normalizer{Now: func() time.Time { return now }}}
	if err := sink.RecordAudit(context.Background(), gojahttp.AuditEvent{
		Event:   "capability.issued",
		Outcome: "completed",
		Attributes: map[string]any{
			"rawToken":   "secret-token",
			"sessionID":  "secret-session",
			"nested":     map[string]any{"authorization": "Bearer secret", "safe": "kept"},
			"capability": "secret-capability",
		},
	}); err != nil {
		t.Fatalf("record audit: %v", err)
	}
	records, err := store.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(records) != 1 || records[0].CreatedAt.IsZero() || !records[0].CreatedAt.Equal(now) {
		t.Fatalf("unexpected records: %#v", records)
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"secret-token", "secret-session", "Bearer secret", "secret-capability"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("stored audit record leaked %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "[REDACTED]") || !strings.Contains(text, "kept") {
		t.Fatalf("stored audit record missing redaction/kept value: %s", text)
	}
}

func TestQueryByOutcome(t *testing.T) {
	store := newSQLiteStore(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 15, 0, 0, 0, time.UTC)
	for _, record := range []audit.Record{
		{Event: "first denied", Outcome: "denied", Reason: "missing role", ActorID: "u1", CreatedAt: now},
		{Event: "completed", Outcome: "completed", ActorID: "u2", CreatedAt: now.Add(time.Second)},
		{Event: "second denied", Outcome: "denied", Reason: "missing csrf", ActorID: "u3", CreatedAt: now.Add(2 * time.Second)},
	} {
		if err := store.InsertAuditRecord(ctx, record); err != nil {
			t.Fatalf("insert %s: %v", record.Event, err)
		}
	}
	denied, err := store.QueryByOutcome(ctx, "denied", 10)
	if err != nil {
		t.Fatalf("query denied: %v", err)
	}
	if len(denied) != 2 || denied[0].Event != "second denied" || denied[1].Event != "first denied" {
		t.Fatalf("unexpected denied records: %#v", denied)
	}
	limited, err := store.QueryByOutcome(ctx, "denied", 1)
	if err != nil {
		t.Fatalf("query limited denied: %v", err)
	}
	if len(limited) != 1 || limited[0].Event != "second denied" {
		t.Fatalf("unexpected limited denied records: %#v", limited)
	}
}

func TestQueryAuditRecordsFiltersBoundsAndOrders(t *testing.T) {
	store := newSQLiteStore(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	for _, record := range []audit.Record{
		{Event: "old denied", Outcome: "denied", TenantID: "o1", ActorID: "u1", ResourceType: "project", ResourceID: "p1", CreatedAt: now},
		{Event: "other tenant", Outcome: "denied", TenantID: "o2", ActorID: "u2", ResourceType: "project", ResourceID: "p2", CreatedAt: now.Add(time.Second)},
		{Event: "new denied", Outcome: "denied", TenantID: "o1", ActorID: "u3", ResourceType: "project", ResourceID: "p3", CreatedAt: now.Add(2 * time.Second)},
		{Event: "allowed", Outcome: "allowed", TenantID: "o1", ActorID: "u4", ResourceType: "project", ResourceID: "p4", CreatedAt: now.Add(3 * time.Second)},
	} {
		if err := store.InsertAuditRecord(ctx, record); err != nil {
			t.Fatalf("insert %s: %v", record.Event, err)
		}
	}

	records, err := store.QueryAuditRecords(ctx, audit.Query{TenantID: "o1", Outcome: "denied", Limit: 10})
	if err != nil {
		t.Fatalf("query records: %v", err)
	}
	if len(records) != 2 || records[0].Event != "new denied" || records[1].Event != "old denied" {
		t.Fatalf("unexpected filtered records: %#v", records)
	}

	limited, err := store.QueryAuditRecords(ctx, audit.Query{TenantID: "o1", Limit: 1})
	if err != nil {
		t.Fatalf("query limited records: %v", err)
	}
	if len(limited) != 1 || limited[0].Event != "allowed" {
		t.Fatalf("unexpected limited records: %#v", limited)
	}
}

func TestPostgresSchemaAndPlaceholders(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := New(Config{DB: db, Dialect: DialectPostgres})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if got := store.queryByOutcomeQuery(); !strings.Contains(got, "$1") || !strings.Contains(got, "$2") {
		t.Fatalf("unexpected postgres query by outcome: %s", got)
	}
	query, args := store.queryAuditRecordsQuery(audit.NormalizeQuery(audit.Query{TenantID: "o1", Outcome: "denied", Limit: 10}, audit.MaxQueryLimit))
	for _, placeholder := range []string{"$1", "$2", "$3", "$4"} {
		if !strings.Contains(query, placeholder) {
			t.Fatalf("postgres query missing %s: %s", placeholder, query)
		}
	}
	if len(args) != 4 {
		t.Fatalf("postgres query args len=%d, want 4", len(args))
	}
	if !strings.Contains(store.Schema(), "JSONB") || !strings.Contains(store.Schema(), "TIMESTAMPTZ") || !strings.Contains(store.Schema(), "BIGSERIAL") {
		t.Fatalf("postgres schema missing expected types: %s", store.Schema())
	}
}

func newSQLiteStore(t testing.TB) *Store {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, err := New(Config{DB: db, Dialect: DialectSQLite})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.ApplySchema(context.Background()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return store
}
