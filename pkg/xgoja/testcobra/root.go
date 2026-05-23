package testcobra

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "testcobra",
		Short: "Fixture Cobra root for xgoja generated attach-mode tests",
	}
}
