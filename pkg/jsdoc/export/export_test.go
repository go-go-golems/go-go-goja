package export

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
	"gopkg.in/yaml.v3"
)

func testStore() *model.DocStore {
	s := model.NewDocStore()
	s.AddFile(&model.FileDoc{
		FilePath: "test.js",
		Package: &model.Package{
			Name:        "pkg",
			Title:       "Package Title",
			Description: "Package desc",
			SourceFile:  "test.js",
		},
		Symbols: []*model.SymbolDoc{
			{
				Name:       "fn",
				Summary:    "Fn summary",
				Concepts:   []string{"c1"},
				Tags:       []string{"t1"},
				SourceFile: "test.js",
				Line:       12,
			},
		},
		Examples: []*model.Example{
			{
				ID:         "ex1",
				Title:      "Example One",
				Symbols:    []string{"fn"},
				SourceFile: "test.js",
				Line:       20,
			},
		},
	})
	return s
}

func TestExport_JSON_Store(t *testing.T) {
	var buf bytes.Buffer
	err := Export(context.Background(), testStore(), &buf, Options{
		Format: FormatJSON,
		Shape:  ShapeStore,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("expected valid json: %v", err)
	}
	if _, ok := m["files"]; !ok {
		t.Fatalf("expected files key in json output")
	}
}

func TestExport_YAML_Files(t *testing.T) {
	var buf bytes.Buffer
	err := Export(context.Background(), testStore(), &buf, Options{
		Format: FormatYAML,
		Shape:  ShapeFiles,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out any
	if err := yaml.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("expected valid yaml: %v", err)
	}
	if _, ok := out.([]any); !ok {
		t.Fatalf("expected yaml array for ShapeFiles, got %T", out)
	}
}

func TestExport_Markdown(t *testing.T) {
	var buf bytes.Buffer
	err := Export(context.Background(), testStore(), &buf, Options{
		Format:   FormatMarkdown,
		TOCDepth: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("## Packages")) {
		t.Fatalf("expected packages section")
	}
	if !bytes.Contains(buf.Bytes(), []byte("## Table of Contents")) {
		t.Fatalf("expected toc section")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Symbol: fn")) {
		t.Fatalf("expected symbol entry, got:\n%s", s)
	}
}
