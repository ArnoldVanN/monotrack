package tag

import (
	"fmt"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.SilenceUsage = true
}

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all tags",
		Long:  "Lists the tags for the specified projects. Expects tags to contain the same names as defined in the configuration file. For example proj-v1.2.3",
		RunE: func(cmd *cobra.Command, args []string) error {
			for p, c := range app.State.Projects {
				fmt.Printf("project: %v\n type: %v\n", p, c.GetType())
				fmt.Printf("    dependencies: %v\n", c.ListDependencies())
			}

			tags, err := git.GetTags()
			if err != nil {
				return err
			}

			lines := strings.Split(strings.TrimSpace(tags), "\n")

			if len(lines) == 0 {
				fmt.Println("No tags matching specified projects")
				return nil
			}

			out := make([]string, len(lines))

			for _, p := range app.State.Projects {
				for _, l := range lines {
					if strings.Contains(l, p.Name()) {
						out = append(out, l)
					}
				}
			}

			fmt.Println(out)
			return nil
		},
	}
)
