package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/batch"
	jsdocexport "github.com/go-go-golems/go-go-goja/pkg/jsdoc/export"
)

type exportCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*exportCommand)(nil)

type exportSettings struct {
	Input           []string `glazed:"input"`
	Inputs          []string `glazed:"inputs"`
	Format          string   `glazed:"format"`
	Shape           string   `glazed:"shape"`
	OutputFile      string   `glazed:"output-file"`
	Pretty          bool     `glazed:"pretty"`
	TOCDepth        int      `glazed:"toc-depth"`
	ContinueOnError bool     `glazed:"continue-on-error"`
}

func newExportCommand() (*exportCommand, error) {
	desc := cmds.NewCommandDescription(
		"export",
		cmds.WithShort("Extract docs from one or more JS files and export in multiple formats"),
		cmds.WithLong(`Build a jsdoc store from one or more JavaScript source files and export it.

Formats:
  - json:     emits DocStore or []FileDoc (shape)
  - yaml:     emits DocStore or []FileDoc (shape)
  - markdown: emits a deterministic single-file Markdown document (with ToC)
  - sqlite:   emits a SQLite database file (binary)

Examples:
  goja-jsdoc export a.js b.js --format json --pretty
  goja-jsdoc export --input a.js --input b.js --format yaml --shape files
  goja-jsdoc export a.js --format markdown --toc-depth 3 --output-file docs.md
  goja-jsdoc export a.js b.js --format sqlite --output-file docs.sqlite`),
		cmds.WithFlags(
			fields.New("input", fields.TypeStringList, fields.WithHelp("Input .js files (repeatable)")),
			fields.New("format", fields.TypeChoice, fields.WithDefault(string(jsdocexport.FormatJSON)), fields.WithChoices(
				string(jsdocexport.FormatJSON),
				string(jsdocexport.FormatYAML),
				string(jsdocexport.FormatMarkdown),
				string(jsdocexport.FormatSQLite),
			), fields.WithHelp("Export format")),
			fields.New("shape", fields.TypeChoice, fields.WithDefault(string(jsdocexport.ShapeStore)), fields.WithChoices(
				string(jsdocexport.ShapeStore),
				string(jsdocexport.ShapeFiles),
			), fields.WithHelp("JSON/YAML shape: full store or only files")),
			fields.New("output-file", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional output file path (defaults to stdout; use '-' for stdout)")),
			fields.New("pretty", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Pretty-print JSON output (ignored for yaml/markdown/sqlite)")),
			fields.New("toc-depth", fields.TypeInteger, fields.WithDefault(3), fields.WithHelp("Markdown ToC depth (levels)")),
			fields.New("continue-on-error", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Continue building the store even if some inputs fail (prints errors to stderr)")),
		),
		cmds.WithArguments(
			fields.New("inputs", fields.TypeStringList, fields.WithHelp("Input .js files")),
		),
	)
	return &exportCommand{CommandDescription: desc}, nil
}

func (c *exportCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := exportSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}

	paths := append([]string{}, settings.Input...)
	paths = append(paths, settings.Inputs...)
	if len(paths) == 0 {
		return errors.Errorf("at least one input file is required (use args or --input)")
	}

	inputs := make([]batch.InputFile, 0, len(paths))
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		inputs = append(inputs, batch.InputFile{Path: p})
	}
	if len(inputs) == 0 {
		return errors.Errorf("no valid inputs provided")
	}

	br, err := batch.BuildStore(ctx, inputs, batch.BatchOptions{ContinueOnError: settings.ContinueOnError})
	if err != nil {
		return err
	}
	for _, be := range br.Errors {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", be.Input.Path, be.Error)
	}

	var out *os.File
	if settings.OutputFile != "" && settings.OutputFile != "-" {
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

	opts := jsdocexport.Options{
		Format:   jsdocexport.Format(settings.Format),
		Shape:    jsdocexport.Shape(settings.Shape),
		TOCDepth: settings.TOCDepth,
		Indent:   "",
	}
	if opts.Format == jsdocexport.FormatJSON && settings.Pretty {
		opts.Indent = "  "
	}

	if err := jsdocexport.Export(ctx, br.Store, out, opts); err != nil {
		return err
	}
	if settings.OutputFile != "" && settings.OutputFile != "-" {
		fmt.Fprintf(os.Stderr, "wrote %s\n", settings.OutputFile)
	}
	return nil
}
