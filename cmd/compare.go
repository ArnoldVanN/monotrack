package cmd

import (
	"fmt"

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
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			base := args[0]
			head := args[1]

			changes, err := versioning.ListChangedProjectsBetweenCommits(base, head)
			if err != nil {
				return err
			}

			for _, c := range changes {
				fmt.Printf("%v\n", c)
			}
			return nil
		},
	}
)
