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

func TestResolveImportsAllowsColonRuntimeAliases(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site.js"), `const assets = require("fs:assets")
import db from "db:readonly"
`)
	graph, err := Build([]SourceSet{{ID: "sites", Kind: SourceKindJSVerbs, Origin: Origin{Kind: OriginDisk, Dir: root}, Extensions: []string{".js"}}}, Options{RuntimeModuleAliases: []string{"fs:assets", "db:readonly"}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := graph.ResolveImports(readSourceFile); err != nil {
		t.Fatalf("ResolveImports: %v", err)
	}
	var site File
	for _, file := range graph.Files() {
		if file.Path == "site.js" {
			site = file
		}
	}
	resolutions := graph.ImportResolutions(site)
	if len(resolutions) != 2 {
		t.Fatalf("resolutions = %#v", resolutions)
	}
	if resolutions[0].Kind != ImportRuntime || resolutions[0].Alias != "fs:assets" {
		t.Fatalf("first runtime resolution = %#v", resolutions[0])
	}
	if resolutions[1].Kind != ImportRuntime || resolutions[1].Alias != "db:readonly" {
		t.Fatalf("second runtime resolution = %#v", resolutions[1])
	}
}

func TestResolveImportsParsesESMExportDynamicAndIgnoresStrings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site.tsx"), `import type { Thing } from "./types"
import { helper } from "./helper"
import assets from "fs:assets"
import "./setup"
export { more } from "./more"
const dynamic = await import("./dynamic")
const commented = "require('not-real')" // import "also-not-real"
export const View = () => <section>{helper}</section>
`)
	writeFile(t, filepath.Join(root, "types.ts"), `export interface Thing { name: string }`)
	writeFile(t, filepath.Join(root, "helper.ts"), `export const helper = 1`)
	writeFile(t, filepath.Join(root, "setup.ts"), `export const setup = true`)
	writeFile(t, filepath.Join(root, "more.ts"), `export const more = true`)
	writeFile(t, filepath.Join(root, "dynamic.ts"), `export const dynamic = true`)
	graph, err := Build([]SourceSet{{ID: "sites", Kind: SourceKindJSVerbs, Origin: Origin{Kind: OriginDisk, Dir: root}, Extensions: []string{".ts", ".tsx"}}}, Options{RuntimeModuleAliases: []string{"fs:assets"}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := graph.ResolveImports(readSourceFile); err != nil {
		t.Fatalf("ResolveImports: %v", err)
	}
	var site File
	for _, file := range graph.Files() {
		if file.Path == "site.tsx" {
			site = file
		}
	}
	resolutions := graph.ImportResolutions(site)
	if len(resolutions) != 6 {
		t.Fatalf("resolutions = %#v", resolutions)
	}
	want := []ImportResolution{
		{Kind: ImportLocal, TargetPath: "types.ts"},
		{Kind: ImportLocal, TargetPath: "helper.ts"},
		{Kind: ImportRuntime, Alias: "fs:assets"},
		{Kind: ImportLocal, TargetPath: "setup.ts"},
		{Kind: ImportLocal, TargetPath: "more.ts"},
		{Kind: ImportLocal, TargetPath: "dynamic.ts"},
	}
	for i, wantResolution := range want {
		got := resolutions[i]
		if got.Kind != wantResolution.Kind || got.TargetPath != wantResolution.TargetPath || got.Alias != wantResolution.Alias {
			t.Fatalf("resolution[%d] = %#v, want kind=%s target=%q alias=%q", i, got, wantResolution.Kind, wantResolution.TargetPath, wantResolution.Alias)
		}
	}
}

func TestResolveImportsRejectsDynamicNonLiteralImport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site.js"), `const assets = require(["fs", "assets"].join(":"))`)
	graph, err := Build([]SourceSet{{ID: "sites", Kind: SourceKindJSVerbs, Origin: Origin{Kind: OriginDisk, Dir: root}, Extensions: []string{".js"}}}, Options{RuntimeModuleAliases: []string{"fs:assets"}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	err = graph.ResolveImports(readSourceFile)
	if err == nil || !strings.Contains(err.Error(), "dynamic non-literal require import") {
		t.Fatalf("expected dynamic import error, got %v", err)
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
