package glazehelp

import (
	"fmt"
	"sync"

	"github.com/go-go-golems/glazed/pkg/help"
)

var (
	systems = map[string]*help.HelpSystem{}
	mu      sync.RWMutex
)

// Register adds a HelpSystem to the registry with the given key.
// If a system with this key already exists, it will be replaced.
func Register(key string, hs *help.HelpSystem) {
	mu.Lock()
	defer mu.Unlock()
	systems[key] = hs
}

// MustRegister adds a HelpSystem to the registry with the given key.
// If a system with this key already exists, it panics.
func MustRegister(key string, hs *help.HelpSystem) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := systems[key]; ok {
		panic(fmt.Sprintf("help system with key %s already registered", key))
	}
	systems[key] = hs
}

// Get retrieves a HelpSystem by key.
// Returns an error if the key is not found.
func Get(key string) (*help.HelpSystem, error) {
	mu.RLock()
	defer mu.RUnlock()
	hs, ok := systems[key]
	if !ok {
		return nil, fmt.Errorf("help system %s not found", key)
	}
	return hs, nil
}

// Keys returns all registered help system keys.
func Keys() []string {
	mu.RLock()
	defer mu.RUnlock()
	keys := make([]string, 0, len(systems))
	for k := range systems {
		keys = append(keys, k)
	}
	return keys
}

// Clear removes all registered help systems.
// Primarily used for testing.
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	systems = make(map[string]*help.HelpSystem)
}
