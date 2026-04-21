package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/replhttp"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type serveCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*serveCommand)(nil)

func newServeCommand(out io.Writer, opts *rootOptions) *serveCommand {
	return &serveCommand{
		CommandDescription: cmds.NewCommandDescription("serve",
			cmds.WithShort("Run the persistent REPL JSON server"),
			cmds.WithFlags(
				fields.New("addr", fields.TypeString, fields.WithDefault("127.0.0.1:3090"), fields.WithHelp("Listen address")),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := serveSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	handler, err := replhttp.NewHandler(app)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              settings.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	group, groupCtx := errgroup.WithContext(runCtx)
	group.Go(func() error {
		fmt.Fprintf(c.out, "goja-repl server listening on http://%s\n", settings.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	group.Go(func() error {
		<-groupCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})
	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return errors.Wrap(err, "run http server")
	}
	return nil
}
