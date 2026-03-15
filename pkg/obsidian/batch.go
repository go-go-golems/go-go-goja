package obsidian

import (
	"context"

	"github.com/pkg/errors"
)

// Batch runs a callback sequentially over query results and preserves per-note errors.
func (c *Client) Batch(ctx context.Context, query *Query, fn BatchFunc) ([]BatchItemResult, error) {
	if fn == nil {
		return nil, errors.New("obsidian: batch function is nil")
	}
	if query == nil {
		query = c.Query()
	}

	notes, err := query.Run(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]BatchItemResult, 0, len(notes))
	for _, note := range notes {
		value, err := fn(ctx, note)
		ret = append(ret, BatchItemResult{
			Path:  note.Path,
			Value: value,
			Err:   err,
		})
	}
	return ret, nil
}
