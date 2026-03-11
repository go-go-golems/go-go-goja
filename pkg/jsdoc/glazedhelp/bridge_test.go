package glazedhelp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	glazedhelp "github.com/go-go-golems/glazed/pkg/help"
	jsdocextract "github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
	jsdocmodel "github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

func TestBuildSectionsCreatesStableSlugsAndRelationships(t *testing.T) {
	store := buildFixtureStore(t, "01-math.js", "02-easing.js")

	result, err := BuildSections(store, Options{})
	if err != nil {
		t.Fatalf("BuildSections() error = %v", err)
	}

	if result.RootSlug != "jsdoc" {
		t.Fatalf("unexpected root slug: %q", result.RootSlug)
	}
	if got := result.PackageSlugs["animation/easing"]; got != "jsdoc/package/animation/easing" {
		t.Fatalf("unexpected package slug: %q", got)
	}
	if got := result.SymbolSlugs["smoothstep"]; got != "jsdoc/symbol/smoothstep" {
		t.Fatalf("unexpected symbol slug: %q", got)
	}
	if got := result.ExampleSlugs["smoothstep-basic"]; got != "jsdoc/example/smoothstep-basic" {
		t.Fatalf("unexpected example slug: %q", got)
	}

	symbol := findSection(t, result, "jsdoc/symbol/smoothstep")
	if !contains(symbol.Topics, "jsdoc/package/animation/easing") {
		t.Fatalf("expected package topic on symbol section: %#v", symbol.Topics)
	}

	example := findSection(t, result, "jsdoc/example/smoothstep-basic")
	if !contains(example.Topics, "jsdoc/symbol/smoothstep") {
		t.Fatalf("expected symbol topic on example section: %#v", example.Topics)
	}
}

func TestComposerRendersRelatedSections(t *testing.T) {
	store := buildFixtureStore(t, "02-easing.js")
	result, err := BuildSections(store, Options{})
	if err != nil {
		t.Fatalf("BuildSections() error = %v", err)
	}

	hs := glazedhelp.NewHelpSystem()
	if err := LoadIntoHelpSystem(context.Background(), hs, result); err != nil {
		t.Fatalf("LoadIntoHelpSystem() error = %v", err)
	}
	hs.SetPageComposer(NewComposer(store, result))

	section, err := hs.GetSectionWithSlug("jsdoc/symbol/smoothstep")
	if err != nil {
		t.Fatalf("GetSectionWithSlug() error = %v", err)
	}
	rendered, err := hs.RenderTopicHelpWithWriter(section, &glazedhelp.RenderOptions{
		Query: glazedhelp.NewSectionQuery().ReturnAllTypes(),
	}, os.Stdout)
	if err != nil {
		t.Fatalf("RenderTopicHelpWithWriter() error = %v", err)
	}

	if !strings.Contains(rendered, "JSDoc Identity") {
		t.Fatalf("expected jsdoc metadata block in rendered page: %q", rendered)
	}
	if !strings.Contains(rendered, "Related Help Pages") {
		t.Fatalf("expected related section block in rendered page: %q", rendered)
	}
	if !strings.Contains(rendered, "smoothstep-basic") {
		t.Fatalf("expected related example in rendered page: %q", rendered)
	}
}

func TestBuildMarkdownFilesReloadsThroughHelpSystem(t *testing.T) {
	store := buildFixtureStore(t, "01-math.js")
	result, err := BuildSections(store, Options{})
	if err != nil {
		t.Fatalf("BuildSections() error = %v", err)
	}

	files, err := BuildMarkdownFiles(result)
	if err != nil {
		t.Fatalf("BuildMarkdownFiles() error = %v", err)
	}

	dir := t.TempDir()
	if err := WriteMarkdownFiles(dir, files); err != nil {
		t.Fatalf("WriteMarkdownFiles() error = %v", err)
	}

	hs := glazedhelp.NewHelpSystem()
	if err := hs.LoadSectionsFromFS(os.DirFS(dir), "."); err != nil {
		t.Fatalf("LoadSectionsFromFS() error = %v", err)
	}

	if _, err := hs.GetSectionWithSlug("jsdoc"); err != nil {
		t.Fatalf("expected root section after reload: %v", err)
	}
	if _, err := hs.GetSectionWithSlug("jsdoc/symbol/clamp"); err != nil {
		t.Fatalf("expected symbol section after reload: %v", err)
	}
}

func buildFixtureStore(t *testing.T, names ...string) *jsdocmodel.DocStore {
	t.Helper()
	store := jsdocmodel.NewDocStore()
	for _, name := range names {
		path := filepath.Join("..", "..", "..", "testdata", "jsdoc", name)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		fd, err := jsdocextract.ParseSource(path, src)
		if err != nil {
			t.Fatalf("ParseSource(%s) error = %v", path, err)
		}
		store.AddFile(fd)
	}
	return store
}

func findSection(t *testing.T, result *Result, slug string) *glazedhelp.Section {
	t.Helper()
	hs := glazedhelp.NewHelpSystem()
	if err := LoadIntoHelpSystem(context.Background(), hs, result); err != nil {
		t.Fatalf("LoadIntoHelpSystem() error = %v", err)
	}
	section, err := hs.GetSectionWithSlug(slug)
	if err != nil {
		t.Fatalf("GetSectionWithSlug(%s) error = %v", slug, err)
	}
	return section
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
