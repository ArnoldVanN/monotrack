package cmd

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(compareCmd)
}

var (
	compareCmd = &cobra.Command{
		Use:   "compare",
		Short: "List which projects changed between commits",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := cmd.InheritedFlags().Lookup("base")
			head := cmd.InheritedFlags().Lookup("head")

			changes, err := versioning.ListChangedProjectNamesBetweenCommits(base.Value.String(), head.Value.String())
			if err != nil {
				return err
			}

			for _, c := range changes {
				proj := app.State.Projects[c]
				fmt.Printf("%v\n", proj.Path())
			}
			return nil
		},
	}
)
