// Package export serializes a jsdoc DocStore to multiple output formats.
package export

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/exportmd"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/exportsq"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

type Format string

const (
	FormatJSON     Format = "json"
	FormatYAML     Format = "yaml"
	FormatSQLite   Format = "sqlite"
	FormatMarkdown Format = "markdown"
)

type Shape string

const (
	ShapeStore Shape = "store"
	ShapeFiles Shape = "files"
)

type Options struct {
	Format Format

	// Shape controls JSON/YAML output: either the full store (default) or just store.Files.
	Shape Shape

	// Indent controls JSON pretty-printing. If empty, JSON is compact.
	Indent string

	// TOCDepth controls Markdown ToC depth. 0 means "use default".
	TOCDepth int
}

func Export(ctx context.Context, store *model.DocStore, w io.Writer, opts Options) error {
	if store == nil {
		return errors.New("store is nil")
	}
	if w == nil {
		return errors.New("writer is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	switch opts.Format {
	case FormatJSON:
		return exportJSON(store, w, opts)
	case FormatYAML:
		return exportYAML(store, w, opts)
	case FormatMarkdown:
		return exportmd.Write(ctx, store, w, exportmd.Options{TOCDepth: opts.TOCDepth})
	case FormatSQLite:
		return exportsq.Write(ctx, store, w, exportsq.Options{})
	default:
		return errors.Errorf("unknown format %q", opts.Format)
	}
}

func exportValue(store *model.DocStore, shape Shape) any {
	switch shape {
	case ShapeFiles:
		return store.Files
	case "", ShapeStore:
		fallthrough
	default:
		return store
	}
}

func exportJSON(store *model.DocStore, w io.Writer, opts Options) error {
	v := exportValue(store, opts.Shape)

	if opts.Indent == "" {
		enc := json.NewEncoder(w)
		return errors.Wrap(enc.Encode(v), "encoding json")
	}

	b, err := json.MarshalIndent(v, "", opts.Indent)
	if err != nil {
		return errors.Wrap(err, "encoding json")
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return errors.Wrap(err, "writing json")
}

func exportYAML(store *model.DocStore, w io.Writer, opts Options) error {
	v := exportValue(store, opts.Shape)
	b, err := yaml.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "encoding yaml")
	}
	// yaml.Marshal doesn't guarantee a trailing newline; normalize for nicer CLI output.
	if len(b) == 0 || b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	_, err = io.Copy(w, bytes.NewReader(b))
	return errors.Wrap(err, "writing yaml")
}
