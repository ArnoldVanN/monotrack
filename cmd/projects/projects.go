package projects

import (
	"fmt"
	"os"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/discovery"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	ProjectsCmd.AddCommand(checkCmd)
	ProjectsCmd.AddCommand(syncCmd)
}

var ProjectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Find projects in the repository that the config does not list",
	Long: `Find projects in the repository that the config does not list.

A directory with a go.mod, package.json or Chart.yaml is a project of that
type, exactly as "monotrack init" discovers them. Ignored files, node workspace
roots and packaged helm subcharts are skipped, as is every directory matching a
discovery.exclude pattern in the config:

  discovery:
    exclude:
      - tools/**

"projects check" fails CI when a project is missing, so a new one is not
silently left out of change detection; "projects sync" adds it. Configured
projects are never changed, renamed or removed.`,
}

// discover returns the projects missing from the loaded config. With verbose
// set, what discovery skipped is reported on stderr.
func discover(verbose bool) (string, *discovery.Result, error) {
	root, err := git.GetRepoRoot()
	if err != nil {
		return "", nil, err
	}
	files, err := git.ListFiles()
	if err != nil {
		return "", nil, err
	}

	cfg := app.State.Config
	res, err := discovery.Discover(os.DirFS(root), files, discovery.Options{
		Existing: cfg.Projects,
		Exclude:  cfg.Discovery.Exclude,
	})
	if err != nil {
		return "", nil, err
	}
	if verbose {
		for _, n := range res.Notes {
			fmt.Fprintln(os.Stderr, n)
		}
	}
	return root, res, nil
}
