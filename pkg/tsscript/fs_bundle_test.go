package tsscript

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestBundleVirtualEntryFSResolvesRelativeImports(t *testing.T) {
	root := fstest.MapFS{
		"verbs/helper.ts": {Data: []byte(`export function message(name: string): string { return "hello " + name }`)},
	}
	artifact, err := BundleVirtualEntryFS(root, Source{
		Path:     "verbs/site.ts",
		Contents: []byte(`import { message } from "./helper"; module.exports = { value: message("xgoja") }`),
	}, Options{})
	if err != nil {
		t.Fatalf("BundleVirtualEntryFS: %v", err)
	}
	if !artifact.Bundled {
		t.Fatal("expected bundled artifact")
	}
	code := string(artifact.Code)
	if !strings.Contains(code, "hello ") || strings.Contains(code, "./helper") {
		t.Fatalf("unexpected bundle output:\n%s", code)
	}
}

func TestBundleVirtualEntryFSExternalizesRuntimeAliases(t *testing.T) {
	root := fstest.MapFS{}
	artifact, err := BundleVirtualEntryFS(root, Source{
		Path:     "verbs/site.ts",
		Contents: []byte(`const express = require("express"); module.exports = express`),
	}, Options{External: []string{"express"}})
	if err != nil {
		t.Fatalf("BundleVirtualEntryFS: %v", err)
	}
	if !strings.Contains(string(artifact.Code), `require("express")`) {
		t.Fatalf("expected express require preserved:\n%s", artifact.Code)
	}
}

func TestBundleVirtualEntryFSReportsMissingImport(t *testing.T) {
	_, err := BundleVirtualEntryFS(fstest.MapFS{}, Source{
		Path:     "verbs/site.ts",
		Contents: []byte(`import "./missing"`),
	}, Options{})
	if err == nil || !strings.Contains(err.Error(), "no matching fs source") {
		t.Fatalf("expected missing fs import error, got %v", err)
	}
}
