package batch

import (
	"context"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

func TestBuildStore_FailFast(t *testing.T) {
	_, err := BuildStore(context.Background(), []InputFile{
		{Path: "this-file-does-not-exist.js"},
	}, BatchOptions{ContinueOnError: false})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBuildStore_ContinueOnError(t *testing.T) {
	res, err := BuildStore(context.Background(), []InputFile{
		{Path: "this-file-does-not-exist.js", DisplayName: "missing"},
		{DisplayName: "inline-ok.js", Content: []byte(`__doc__({"name":"a","doc":"A"})`)},
	}, BatchOptions{ContinueOnError: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.Store == nil {
		t.Fatalf("expected non-nil result/store")
	}
	if got := len(res.Store.Files); got != 1 {
		t.Fatalf("expected 1 parsed file, got %d", got)
	}
	if got := len(res.Store.BySymbol); got != 1 {
		t.Fatalf("expected 1 symbol, got %d", got)
	}
	if got := len(res.Errors); got != 1 {
		t.Fatalf("expected 1 error, got %d", got)
	}
	if res.Errors[0].Input.DisplayName != "missing" {
		t.Fatalf("expected error displayName to be preserved, got %q", res.Errors[0].Input.DisplayName)
	}
}

func TestBuildStore_InvalidInputs(t *testing.T) {
	_, err := BuildStore(context.Background(), []InputFile{{}}, BatchOptions{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	_, err = BuildStore(context.Background(), []InputFile{{
		Path:    "x.js",
		Content: []byte("x"),
	}}, BatchOptions{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBuildStore_CustomParsePath(t *testing.T) {
	called := false
	res, err := BuildStore(context.Background(), []InputFile{
		{Path: "virtual.js"},
	}, BatchOptions{
		ParsePath: func(path string) (*model.FileDoc, error) {
			called = true
			return extract.ParseSource(path, []byte(`__doc__({"name":"custom"})`))
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected custom ParsePath to be called")
	}
	if got := len(res.Store.BySymbol); got != 1 {
		t.Fatalf("expected 1 symbol, got %d", got)
	}
	if _, ok := res.Store.BySymbol["custom"]; !ok {
		t.Fatalf("expected custom symbol in store")
	}
}
