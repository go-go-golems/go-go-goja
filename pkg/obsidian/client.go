package obsidian

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/obsidiancli"
	"github.com/pkg/errors"
)

// Client provides a higher-level Obsidian API over the CLI transport.
type Client struct {
	runner Runner
	cache  *Cache
}

// NewClient creates a new high-level client.
func NewClient(cfg Config, runner Runner) *Client {
	if cfg.Cache == nil {
		cfg.Cache = NewCache()
	}
	if runner == nil {
		runner = obsidiancli.NewRunner(cfg.CLI, nil)
	}
	return &Client{
		runner: runner,
		cache:  cfg.Cache,
	}
}

// Version returns the reported Obsidian version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	result, err := c.runner.Run(ctx, obsidiancli.SpecVersion, obsidiancli.CallOptions{})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resultString(result)), nil
}

// Files lists matching files using native CLI filters where possible.
func (c *Client) Files(ctx context.Context, options FileListOptions) ([]string, error) {
	call := obsidiancli.CallOptions{
		Vault: options.Vault,
		Parameters: map[string]any{
			"folder": strings.TrimSpace(options.Folder),
			"ext":    normalizeExt(options.Ext),
		},
	}
	if options.Limit > 0 {
		call.Parameters["limit"] = options.Limit
	}
	result, err := c.runner.Run(ctx, obsidiancli.SpecFilesList, call)
	if err != nil {
		return nil, err
	}
	return resultStrings(result)
}

// Search runs a vault search and returns matching paths.
func (c *Client) Search(ctx context.Context, term string, options SearchOptions) ([]string, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, errors.New("obsidian: search term is empty")
	}
	call := obsidiancli.CallOptions{
		Vault: options.Vault,
		Parameters: map[string]any{
			"q":      term,
			"folder": strings.TrimSpace(options.Folder),
			"ext":    normalizeExt(options.Ext),
		},
	}
	if options.Limit > 0 {
		call.Parameters["limit"] = options.Limit
	}
	result, err := c.runner.Run(ctx, obsidiancli.SpecSearch, call)
	if err != nil {
		return nil, err
	}
	return resultStrings(result)
}

// Read returns the content of one note or markdown file.
func (c *Client) Read(ctx context.Context, ref string) (string, error) {
	path, err := c.resolveReference(ctx, ref)
	if err != nil {
		return "", err
	}
	if content, ok := c.cache.Get(path); ok {
		return content, nil
	}

	result, err := c.runner.Run(ctx, obsidiancli.SpecFileRead, obsidiancli.CallOptions{
		Parameters: map[string]any{
			"path": path,
		},
	})
	if err != nil {
		return "", err
	}
	content := resultString(result)
	c.cache.Set(path, content)
	return content, nil
}

// Create creates a note and invalidates related caches.
func (c *Client) Create(ctx context.Context, title string, options CreateOptions) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", errors.New("obsidian: title is empty")
	}
	result, err := c.runner.Run(ctx, obsidiancli.SpecFileCreate, obsidiancli.CallOptions{
		Vault: options.Vault,
		Parameters: map[string]any{
			"title":    title,
			"content":  options.Content,
			"folder":   strings.TrimSpace(options.Folder),
			"template": strings.TrimSpace(options.Template),
		},
	})
	if err != nil {
		return "", err
	}
	c.Invalidate()
	return strings.TrimSpace(resultString(result)), nil
}

// Append adds content to the end of a note.
func (c *Client) Append(ctx context.Context, ref string, content string) error {
	path, err := c.resolveReference(ctx, ref)
	if err != nil {
		return err
	}
	_, err = c.runner.Run(ctx, obsidiancli.SpecFileAppend, obsidiancli.CallOptions{
		Parameters: map[string]any{
			"path":    path,
			"content": content,
		},
	})
	if err != nil {
		return err
	}
	c.Invalidate(path)
	return nil
}

// Prepend adds content to the start of a note.
func (c *Client) Prepend(ctx context.Context, ref string, content string) error {
	path, err := c.resolveReference(ctx, ref)
	if err != nil {
		return err
	}
	_, err = c.runner.Run(ctx, obsidiancli.SpecFilePrepend, obsidiancli.CallOptions{
		Parameters: map[string]any{
			"path":    path,
			"content": content,
		},
	})
	if err != nil {
		return err
	}
	c.Invalidate(path)
	return nil
}

// Move moves a note to a new folder or path.
func (c *Client) Move(ctx context.Context, ref string, destination string) error {
	path, err := c.resolveReference(ctx, ref)
	if err != nil {
		return err
	}
	_, err = c.runner.Run(ctx, obsidiancli.SpecFileMove, obsidiancli.CallOptions{
		Parameters: map[string]any{
			"path":        path,
			"destination": strings.TrimSpace(destination),
		},
	})
	if err != nil {
		return err
	}
	c.Invalidate(path)
	return nil
}

