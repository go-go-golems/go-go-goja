package main

import (
	"context"
	"io"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
)

type docsCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*docsCommand)(nil)

func newDocsCommand(out io.Writer, opts *rootOptions) *docsCommand {
	return &docsCommand{
		CommandDescription: cmds.NewCommandDescription("docs",
			cmds.WithShort("Show persisted REPL-authored docs for a session"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *docsCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	return c.runWithApp(func(ctx context.Context, app *replapi.App) error {
		docs, err := app.Docs(ctx, settings.SessionID)
		if err != nil {
			return err
		}
		return writeJSON(c.out, map[string]any{"docs": docs})
	})(ctx, vals)
}
