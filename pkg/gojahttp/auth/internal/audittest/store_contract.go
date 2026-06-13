// Package audittest provides reusable conformance tests for audit.Store
// implementations.
package audittest

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
)

// Harness exposes an audit store plus a snapshot/query hook for contract tests.
type Harness struct {
	Store    audit.Store
	Snapshot func() []audit.Record
}

// NewHarness constructs an empty audit store harness for a single contract test.
type NewHarness func(testing.TB) Harness

// RunStoreContract verifies that audit stores persist records in order and do
// not expose mutable caller-owned maps through insert or query paths.
func RunStoreContract(t *testing.T, newHarness NewHarness) {
	t.Helper()

	t.Run("insert snapshot and clone isolation", func(t *testing.T) {
		h := requireHarness(t, newHarness(t))
		now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
		record := audit.Record{
			Event:        "project.updated",
			Outcome:      "completed",
			StatusCode:   200,
			RouteName:    "project.update",
			Method:       "PATCH",
			Pattern:      "/orgs/:orgId/projects/:projectId",
			Action:       "project.update",
			ActorID:      "u1",
			ActorKind:    "user",
			TenantID:     "o1",
			ResourceType: "project",
			ResourceID:   "p1",
			RequestID:    "req-1",
			IPHash:       "hash-ip",
			UserAgent:    "agent",
			Attributes:   map[string]any{"safe": "ok", "nested": map[string]any{"value": "kept"}},
			CreatedAt:    now,
		}
		if err := h.Store.InsertAuditRecord(context.Background(), record); err != nil {
			t.Fatalf("insert: %v", err)
		}
		record.Attributes["safe"] = "mutated"
		record.Attributes["nested"].(map[string]any)["value"] = "mutated"

		snapshot := h.Snapshot()
		if len(snapshot) != 1 {
			t.Fatalf("expected one record, got %#v", snapshot)
		}
		got := snapshot[0]
		if got.Event != "project.updated" || got.ActorID != "u1" || got.ResourceID != "p1" || !got.CreatedAt.Equal(now) {
			t.Fatalf("unexpected audit record: %#v", got)
		}
		if got.Attributes["safe"] != "ok" || got.Attributes["nested"].(map[string]any)["value"] != "kept" {
			t.Fatalf("record mutated through caller-owned input: %#v", got.Attributes)
		}

		snapshot[0].Attributes["safe"] = "changed-through-snapshot"
		again := h.Snapshot()
		if again[0].Attributes["safe"] != "ok" {
			t.Fatalf("record mutated through snapshot: %#v", again[0].Attributes)
		}
	})

	t.Run("multiple records preserve insertion order", func(t *testing.T) {
		h := requireHarness(t, newHarness(t))
		if err := h.Store.InsertAuditRecord(context.Background(), audit.Record{Event: "first"}); err != nil {
			t.Fatalf("insert first: %v", err)
		}
		if err := h.Store.InsertAuditRecord(context.Background(), audit.Record{Event: "second"}); err != nil {
			t.Fatalf("insert second: %v", err)
		}
		snapshot := h.Snapshot()
		if len(snapshot) != 2 || snapshot[0].Event != "first" || snapshot[1].Event != "second" {
			t.Fatalf("unexpected order: %#v", snapshot)
		}
	})
}

func requireHarness(t *testing.T, h Harness) Harness {
	t.Helper()
	if h.Store == nil || h.Snapshot == nil {
		t.Fatalf("incomplete audit store harness: %#v", h)
	}
	return h
}
