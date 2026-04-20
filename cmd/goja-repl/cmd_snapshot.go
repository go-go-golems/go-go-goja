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

type snapshotCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*snapshotCommand)(nil)

func newSnapshotCommand(out io.Writer, opts *rootOptions) *snapshotCommand {
	return &snapshotCommand{
		CommandDescription: cmds.NewCommandDescription("snapshot",
			cmds.WithShort("Load the current live snapshot for a session"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *snapshotCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	return c.runWithApp(func(ctx context.Context, app *replapi.App) error {
		summary, err := app.Snapshot(ctx, settings.SessionID)
		if err != nil {
			return err
		}
		return writeJSON(c.out, map[string]any{"session": summary})
	})(ctx, vals)
}
