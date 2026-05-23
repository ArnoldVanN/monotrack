package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/utils"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(compareCmd)
}

var (
	compareCmd = &cobra.Command{
		Use:   "compare",
		Short: "List which projects changed between their latest tag and the specified commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			head := cmd.InheritedFlags().Lookup("head")

			if head.Value.String() == "" {
				return fmt.Errorf("missing head SHA")
			}

			projectToTags, err := git.GetTagsForProjects(app.State.Config, app.State.Projects)
			if err != nil {
				return err
			}

			// Set default tags if they dont exist
			for name, project := range app.State.Projects {
				if _, ok := projectToTags[project]; !ok {
					projectToTags[project] = []string{name + "/v0.0.0"}
				}
			}

			projectToLatest, err := utils.GetLatestTagPerProject(app.State.Config, projectToTags, head.Value.String())
			if err != nil {
				return err
			}

			// Diff projects with existing tags only
			zeroTagProjects := make(map[string]bool)
			for proj, t := range projectToLatest {
				// TODO: if it's a preRelease, will have to do extra checks here
				if strings.HasPrefix(t, "v0.0.0") {
					zeroTagProjects[proj] = true
					delete(projectToLatest, proj)
				}
			}

			changes, err := getChangedProjects(projectToLatest, head.Value.String())
			if err != nil {
				return err
			}

			// assume project has changed if it doesnt have tags yet
			maps.Copy(changes, zeroTagProjects)

			changedProjects := make(map[string]projects.ProjectConfig)
			for n := range changes {
				proj, ok := app.State.Config.Projects[n]
				if !ok {
					return fmt.Errorf("invalid project name: %q", n)
				}
				changedProjects[n] = proj
			}

			if out == "json" {
				o := make([]printer.Output, 0, len(changedProjects))

				for k, v := range changedProjects {
					o = append(o, printer.Output{
						Name: k,
						Path: v.Path,
						Type: string(v.Type),
					})
				}

				b, err := json.Marshal(o)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Println(string(b))
			} else {
				for k := range changedProjects {
					fmt.Printf("%s\n", k)
				}
			}
			return nil
		},
	}
)

/*
Get changed projects.
Expects a map of projects to versions.
Returns a map of all projects including dependencies and whether they changed
*/
func getChangedProjects(p map[string]string, head string) (map[string]bool, error) {
	baseCommits := make(map[string]string)

	for proj, v := range p {
		base, err := git.GetBase(proj + "/" + v)
		if err != nil {
			return nil, err
		}

		baseCommits[proj] = base
	}

	allProjects := make(map[string]bool, 0)

	for _, c := range baseCommits {
		pr, err := versioning.ListProjectsChangedBetweenCommits(c, head)
		if err != nil {
			return nil, err
		}

		maps.Copy(allProjects, pr)
	}

	return allProjects, nil
}
