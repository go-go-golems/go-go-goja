package main

import (
	"context"
	"io"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
)

type sessionsCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*sessionsCommand)(nil)

func newSessionsCommand(out io.Writer, opts *rootOptions) *sessionsCommand {
	return &sessionsCommand{
		CommandDescription: cmds.NewCommandDescription("sessions",
			cmds.WithShort("List durable REPL sessions"),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *sessionsCommand) Run(ctx context.Context, vals *values.Values) error {
	return c.runWithApp(func(ctx context.Context, app *replapi.App) error {
		sessions, err := app.ListSessions(ctx)
		if err != nil {
			return err
		}
		return writeJSON(c.out, map[string]any{"sessions": sessions})
	})(ctx, vals)
}
