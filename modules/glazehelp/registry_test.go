package glazehelp

import (
	"testing"

	"github.com/go-go-golems/glazed/pkg/help"
)

func TestRegistry(t *testing.T) {
	// Clear registry before testing
	Clear()

	// Test registration
	hs1 := help.NewHelpSystem()
	hs2 := help.NewHelpSystem()

	Register("test1", hs1)
	Register("test2", hs2)

	// Test retrieval
	retrieved1, err := Get("test1")
	if err != nil {
		t.Fatalf("Failed to get test1: %v", err)
	}
	if retrieved1 != hs1 {
		t.Errorf("Retrieved wrong help system for test1")
	}

	retrieved2, err := Get("test2")
	if err != nil {
		t.Fatalf("Failed to get test2: %v", err)
	}
	if retrieved2 != hs2 {
		t.Errorf("Retrieved wrong help system for test2")
	}

	// Test non-existent key
	_, err = Get("nonexistent")
	if err == nil {
		t.Errorf("Expected error for nonexistent key")
	}

	// Test keys listing
	keys := Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	expectedKeys := map[string]bool{"test1": true, "test2": true}
	for _, key := range keys {
		if !expectedKeys[key] {
			t.Errorf("Unexpected key: %s", key)
		}
	}
}

func TestMustRegister(t *testing.T) {
	Clear()

	hs := help.NewHelpSystem()

	// First registration should succeed
	MustRegister("unique", hs)

	// Second registration with same key should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for duplicate key")
		}
	}()
	MustRegister("unique", hs)
}

func TestClear(t *testing.T) {
	hs := help.NewHelpSystem()
	Register("test", hs)

	if len(Keys()) == 0 {
		t.Errorf("Expected at least one key before clear")
	}

	Clear()

	if len(Keys()) != 0 {
		t.Errorf("Expected no keys after clear")
	}
}