// Rename renames a note.
func (c *Client) Rename(ctx context.Context, ref string, newName string) error {
	path, err := c.resolveReference(ctx, ref)
	if err != nil {
		return err
	}
	_, err = c.runner.Run(ctx, obsidiancli.SpecFileRename, obsidiancli.CallOptions{
		Parameters: map[string]any{
			"path": path,
			"name": strings.TrimSpace(newName),
		},
	})
	if err != nil {
		return err
	}
	c.Invalidate(path)
	return nil
}

// Delete removes a note using trash or permanent deletion.
func (c *Client) Delete(ctx context.Context, ref string, options DeleteOptions) error {
	path, err := c.resolveReference(ctx, ref)
	if err != nil {
		return err
	}
	spec := obsidiancli.SpecFileTrash
	if options.Permanent {
		spec = obsidiancli.SpecFileDelete
	}
	_, err = c.runner.Run(ctx, spec, obsidiancli.CallOptions{
		Vault: options.Vault,
		Parameters: map[string]any{
			"path": path,
		},
	})
	if err != nil {
		return err
	}
	c.Invalidate(path)
	return nil
}

// Note resolves a user-facing note reference into a lazy note object.
func (c *Client) Note(ctx context.Context, ref string) (*Note, error) {
	path, err := c.resolveReference(ctx, ref)
	if err != nil {
		return nil, err
	}
	return c.noteForPath(path), nil
}

// Query creates a new fluent query builder.
func (c *Client) Query() *Query {
	return &Query{
		client: c,
		mode:   queryModeFiles,
	}
}

// Invalidate removes cached note contents. No paths means clear everything.
func (c *Client) Invalidate(paths ...string) {
	if c.cache == nil {
		return
	}
	if len(paths) == 0 {
		c.cache.Clear()
		return
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		c.cache.Delete(path)
	}
}

func (c *Client) resolveReference(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("obsidian: note reference is empty")
	}
	if looksLikePath(ref) {
		return ref, nil
	}

	files, err := c.Files(ctx, FileListOptions{Ext: "md"})
	if err != nil {
		return "", err
	}

	matches := make([]string, 0, 4)
	for _, candidate := range files {
		if matchesReference(candidate, ref) {
			matches = append(matches, candidate)
		}
	}
	switch len(matches) {
	case 0:
		return "", &obsidiancli.NotFoundError{Ref: ref}
	case 1:
		return matches[0], nil
	default:
		return "", &obsidiancli.AmbiguousReferenceError{Ref: ref, Matches: matches}
	}
}

func (c *Client) noteForPath(path string) *Note {
	return &Note{
		client: c,
		Path:   path,
		Title:  displayName(path),
	}
}

func resultString(result obsidiancli.Result) string {
	if s, ok := result.Parsed.(string); ok {
		return s
	}
	if result.Stdout != "" {
		return result.Stdout
	}
	return fmt.Sprint(result.Parsed)
}

func resultStrings(result obsidiancli.Result) ([]string, error) {
	switch value := result.Parsed.(type) {
	case []string:
		return append([]string(nil), value...), nil
	case []any:
		ret := make([]string, 0, len(value))
		for _, item := range value {
			ret = append(ret, fmt.Sprint(item))
		}
		return ret, nil
	case string:
		return obsidiancli.ParseLineList(value), nil
	default:
		return nil, errors.Errorf("obsidian: unexpected list result type %T", result.Parsed)
	}
}

func extractPaths(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		ret := make([]string, 0, len(typed))
		for _, item := range typed {
			switch row := item.(type) {
			case string:
				ret = append(ret, row)
			case map[string]any:
				if path, ok := pathFromMap(row); ok {
					ret = append(ret, path)
				}
			}
		}
		return ret
	case map[string]any:
		if path, ok := pathFromMap(typed); ok {
			return []string{path}
		}
	}
	return nil
}

func pathFromMap(value map[string]any) (string, bool) {
	for _, key := range []string{"path", "file", "note"} {
		raw, ok := value[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(raw))
		if text != "" {
			return text, true
		}
	}
	return "", false
}

func callParameters(parameters map[string]any) map[string]any {
	if len(parameters) == 0 {
		return nil
	}
	ret := map[string]any{}
	for key, value := range parameters {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
		case int:
			if typed <= 0 {
				continue
			}
		}
		ret[key] = value
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func normalizeExt(ext string) string {
	ext = strings.TrimSpace(ext)
	ext = strings.TrimPrefix(ext, ".")
	return ext
}

func looksLikePath(ref string) bool {
	return strings.Contains(ref, "/") || strings.Contains(ref, "\\") || filepath.Ext(ref) != ""
}

func matchesReference(candidate string, ref string) bool {
	candidate = strings.TrimSpace(candidate)
	ref = strings.TrimSpace(ref)
	if candidate == "" || ref == "" {
		return false
	}
	if strings.EqualFold(candidate, ref) {
		return true
	}

	base := displayName(candidate)
	file := filepath.Base(candidate)
	return strings.EqualFold(base, ref) || strings.EqualFold(file, ref)
}

func displayName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
