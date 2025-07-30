package glazehelp

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/help/model"
	"github.com/go-go-golems/go-go-goja/modules"
)

func TestGlazeHelpModule(t *testing.T) {
	// Clear registry and setup test help system
	Clear()

	hs := help.NewHelpSystem()

	// Add a test section
	section := &help.Section{
		Section: &model.Section{
			Title:          "Test Section",
			Slug:           "test-section",
			Short:          "A test section",
			Content:        "This is test content",
			Topics:         []string{"testing", "example"},
			Commands:       []string{"test"},
			IsTopLevel:     true,
			ShowPerDefault: true,
			SectionType:    model.SectionGeneralTopic,
		},
		HelpSystem: hs,
	}
	hs.AddSection(section)

	Register("test", hs)

	// Setup goja runtime with require
	vm := goja.New()
	reg := require.NewRegistry()

	// Register our module
	modules.EnableAll(reg)
	_ = reg.Enable(vm)

	// Test keys() function
	result, err := vm.RunString(`
		const help = require("glazehelp");
		help.keys();
	`)
	if err != nil {
		t.Fatalf("Error running keys(): %v", err)
	}

	keysResult := result.Export()
	keys, ok := keysResult.([]string)
	if !ok {
		// Try []interface{} as fallback
		keysInterface := keysResult.([]interface{})
		keys = make([]string, len(keysInterface))
		for i, v := range keysInterface {
			keys[i] = v.(string)
		}
	}
	if len(keys) != 1 || keys[0] != "test" {
		t.Errorf("Expected keys to contain 'test', got %v", keys)
	}

	// Test section() function
	result, err = vm.RunString(`
		help.section("test", "test-section");
	`)
	if err != nil {
		t.Fatalf("Error running section(): %v", err)
	}

	sectionMap := result.Export().(map[string]interface{})
	if sectionMap["title"] != "Test Section" {
		t.Errorf("Expected title 'Test Section', got %v", sectionMap["title"])
	}
	if sectionMap["slug"] != "test-section" {
		t.Errorf("Expected slug 'test-section', got %v", sectionMap["slug"])
	}

	// Test topics() function
	result, err = vm.RunString(`
		help.topics("test");
	`)
	if err != nil {
		t.Fatalf("Error running topics(): %v", err)
	}

	topicsResult := result.Export()
	topics, ok := topicsResult.([]string)
	if !ok {
		// Try []interface{} as fallback
		topicsInterface := topicsResult.([]interface{})
		topics = make([]string, len(topicsInterface))
		for i, v := range topicsInterface {
			topics[i] = v.(string)
		}
	}
	if len(topics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(topics))
	}

	topicMap := make(map[string]bool)
	for _, topic := range topics {
		topicMap[topic] = true
	}
	if !topicMap["testing"] || !topicMap["example"] {
		t.Errorf("Expected topics 'testing' and 'example', got %v", topics)
	}

	// Test query() function
	result, err = vm.RunString(`
		help.query("test", "");
	`)
	if err != nil {
		t.Fatalf("Error running query(): %v", err)
	}

	sectionsResult := result.Export()
	sections, ok := sectionsResult.([]map[string]interface{})
	if !ok {
		// Try []interface{} as fallback
		sectionsInterface := sectionsResult.([]interface{})
		sections = make([]map[string]interface{}, len(sectionsInterface))
		for i, v := range sectionsInterface {
			sections[i] = v.(map[string]interface{})
		}
	}
	if len(sections) != 1 {
		t.Errorf("Expected 1 section from query, got %d", len(sections))
	}

	firstSection := sections[0]
	if firstSection["title"] != "Test Section" {
		t.Errorf("Expected first section title 'Test Section', got %v", firstSection["title"])
	}

	// Test error handling for non-existent key
	_, err = vm.RunString(`
		try {
			help.section("nonexistent", "slug");
		} catch (e) {
			throw new Error("Expected error for nonexistent key: " + e.message);
		}
	`)
	if err == nil {
		t.Errorf("Expected error for nonexistent help system key")
	}
	if !strings.Contains(err.Error(), "help system nonexistent not found") {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	// Test null return for non-existent section
	result, err = vm.RunString(`
		help.section("test", "nonexistent-slug");
	`)
	if err != nil {
		t.Fatalf("Error running section() with nonexistent slug: %v", err)
	}

	if result.Export() != nil {
		t.Errorf("Expected null for nonexistent section, got %v", result.Export())
	}
}

func TestGlazeHelpRender(t *testing.T) {
	Clear()

	hs := help.NewHelpSystem()
	Register("render-test", hs)

	// Setup goja runtime
	vm := goja.New()
	reg := require.NewRegistry()
	modules.EnableAll(reg)
	_ = reg.Enable(vm)

	// Test render() function
	result, err := vm.RunString(`
		const help = require("glazehelp");
		help.render("render-test");
	`)
	if err != nil {
		t.Fatalf("Error running render(): %v", err)
	}

	rendered := result.Export()
	// Should return a HelpPage object (even if minimal)
	if rendered == nil {
		t.Errorf("Expected non-nil rendered content")
	}
}
