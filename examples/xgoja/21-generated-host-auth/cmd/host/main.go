package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/go-go-goja/examples/xgoja/21-generated-host-auth/internal/xgojaruntime"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	"github.com/spf13/cobra"
)

func main() {
	var configureErr error
	bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
		ConfigureServices: func(services *app.HostServices) {
			configureErr = services.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.BuilderOptions{
				Config: defaultAuthConfig(),
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

func defaultAuthConfig() hostauth.Config {
	return hostauth.Config{
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
}
