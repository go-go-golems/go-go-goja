package testcobra

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "testcobra",
		Short: "Fixture Cobra root for xgoja generated attach-mode tests",
	}
	root.AddCommand(&cobra.Command{
		Use:   "fixture",
		Short: "Print a fixture response from the Cobra target package",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "testcobra fixture")
			return err
		},
	})
	return root
}
