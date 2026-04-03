package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/go-go-golems/go-go-goja/pkg/webrepl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		log.Fatal().Err(err).Msg("web repl failed")
	}
}

func newRootCommand() *cobra.Command {
	var (
		addr               string
		logLevel           string
		pluginDirs         []string
		allowPluginModules []string
	)

	cmd := &cobra.Command{
		Use:   "web-repl",
		Short: "Run the experimental web REPL with session introspection",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := configureLogger(logLevel)
			log.Logger = logger

			pluginSetup := host.NewRuntimeSetup(pluginDirs, allowPluginModules)
			builder := pluginSetup.WithBuilder(
				engine.NewBuilder().WithModules(engine.DefaultRegistryModules()),
			)
			factory, err := builder.Build()
			if err != nil {
				return errors.Wrap(err, "build engine factory")
			}

			service := replsession.NewService(factory, logger)
			handler, err := webrepl.NewHandler(service)
			if err != nil {
				return errors.Wrap(err, "build web repl handler")
			}

			srv := &http.Server{
				Addr:              addr,
				Handler:           handler,
				ReadHeaderTimeout: 5 * time.Second,
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			logger.Info().Str("addr", addr).Msg("starting web repl")
			fmt.Printf("web repl listening on http://%s\n", addr)

			group, groupCtx := errgroup.WithContext(ctx)
			group.Go(func() error {
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
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:3090", "listen address")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level: trace, debug, info, warn, error")
	cmd.Flags().StringSliceVar(&pluginDirs, "plugin-dir", nil, fmt.Sprintf("plugin directory (defaults to %s/... when omitted)", host.DefaultDiscoveryRoot()))
	cmd.Flags().StringSliceVar(&allowPluginModules, "allow-plugin-module", nil, "allow only the listed plugin module names")
	return cmd
}

func configureLogger(level string) zerolog.Logger {
	zerolog.SetGlobalLevel(parseLevel(level))
	return zerolog.New(os.Stderr).With().Timestamp().Logger()
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error", "err":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
