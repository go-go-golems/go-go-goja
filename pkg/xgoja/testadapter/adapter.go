package testadapter

import (
	"context"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/spf13/cobra"
)

func Build(ctx context.Context, host *app.Host) (*cobra.Command, error) {
	_ = ctx
	root := &cobra.Command{
		Use:   "testadapter",
		Short: "Fixture adapter root for xgoja generated adapter-mode tests",
	}
	host.AttachDefaultCommands(root)
	return root, nil
}
