package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedhelp "github.com/go-go-golems/glazed/pkg/help"
	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/batch"
	jsdocglazedhelp "github.com/go-go-golems/go-go-goja/pkg/jsdoc/glazedhelp"
)

type previewHelpCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*previewHelpCommand)(nil)

type previewHelpSettings struct {
	Input      []string `glazed:"input"`
	Inputs     []string `glazed:"inputs"`
	Dir        string   `glazed:"dir"`
	Recursive  bool     `glazed:"recursive"`
	Slug       string   `glazed:"slug"`
	OutputDir  string   `glazed:"output-dir"`
	ListSlugs  bool     `glazed:"list-slugs"`
	AppCommand []string `glazed:"command"`
}

func newPreviewHelpCommand() (*previewHelpCommand, error) {
	desc := cmds.NewCommandDescription(
		"preview-help",
		cmds.WithShort("Load JavaScript docs into a Glazed help system and render a bridged page"),
		cmds.WithLong(`Build a jsdoc store from JavaScript sources, bridge it into a temporary
Glazed HelpSystem, and render a selected help page using the runtime page composer.

This is the primary preview path for the jsdoc-to-Glazed bridge.`),
		cmds.WithFlags(
			fields.New("input", fields.TypeStringList, fields.WithHelp("Input .js files (repeatable)")),
			fields.New("dir", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional directory to scan for .js files")),
			fields.New("recursive", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Recursively scan --dir for .js files")),
			fields.New("slug", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Slug to render (defaults to the root bridged page)")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional directory to write generated Glazed markdown files")),
			fields.New("list-slugs", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("List bridged slugs instead of rendering a page")),
			fields.New("command", fields.TypeStringList, fields.WithHelp("Optional command names to attach to all bridged sections")),
		),
		cmds.WithArguments(
			fields.New("inputs", fields.TypeStringList, fields.WithHelp("Input .js files")),
		),
	)
	return &previewHelpCommand{CommandDescription: desc}, nil
}

func (c *previewHelpCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := previewHelpSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}

	paths := append([]string{}, settings.Input...)
	paths = append(paths, settings.Inputs...)
	if settings.Dir != "" {
		dirPaths, err := collectJSFiles(settings.Dir, settings.Recursive)
		if err != nil {
			return err
		}
		paths = append(paths, dirPaths...)
	}
	if len(paths) == 0 {
		return errors.New("at least one input file is required (use args or --input)")
	}

	uniq := uniqueSortedStrings(paths)
	inputs := make([]batch.InputFile, 0, len(uniq))
	for _, path := range uniq {
		inputs = append(inputs, batch.InputFile{Path: path})
	}

	br, err := batch.BuildStore(ctx, inputs, batch.BatchOptions{})
	if err != nil {
		return err
	}

	result, err := jsdocglazedhelp.BuildSections(br.Store, jsdocglazedhelp.Options{
		DefaultCommands: settings.AppCommand,
	})
	if err != nil {
		return err
	}

	if settings.OutputDir != "" {
		files, err := jsdocglazedhelp.BuildMarkdownFiles(result)
		if err != nil {
			return err
		}
		if err := jsdocglazedhelp.WriteMarkdownFiles(settings.OutputDir, files); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote glazed markdown to %s\n", settings.OutputDir)
	}

	if settings.ListSlugs {
		slugs := make([]string, 0, len(result.Sections))
		for _, section := range result.Sections {
			slugs = append(slugs, section.Slug)
		}
		sort.Strings(slugs)
		for _, slug := range slugs {
			fmt.Fprintln(os.Stdout, slug)
		}
		return nil
	}

	hs := glazedhelp.NewHelpSystem()
	if err := jsdocglazedhelp.LoadIntoHelpSystem(ctx, hs, result); err != nil {
		return err
	}
	hs.SetPageComposer(jsdocglazedhelp.NewComposer(br.Store, result))

	slug := strings.TrimSpace(settings.Slug)
	if slug == "" {
		slug = result.RootSlug
	}

	section, err := hs.GetSectionWithSlug(slug)
	if err != nil {
		return err
	}

	rendered, err := hs.RenderTopicHelpWithWriter(section, &glazedhelp.RenderOptions{
		Query:                 glazedhelp.NewSectionQuery().ReturnAnyOfTopics(slug).ReturnAllTypes().FilterSections(section),
		ShowAllSections:       true,
		ShowDocumentationList: true,
		HelpCommand:           "goja-jsdoc preview-help",
	}, os.Stdout)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
