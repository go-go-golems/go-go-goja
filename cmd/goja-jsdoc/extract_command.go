package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
)

type extractCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*extractCommand)(nil)

type extractSettings struct {
	File       string `glazed:"file"`
	Pretty     bool   `glazed:"pretty"`
	OutputFile string `glazed:"output-file"`
}

func newExtractCommand() (*extractCommand, error) {
	desc := cmds.NewCommandDescription(
		"extract",
		cmds.WithShort("Extract docs from a JavaScript file and emit JSON"),
		cmds.WithLong(`Parse a single JavaScript file and extract documentation metadata from:
- __package__({...})
- __doc__(...)
- __example__(...)
- doc tagged-template prose blocks (tag name: "doc")

Output is JSON for parity with the original jsdocex CLI.`),
		cmds.WithFlags(
			fields.New("file", fields.TypeString, fields.WithHelp("Path to the input .js file")),
			fields.New("pretty", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Pretty-print JSON output")),
			fields.New("output-file", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional output file path (defaults to stdout)")),
		),
	)
	return &extractCommand{CommandDescription: desc}, nil
}

func (c *extractCommand) Run(_ context.Context, vals *values.Values) error {
	settings := extractSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if settings.File == "" {
		return errors.Errorf("--file is required")
	}

	src, err := os.ReadFile(settings.File)
	if err != nil {
		return errors.Wrapf(err, "reading %s", settings.File)
	}
	fd, err := extract.ParseSource(settings.File, src)
	if err != nil {
		return err
	}

	var out *os.File
	if settings.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(settings.OutputFile), 0o755); err != nil {
			return errors.Wrap(err, "create output directory")
		}
		f, err := os.Create(settings.OutputFile)
		if err != nil {
			return errors.Wrap(err, "create output file")
		}
		defer func() { _ = f.Close() }()
		out = f
	} else {
		out = os.Stdout
	}

	enc := json.NewEncoder(out)
	if settings.Pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(fd); err != nil {
		return errors.Wrap(err, "encode json")
	}
	if settings.OutputFile != "" {
		fmt.Fprintf(os.Stderr, "wrote %s\n", settings.OutputFile)
	}
	return nil
}
