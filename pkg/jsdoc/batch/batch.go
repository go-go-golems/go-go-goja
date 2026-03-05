// Package batch builds DocStore instances from multiple inputs (paths and/or inline content).
package batch

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

// InputFile describes one JavaScript file to parse.
//
// Exactly one of Path or Content must be set.
type InputFile struct {
	// Path is a filesystem path to read and parse.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Content is inline source content to parse.
	Content []byte `json:"content,omitempty" yaml:"content,omitempty"`

	// DisplayName is used for reporting/errors when Path is empty.
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"`
}

// BatchOptions controls batch extraction behavior.
type BatchOptions struct {
	ContinueOnError bool
}

// InputSummary is a lossy (safe to serialize) view of an input.
type InputSummary struct {
	Path        string `json:"path,omitempty" yaml:"path,omitempty"`
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"`
}

// BatchError is an error associated with a particular input.
type BatchError struct {
	Input InputSummary `json:"input" yaml:"input"`
	Error string       `json:"error" yaml:"error"`
}

// BatchResult is the result of building a store from multiple inputs.
type BatchResult struct {
	Store  *model.DocStore `json:"store" yaml:"store"`
	Errors []BatchError    `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// BuildStore parses all inputs and builds a DocStore.
//
// If ContinueOnError is false, BuildStore fails fast and returns a non-nil error.
// If ContinueOnError is true, BuildStore returns a partial store with per-input errors.
func BuildStore(ctx context.Context, inputs []InputFile, opts BatchOptions) (*BatchResult, error) {
	store := model.NewDocStore()
	result := &BatchResult{Store: store}

	for i, in := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		fd, err := parseOne(in, i)
		if err != nil {
			if !opts.ContinueOnError {
				return nil, err
			}
			result.Errors = append(result.Errors, BatchError{
				Input: InputSummary{Path: in.Path, DisplayName: in.DisplayName},
				Error: err.Error(),
			})
			continue
		}
		store.AddFile(fd)
	}

	return result, nil
}

func parseOne(in InputFile, index int) (*model.FileDoc, error) {
	hasPath := in.Path != ""
	hasContent := len(in.Content) > 0

	switch {
	case hasPath && hasContent:
		return nil, errors.Errorf("input %d: both path and content are set (choose one)", index)
	case hasPath:
		return extract.ParseFile(in.Path)
	case hasContent:
		return extract.ParseSource(bestInlineName(in, index), in.Content)
	default:
		return nil, errors.Errorf("input %d: neither path nor content is set", index)
	}
}

func bestInlineName(in InputFile, index int) string {
	if in.Path != "" {
		return in.Path
	}
	if in.DisplayName != "" {
		return in.DisplayName
	}
	return fmt.Sprintf("inline:%d", index)
}
