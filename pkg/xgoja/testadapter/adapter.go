package testadapter

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/spf13/cobra"
)

func Build(ctx context.Context, host *app.Host) (*cobra.Command, error) {
	_ = ctx
	root := &cobra.Command{
		Use:   "testadapter",
		Short: "Fixture adapter root for xgoja generated adapter-mode tests",
	}
	root.AddCommand(&cobra.Command{
		Use:   "fixture",
		Short: "Print a fixture response from the adapter target package",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "testadapter fixture")
			return err
		},
	})
	host.AttachDefaultCommands(root)
	return root, nil
}
