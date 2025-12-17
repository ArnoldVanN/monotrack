package cmd

import (
	"fmt"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
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

			diff, err := git.GitDiff(base, head)
			if err != nil {
				return err
			}

			lines := strings.Split(strings.TrimSpace(diff), "\n")
			if len(lines) == 0 {
				fmt.Println("No changes detected")
				return nil
			}

			// set of parent project names
			reverseDeps := make(map[string]map[string]struct{})

			for _, p := range app.State.Projects {
				for _, d := range p.ListDependencies() {
					if reverseDeps[d.Name()] == nil {
						reverseDeps[d.Name()] = make(map[string]struct{})
					}
					reverseDeps[d.Name()][p.Name()] = struct{}{}
				}
			}

			changedMap := make(map[string]struct{})

			for _, p := range app.State.Projects {
				for _, l := range lines {
					if strings.Contains(l, p.Path()) {
						changedMap[p.Name()] = struct{}{}
						collectParents(p.Name(), reverseDeps, changedMap)
					}
				}
			}

			changed := make([]string, 0, len(changedMap))
			for k := range changedMap {
				changed = append(changed, k)
			}

			for _, c := range changed {
				fmt.Printf("%v\n", c)
			}
			return nil
		},
	}
)
