package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestCLICommandFlow(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "repl.sqlite")

	createOut := &bytes.Buffer{}
	createRoot, err := newRootCommand(createOut)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	createRoot.SetArgs([]string{"--db-path", dbPath, "create"})
	if err := createRoot.Execute(); err != nil {
		t.Fatalf("execute create: %v", err)
	}

	var createPayload struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create output: %v", err)
	}
	if createPayload.Session.ID == "" {
		t.Fatal("expected session id from create command")
	}

	evalOut := &bytes.Buffer{}
	evalRoot, err := newRootCommand(evalOut)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	evalRoot.SetArgs([]string{"--db-path", dbPath, "eval", "--session-id", createPayload.Session.ID, "--source", "const x = 1; x"})
	if err := evalRoot.Execute(); err != nil {
		t.Fatalf("execute eval: %v", err)
	}

	historyOut := &bytes.Buffer{}
	historyRoot, err := newRootCommand(historyOut)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	historyRoot.SetArgs([]string{"--db-path", dbPath, "history", "--session-id", createPayload.Session.ID})
	if err := historyRoot.Execute(); err != nil {
		t.Fatalf("execute history: %v", err)
	}

	var historyPayload struct {
		History []any `json:"history"`
	}
	if err := json.Unmarshal(historyOut.Bytes(), &historyPayload); err != nil {
		t.Fatalf("decode history output: %v", err)
	}
	if len(historyPayload.History) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(historyPayload.History))
	}
}
