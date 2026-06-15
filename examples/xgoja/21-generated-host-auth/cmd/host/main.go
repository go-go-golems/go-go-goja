package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/go-go-goja/examples/xgoja/21-generated-host-auth/internal/xgojaruntime"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	"github.com/spf13/cobra"
)

func main() {
	authConfig, err := authConfigFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var configureErr error
	bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
		ConfigureServices: func(services *app.HostServices) {
			configureErr = services.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.BuilderOptions{
				Config: authConfig,
			}))
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if configureErr != nil {
		fmt.Fprintln(os.Stderr, configureErr)
		os.Exit(1)
	}

	root := &cobra.Command{
		Use:          "generated-host-auth",
		Short:        "Serve the generated-host auth xgoja example",
		SilenceUsage: true,
	}
	bundle.AttachDefaultCommands(root)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func authConfigFromEnv() (hostauth.Config, error) {
	cfg := hostauth.Config{
		Mode: hostauth.ModeDev,
		Session: hostauth.SessionConfig{
			Cookie: hostauth.CookieConfig{
				AllowInsecureHTTP: true,
			},
		},
		Stores: hostauth.StoresConfig{
			Default: hostauth.StoreConfig{
				Driver: string(hostauth.StoreDriverMemory),
			},
		},
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("XGOJA_AUTH_STORE"))) {
	case "", "memory":
		return cfg, nil
	case "sqlite":
		dsn := strings.TrimSpace(os.Getenv("XGOJA_AUTH_SQLITE_DSN"))
		if dsn == "" {
			return hostauth.Config{}, fmt.Errorf("XGOJA_AUTH_SQLITE_DSN is required when XGOJA_AUTH_STORE=sqlite")
		}
		applySchema := true
		cfg.Stores.Default = hostauth.StoreConfig{
			Driver:      string(hostauth.StoreDriverSQLite),
			DSN:         dsn,
			ApplySchema: &applySchema,
		}
		return cfg, nil
	default:
		return hostauth.Config{}, fmt.Errorf("unsupported XGOJA_AUTH_STORE %q (want memory or sqlite)", os.Getenv("XGOJA_AUTH_STORE"))
	}
}
