package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/server"
)

type serveCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*serveCommand)(nil)

type serveSettings struct {
	Dir  string `glazed:"dir"`
	Host string `glazed:"host"`
	Port int    `glazed:"port"`
}

func newServeCommand() (*serveCommand, error) {
	desc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Start the doc browser web server for a directory of JS files"),
		cmds.WithLong(`Parse all .js files in a directory into a doc store, start the web UI and JSON API,
and watch for changes (SSE live reload).

This command uses Glazed for command/flag definitions only.`),
		cmds.WithFlags(
			fields.New("dir", fields.TypeString, fields.WithDefault("."), fields.WithHelp("Directory containing .js files to parse (non-recursive initial parse)")),
			fields.New("host", fields.TypeString, fields.WithDefault("127.0.0.1"), fields.WithHelp("HTTP bind host")),
			fields.New("port", fields.TypeInteger, fields.WithDefault(8080), fields.WithHelp("HTTP bind port")),
		),
	)
	return &serveCommand{CommandDescription: desc}, nil
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := serveSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}

	store := model.NewDocStore()
	docs, err := extract.ParseDir(settings.Dir)
	if err != nil {
		return err
	}
	for _, fd := range docs {
		store.AddFile(fd)
	}
	fmt.Printf("Loaded %d files from %s\n", len(docs), settings.Dir)

	srv := server.New(store, settings.Dir, settings.Host, settings.Port)

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	return srv.Run(runCtx)
}
