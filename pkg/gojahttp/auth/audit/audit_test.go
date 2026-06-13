package audit

import (
	"bytes"
	"context"
	"encoding/json"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestNormalizeRecordAndRedaction(t *testing.T) {
	now := time.Date(2026, 6, 12, 18, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodPatch, "/orgs/o1/projects/p1", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Request-Id", "req-1")
	req.Header.Set("User-Agent", "agent-test")
	normalizer := Normalizer{Now: func() time.Time { return now }, IPHash: func(ip string) string { return "hash:" + ip }}
	record := normalizer.Normalize(gojahttp.AuditEvent{
		HTTPRequest: req,
		Event:       "project.updated",
		Outcome:     "completed",
		StatusCode:  http.StatusOK,
		RouteName:   "project.update",
		Method:      http.MethodPatch,
		Pattern:     "/orgs/:orgId/projects/:projectId",
		Action:      "project.update",
		Actor:       &gojahttp.Actor{ID: "u1", Kind: "user"},
		Resource:    &gojahttp.ResourceRef{Type: "project", ID: "p1", TenantID: "o1"},
		Attributes: map[string]any{
			"safe":          "ok",
			"sessionID":     "secret-session",
			"capability":    "secret-capability",
			"nested":        map[string]any{"accessToken": "secret-token", "value": "kept"},
			"authorization": "Bearer secret",
		},
	})
	if record.Event != "project.updated" || record.ActorID != "u1" || record.ResourceID != "p1" || record.TenantID != "o1" {
		t.Fatalf("unexpected record: %#v", record)
	}
	if record.RequestID != "req-1" || record.UserAgent != "agent-test" || record.IPHash != "hash:203.0.113.10" || !record.CreatedAt.Equal(now) {
		t.Fatalf("unexpected request metadata: %#v", record)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"secret-session", "secret-capability", "secret-token", "Bearer secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("record leaked %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "[REDACTED]") || !strings.Contains(text, "kept") {
		t.Fatalf("record missing redaction/kept value: %s", text)
	}
}

func TestMemorySinkAndStoreSink(t *testing.T) {
	ctx := context.Background()
	memory := &MemorySink{Normalizer: Normalizer{Now: func() time.Time { return time.Unix(1, 0) }}}
	if err := memory.RecordAudit(ctx, gojahttp.AuditEvent{Event: "demo", Outcome: "completed"}); err != nil {
		t.Fatalf("memory sink: %v", err)
	}
	if got := memory.Snapshot(); len(got) != 1 || got[0].Event != "demo" {
		t.Fatalf("unexpected memory records: %#v", got)
	}
	store := &recordingStore{}
	sink := Sink{Store: store}
	if err := sink.RecordAudit(ctx, gojahttp.AuditEvent{Event: "stored", Outcome: "allowed"}); err != nil {
		t.Fatalf("store sink: %v", err)
	}
	if len(store.records) != 1 || store.records[0].Event != "stored" {
		t.Fatalf("unexpected store records: %#v", store.records)
	}
}

func TestLogSinkOmitsSensitiveRequestMetadata(t *testing.T) {
	var buf bytes.Buffer
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "sensitive-request-id")
	req.Header.Set("User-Agent", "sensitive-user-agent")
	sink := LogSink{Logger: stdlog.New(&buf, "", 0)}
	if err := sink.RecordAudit(context.Background(), gojahttp.AuditEvent{HTTPRequest: req, Event: "cap", Outcome: "issued", Reason: "sensitive-reason", Attributes: map[string]any{"rawToken": "secret"}}); err != nil {
		t.Fatalf("log sink: %v", err)
	}
	output := buf.String()
	for _, forbidden := range []string{"secret", "sensitive-request-id", "sensitive-user-agent", "sensitive-reason"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("log output leaked %q: %s", forbidden, output)
		}
	}
	if !strings.Contains(output, `"event":"cap"`) || !strings.Contains(output, `"outcome":"issued"`) {
		t.Fatalf("unexpected log output: %s", output)
	}
}

type recordingStore struct{ records []Record }

func (s *recordingStore) InsertAuditRecord(_ context.Context, record Record) error {
	s.records = append(s.records, record)
	return nil
}
