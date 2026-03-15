package obsidian

import (
	"context"
	"sync"

	"github.com/go-go-golems/go-go-goja/pkg/obsidianmd"
)

// Note represents one lazily loaded markdown note.
type Note struct {
	client *Client

	Path  string
	Title string

	mu          sync.Mutex
	loaded      bool
	content     string
	frontmatter map[string]any
	wikilinks   []string
	headings    []obsidianmd.Heading
	tags        []string
	tasks       []obsidianmd.Task
}

// Content returns the note body, loading it if needed.
func (n *Note) Content(ctx context.Context) (string, error) {
	if err := n.ensureLoaded(ctx); err != nil {
		return "", err
	}
	return n.content, nil
}

// Frontmatter returns parsed YAML frontmatter.
func (n *Note) Frontmatter(ctx context.Context) (map[string]any, error) {
	if err := n.ensureLoaded(ctx); err != nil {
		return nil, err
	}
	if n.frontmatter == nil {
		return nil, nil
	}
	ret := map[string]any{}
	for key, value := range n.frontmatter {
		ret[key] = value
	}
	return ret, nil
}

// Wikilinks returns outgoing wikilink targets in the note body.
func (n *Note) Wikilinks(ctx context.Context) ([]string, error) {
	if err := n.ensureLoaded(ctx); err != nil {
		return nil, err
	}
	return append([]string(nil), n.wikilinks...), nil
}

// Headings returns markdown headings discovered in the note.
func (n *Note) Headings(ctx context.Context) ([]obsidianmd.Heading, error) {
	if err := n.ensureLoaded(ctx); err != nil {
		return nil, err
	}
	return append([]obsidianmd.Heading(nil), n.headings...), nil
}

// Tags returns hash-tags found in the note body.
func (n *Note) Tags(ctx context.Context) ([]string, error) {
	if err := n.ensureLoaded(ctx); err != nil {
		return nil, err
	}
	return append([]string(nil), n.tags...), nil
}

// Tasks returns markdown checkbox tasks found in the note.
func (n *Note) Tasks(ctx context.Context) ([]obsidianmd.Task, error) {
	if err := n.ensureLoaded(ctx); err != nil {
		return nil, err
	}
	return append([]obsidianmd.Task(nil), n.tasks...), nil
}

// Reload forces a re-read from the CLI transport and refreshes derived metadata.
func (n *Note) Reload(ctx context.Context) error {
	n.mu.Lock()
	n.loaded = false
	n.content = ""
	n.frontmatter = nil
	n.wikilinks = nil
	n.headings = nil
	n.tags = nil
	n.tasks = nil
	n.mu.Unlock()

	n.client.Invalidate(n.Path)
	return n.ensureLoaded(ctx)
}

func (n *Note) ensureLoaded(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.loaded {
		return nil
	}

	content, err := n.client.Read(ctx, n.Path)
	if err != nil {
		return err
	}

	doc, err := obsidianmd.ParseDocument(content)
	if err != nil {
		return err
	}

	n.loaded = true
	n.content = content
	n.frontmatter = doc.Frontmatter
	n.wikilinks = obsidianmd.ExtractWikilinks(doc.Body)
	n.headings = obsidianmd.ExtractHeadings(doc.Body)
	n.tags = obsidianmd.ExtractTags(doc.Body)
	n.tasks = obsidianmd.ExtractTasks(doc.Body)
	return nil
}
