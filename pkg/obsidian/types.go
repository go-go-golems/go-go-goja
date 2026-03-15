package obsidian

import (
	"context"

	"github.com/go-go-golems/go-go-goja/pkg/obsidiancli"
)

// Runner is the subprocess-backed transport required by the high-level client.
type Runner interface {
	Run(ctx context.Context, spec obsidiancli.CommandSpec, call obsidiancli.CallOptions) (obsidiancli.Result, error)
}

// Config configures the high-level Obsidian client.
type Config struct {
	CLI   obsidiancli.Config
	Cache *Cache
}

// FileListOptions describe native file listing filters.
type FileListOptions struct {
	Folder string
	Ext    string
	Limit  int
	Vault  string
}

// SearchOptions describe native CLI search filters.
type SearchOptions struct {
	Folder string
	Ext    string
	Limit  int
	Vault  string
}

// CreateOptions configure note creation.
type CreateOptions struct {
	Content  string
	Folder   string
	Template string
	Vault    string
}

// DeleteOptions configure note deletion behavior.
type DeleteOptions struct {
	Permanent bool
	Vault     string
}

// BatchFunc applies work to each note returned by a query.
type BatchFunc func(ctx context.Context, note *Note) (any, error)

// BatchItemResult captures one sequential batch callback result.
type BatchItemResult struct {
	Path  string
	Value any
	Err   error
}
