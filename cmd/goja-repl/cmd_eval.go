package main

import (
	"io"
	"context"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type evalCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*evalCommand)(nil)

func newEvalCommand(out io.Writer, opts *rootOptions) *evalCommand {
	return &evalCommand{
		CommandDescription: cmds.NewCommandDescription("eval",
			cmds.WithShort("Evaluate source in a persistent session"),
			cmds.WithFlags(
				fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id")),
				fields.New("source", fields.TypeString, fields.WithRequired(true), fields.WithHelp("JavaScript source to evaluate")),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *evalCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := evalSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	return c.runWithApp(func(ctx context.Context, app *replapi.App) error {
		resp, err := app.Evaluate(ctx, settings.SessionID, settings.Source)
		if err != nil {
			return err
		}
		return writeJSON(c.out, resp)
	})(ctx, vals)
}
