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

type restoreCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*restoreCommand)(nil)

func newRestoreCommand(out io.Writer, opts *rootOptions) *restoreCommand {
	return &restoreCommand{
		CommandDescription: cmds.NewCommandDescription("restore",
			cmds.WithShort("Replay-restore a persisted session into a live runtime"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *restoreCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	return c.runWithApp(func(ctx context.Context, app *replapi.App) error {
		summary, err := app.Restore(ctx, settings.SessionID)
		if err != nil {
			return err
		}
		return writeJSON(c.out, map[string]any{"session": summary})
	})(ctx, vals)
}
