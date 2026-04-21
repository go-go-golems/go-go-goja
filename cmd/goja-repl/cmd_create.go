package main

import (
	"context"
	"io"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
)

type createCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*createCommand)(nil)

func newCreateCommand(out io.Writer, opts *rootOptions) *createCommand {
	return &createCommand{
		CommandDescription: cmds.NewCommandDescription("create",
			cmds.WithShort("Create a fresh persistent REPL session"),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *createCommand) Run(ctx context.Context, vals *values.Values) error {
	return c.runWithApp(func(ctx context.Context, app *replapi.App) error {
		session, err := app.CreateSession(ctx)
		if err != nil {
			return err
		}
		return writeJSON(c.out, map[string]any{"session": session})
	})(ctx, vals)
}
