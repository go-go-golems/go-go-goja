package sandbox

import (
	"sort"
	"strings"
	"sync"
)

const storeSeparator = "\x1f"

// MemoryStore is a concurrency-safe in-memory key/value store.
//
// It is intentionally simple: the first sandbox version keeps all state in
// process memory so that the scripting model stays easy to understand and
// test.
type MemoryStore struct {
	root   *memoryStoreRoot
	prefix string
}

type memoryStoreRoot struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewMemoryStore creates a fresh store with no namespace prefix.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{root: &memoryStoreRoot{data: map[string]any{}}}
}

func (s *MemoryStore) ensureRoot() *memoryStoreRoot {
	if s == nil {
		return &memoryStoreRoot{data: map[string]any{}}
	}
	if s.root == nil {
		s.root = &memoryStoreRoot{data: map[string]any{}}
	}
	if s.root.data == nil {
		s.root.data = map[string]any{}
	}
	return s.root
}

func (s *MemoryStore) scopedKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	prefix := strings.Trim(s.prefix, storeSeparator)
	if prefix == "" {
		return key
	}
	return prefix + storeSeparator + key
}

func namespacePrefix(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	return strings.Join(cleaned, storeSeparator)
}

// Namespace returns a new store view rooted at the same backing data but with
// an additional prefix.
func (s *MemoryStore) Namespace(parts ...string) *MemoryStore {
	root := s.ensureRoot()
	prefixParts := make([]string, 0, 1+len(parts))
	if s != nil && strings.TrimSpace(s.prefix) != "" {
		prefixParts = append(prefixParts, strings.Split(strings.Trim(s.prefix, storeSeparator), storeSeparator)...)
	}
	prefixParts = append(prefixParts, parts...)
	return &MemoryStore{
		root:   root,
		prefix: namespacePrefix(prefixParts...),
	}
}

// Get retrieves a value by key. If the key is not present, defaultValue is
// returned.
func (s *MemoryStore) Get(key string, defaultValue any) any {
	root := s.ensureRoot()
	lookup := s.scopedKey(key)
	if lookup == "" {
		return defaultValue
	}

	root.mu.RLock()
	value, ok := root.data[lookup]
	root.mu.RUnlock()
	if !ok {
		return defaultValue
	}
	return value
}

// Set stores a value by key.
func (s *MemoryStore) Set(key string, value any) {
	root := s.ensureRoot()
	lookup := s.scopedKey(key)
	if lookup == "" {
		return
	}

	root.mu.Lock()
	root.data[lookup] = value
	root.mu.Unlock()
}

// Delete removes a key from the store. It returns true when a value was
// removed.
func (s *MemoryStore) Delete(key string) bool {
	root := s.ensureRoot()
	lookup := s.scopedKey(key)
	if lookup == "" {
		return false
	}

	root.mu.Lock()
	defer root.mu.Unlock()
	if _, ok := root.data[lookup]; !ok {
		return false
	}
	delete(root.data, lookup)
	return true
}

// Keys returns the keys visible from the current namespace, optionally
// filtered by a relative prefix.
func (s *MemoryStore) Keys(prefix string) []string {
	root := s.ensureRoot()
	namespace := strings.Trim(s.prefix, storeSeparator)
	filter := strings.TrimSpace(prefix)

	root.mu.RLock()
	defer root.mu.RUnlock()

	out := make([]string, 0, len(root.data))
	for key := range root.data {
		relative, ok := s.relativeKey(key, namespace)
		if !ok {
			continue
		}
		if filter != "" && !strings.HasPrefix(relative, filter) {
			continue
		}
		out = append(out, relative)
	}
	sort.Strings(out)
	return out
}

// Snapshot returns a copy of all keys visible from the current namespace.
func (s *MemoryStore) Snapshot() map[string]any {
	root := s.ensureRoot()
	namespace := strings.Trim(s.prefix, storeSeparator)

	root.mu.RLock()
	defer root.mu.RUnlock()

	snapshot := map[string]any{}
	for key, value := range root.data {
		relative, ok := s.relativeKey(key, namespace)
		if !ok {
			continue
		}
		snapshot[relative] = value
	}
	return snapshot
}

func (s *MemoryStore) relativeKey(fullKey, namespace string) (string, bool) {
	fullKey = strings.TrimSpace(fullKey)
	if fullKey == "" {
		return "", false
	}
	namespace = strings.Trim(namespace, storeSeparator)
	if namespace == "" {
		return fullKey, true
	}
	prefix := namespace + storeSeparator
	if fullKey == namespace {
		return "", false
	}
	if !strings.HasPrefix(fullKey, prefix) {
		return "", false
	}
	return strings.TrimPrefix(fullKey, prefix), true
}
