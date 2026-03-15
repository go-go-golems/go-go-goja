package obsidian

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/obsidiancli"
	"github.com/pkg/errors"
)

type queryMode string

const (
	queryModeFiles      queryMode = "files"
	queryModeSearch     queryMode = "search"
	queryModeOrphans    queryMode = "orphans"
	queryModeDeadEnds   queryMode = "dead-ends"
	queryModeUnresolved queryMode = "unresolved"
)

// Query composes native CLI filters with post-filters that require note content.
type Query struct {
	client *Client

	mode         queryMode
	searchTerm   string
	folder       string
	ext          string
	nameContains string
	tag          string
	limit        int
	vault        string
}

// InFolder restricts results to a folder.
func (q *Query) InFolder(folder string) *Query {
	q.folder = strings.TrimSpace(folder)
	return q
}

// WithExtension restricts results to one file extension.
func (q *Query) WithExtension(ext string) *Query {
	q.ext = normalizeExt(ext)
	return q
}

// Search switches the primary source plan to a CLI search.
func (q *Query) Search(term string) *Query {
	q.mode = queryModeSearch
	q.searchTerm = strings.TrimSpace(term)
	return q
}

// NameContains applies a basename post-filter.
func (q *Query) NameContains(substr string) *Query {
	q.nameContains = strings.ToLower(strings.TrimSpace(substr))
	return q
}

// Tagged applies a content-based tag post-filter.
func (q *Query) Tagged(tag string) *Query {
	q.tag = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tag)), "#")
	return q
}

// Limit caps the number of results.
func (q *Query) Limit(limit int) *Query {
	q.limit = limit
	return q
}

// Vault overrides the configured default vault for this query.
func (q *Query) Vault(vault string) *Query {
	q.vault = strings.TrimSpace(vault)
	return q
}

// Orphans switches the source plan to orphan notes.
func (q *Query) Orphans() *Query {
	q.mode = queryModeOrphans
	return q
}

// DeadEnds switches the source plan to dead-end notes.
func (q *Query) DeadEnds() *Query {
	q.mode = queryModeDeadEnds
	return q
}

// Unresolved switches the source plan to unresolved links.
func (q *Query) Unresolved() *Query {
	q.mode = queryModeUnresolved
	return q
}

// Run executes the planned query.
func (q *Query) Run(ctx context.Context) ([]*Note, error) {
	if q == nil || q.client == nil {
		return nil, errors.New("obsidian: query client is nil")
	}
	paths, err := q.runPaths(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*Note, 0, len(paths))
	for _, path := range paths {
		ret = append(ret, q.client.noteForPath(path))
	}
	return ret, nil
}

func (q *Query) runPaths(ctx context.Context) ([]string, error) {
	paths, err := q.nativePaths(ctx)
	if err != nil {
		return nil, err
	}
	paths, err = q.postFilterPaths(ctx, paths)
	if err != nil {
		return nil, err
	}
	if q.limit > 0 && len(paths) > q.limit {
		paths = paths[:q.limit]
	}
	return paths, nil
}

func (q *Query) nativePaths(ctx context.Context) ([]string, error) {
	if q.mode == "" {
		q.mode = queryModeFiles
	}

	switch q.mode {
	case queryModeFiles:
		return q.client.Files(ctx, FileListOptions{
			Folder: q.folder,
			Ext:    q.ext,
			Limit:  q.limit,
			Vault:  q.vault,
		})
	case queryModeSearch:
		return q.client.Search(ctx, q.searchTerm, SearchOptions{
			Folder: q.folder,
			Ext:    q.ext,
			Limit:  q.limit,
			Vault:  q.vault,
		})
	case queryModeOrphans:
		return q.runLineList(ctx, obsidiancli.SpecLinksOrphans)
	case queryModeDeadEnds:
		return q.runLineList(ctx, obsidiancli.SpecLinksDeadEnds)
	case queryModeUnresolved:
		result, err := q.client.runner.Run(ctx, obsidiancli.SpecLinksUnresolved, obsidiancli.CallOptions{
			Vault:      q.vault,
			Parameters: callParameters(q.nativeFilterParameters()),
		})
		if err != nil {
			return nil, err
		}
		return extractPaths(result.Parsed), nil
	}

	return nil, errors.Errorf("obsidian: unsupported query mode %q", q.mode)
}

func (q *Query) postFilterPaths(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	ret := make([]string, 0, len(paths))
	for _, path := range paths {
		if q.nameContains != "" {
			name := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
			if !strings.Contains(name, q.nameContains) {
				continue
			}
		}
		if q.tag != "" {
			note := q.client.noteForPath(path)
			tags, err := note.Tags(ctx)
			if err != nil {
				return nil, err
			}
			if !containsFolded(tags, q.tag) {
				continue
			}
		}
		ret = append(ret, path)
	}
	return ret, nil
}

func (q *Query) runLineList(ctx context.Context, spec obsidiancli.CommandSpec) ([]string, error) {
	result, err := q.client.runner.Run(ctx, spec, obsidiancli.CallOptions{
		Vault:      q.vault,
		Parameters: callParameters(q.nativeFilterParameters()),
	})
	if err != nil {
		return nil, err
	}
	return resultStrings(result)
}

func (q *Query) nativeFilterParameters() map[string]any {
	return map[string]any{
		"folder": q.folder,
		"ext":    q.ext,
		"limit":  q.limit,
	}
}

func containsFolded(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimPrefix(value, "#"), needle) {
			return true
		}
	}
	return false
}
