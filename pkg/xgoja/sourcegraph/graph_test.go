package sourcegraph

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestBuildDiskSourceSetFiltersFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site.ts"), `import { helper } from "./helper"`)
	writeFile(t, filepath.Join(root, "helper.ts"), `export const helper = 1`)
	writeFile(t, filepath.Join(root, "site.test.ts"), `export const test = 1`)
	writeFile(t, filepath.Join(root, "README.md"), `ignore`)

	graph, err := Build([]SourceSet{{
		ID:         "sites",
		Kind:       SourceKindJSVerbs,
		Origin:     Origin{Kind: OriginDisk, Dir: root},
		Include:    []string{"**/*.ts"},
		Exclude:    []string{"**/*.test.ts"},
		Extensions: []string{".ts"},
	}}, Options{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	files := graph.FilesForSourceSet("sites")
	if len(files) != 2 {
		t.Fatalf("files = %#v", files)
	}
	if files[0].Path != "helper.ts" || files[1].Path != "site.ts" {
		t.Fatalf("unexpected file order: %#v", files)
	}
}

func TestBuildFSProviderSourceSet(t *testing.T) {
	providerFS := fstest.MapFS{
		"verbs/site.ts":   {Data: []byte(`export const site = 1`)},
		"verbs/ignore.md": {Data: []byte(`ignore`)},
	}
	graph, err := Build([]SourceSet{{
		ID:         "provider-sites",
		Kind:       SourceKindJSVerbs,
		Origin:     Origin{Kind: OriginProvider, FS: providerFS, Root: "verbs", Provider: "http", Source: "sites"},
		Extensions: []string{".ts"},
	}}, Options{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	files := graph.Files()
	if len(files) != 1 || files[0].Path != "site.ts" || files[0].OriginKind != OriginProvider {
		t.Fatalf("files = %#v", files)
	}
}

func TestResolveImportsClassifiesLocalAndRuntime(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site.ts"), `import { helper } from "./helper"
const express = require("express")
`)
	writeFile(t, filepath.Join(root, "helper.ts"), `export const helper = 1`)
	graph, err := Build([]SourceSet{{ID: "sites", Kind: SourceKindJSVerbs, Origin: Origin{Kind: OriginDisk, Dir: root}, Extensions: []string{".ts"}}}, Options{RuntimeModuleAliases: []string{"express"}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := graph.ResolveImports(readSourceFile); err != nil {
		t.Fatalf("ResolveImports: %v", err)
	}
	var site File
	for _, file := range graph.Files() {
		if file.Path == "site.ts" {
			site = file
		}
	}
	resolutions := graph.ImportResolutions(site)
	if len(resolutions) != 2 {
		t.Fatalf("resolutions = %#v", resolutions)
	}
	if resolutions[0].Kind != ImportLocal || resolutions[0].TargetPath != "helper.ts" {
		t.Fatalf("local resolution = %#v", resolutions[0])
	}
	if resolutions[1].Kind != ImportRuntime || resolutions[1].Alias != "express" {
		t.Fatalf("runtime resolution = %#v", resolutions[1])
	}
}

func TestResolveImportsRejectsUnknownBareImport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site.ts"), `import x from "left-pad"`)
	graph, err := Build([]SourceSet{{ID: "sites", Kind: SourceKindJSVerbs, Origin: Origin{Kind: OriginDisk, Dir: root}, Extensions: []string{".ts"}}}, Options{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	err = graph.ResolveImports(readSourceFile)
	if err == nil || !strings.Contains(err.Error(), "unknown bare specifier") {
		t.Fatalf("expected unknown bare import error, got %v", err)
	}
}

func TestResolveImportsRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site.ts"), `import x from "../outside"`)
	graph, err := Build([]SourceSet{{ID: "sites", Kind: SourceKindJSVerbs, Origin: Origin{Kind: OriginDisk, Dir: root}, Extensions: []string{".ts"}}}, Options{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	err = graph.ResolveImports(readSourceFile)
	if err == nil || !strings.Contains(err.Error(), "outside source root") {
		t.Fatalf("expected path escape error, got %v", err)
	}
}

func readSourceFile(file File) ([]byte, error) {
	if file.AbsPath != "" {
		return os.ReadFile(file.AbsPath)
	}
	return nil, fs.ErrNotExist
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
