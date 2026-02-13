package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadInputNoArgsStartsEmptyScratchBuffer(t *testing.T) {
	filename, src, err := loadInput(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if filename != defaultFilename {
		t.Fatalf("expected default filename %q, got %q", defaultFilename, filename)
	}
	if src != "" {
		t.Fatalf("expected empty source, got %q", src)
	}
}

func TestLoadInputReadsExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.js")
	want := "const x = 1;"
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	filename, src, err := loadInput([]string{path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if filename != path {
		t.Fatalf("expected filename %q, got %q", path, filename)
	}
	if src != want {
		t.Fatalf("expected source %q, got %q", want, src)
	}
}

func TestLoadInputMissingFileStartsEmptyBuffer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new-file.js")

	filename, src, err := loadInput([]string{path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if filename != path {
		t.Fatalf("expected filename %q, got %q", path, filename)
	}
	if src != "" {
		t.Fatalf("expected empty source, got %q", src)
	}
}

func TestLoadInputRejectsMultipleArgs(t *testing.T) {
	_, _, err := loadInput([]string{"a.js", "b.js"})
	if err == nil {
		t.Fatal("expected error for multiple args")
	}
	if !strings.Contains(err.Error(), "usage: ast-parse-editor [file.js]") {
		t.Fatalf("expected usage error, got %v", err)
	}
}
