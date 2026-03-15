package obsidian

import "sync"

// Cache stores note contents by resolved path.
type Cache struct {
	mu       sync.RWMutex
	contents map[string]string
}

// NewCache creates an empty content cache.
func NewCache() *Cache {
	return &Cache{
		contents: map[string]string{},
	}
}

// Get returns the cached content for a path if present.
func (c *Cache) Get(path string) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	content, ok := c.contents[path]
	return content, ok
}

// Set stores content for a path.
func (c *Cache) Set(path string, content string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contents[path] = content
}

// Delete removes one cached path.
func (c *Cache) Delete(path string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.contents, path)
}

// Clear removes every cached note.
func (c *Cache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.contents)
}
