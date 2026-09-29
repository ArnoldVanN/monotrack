package tag

import (
	"fmt"
	"maps"
	"slices"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.SilenceUsage = true
}

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all tags",
		Long:  "Lists the tags for the specified projects. Expects tags to contain the same project names as defined in the configuration file. For example 'frontend-v1.2.3' or 'backend/v1.2.3'",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := printer.Format(cmd, "plain"); err != nil {
				return err
			}

			tags, err := git.GetTagsForProjects(app.State.Config, app.State.Projects)
			if err != nil {
				return err
			}

			for _, name := range slices.Sorted(maps.Keys(app.State.Projects)) {
				for _, t := range tags[app.State.Projects[name]] {
					fmt.Println(t)
				}
			}
			return nil
		},
	}
)
