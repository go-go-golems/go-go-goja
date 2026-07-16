package main

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/replhttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
				fields.New("allow-remote", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Acknowledge that a non-loopback listener has no built-in authentication and may expose host capabilities")),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) (retErr error) {
	settings := serveSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if err := validateServeListenAddress(settings.Addr, settings.AllowRemote); err != nil {
		return err
	}
	app, store, err := c.newApp(ctx)
	if err != nil {
		return err
	}
	// This defer runs only after group.Wait has stopped accepting requests and
	// waited for handlers. It then releases runtimes/leases before SQLite.
	defer func() {
		retErr = stderrors.Join(retErr, errors.Wrap(closeAppAndStore(app, store), "close repl server resources"))
	}()

	handler, err := replhttp.NewHandler(app, replhttp.WithHandlerLogger(log.Logger))
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              settings.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
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

func validateServeListenAddress(addr string, allowRemote bool) error {
	loopback, err := isLoopbackListenAddress(addr)
	if err != nil {
		return err
	}
	if !loopback && !allowRemote {
		return fmt.Errorf("refusing non-loopback listen address %q without --allow-remote; the REPL server has no built-in authentication and enabled modules may expose host capabilities", addr)
	}
	return nil
}

func isLoopbackListenAddress(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false, fmt.Errorf("invalid listen address %q: %w", addr, err)
	}
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true, nil
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback(), nil
}
