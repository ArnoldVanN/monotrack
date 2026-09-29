package deps

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/dependencies"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	DepsCmd.PersistentFlags().BoolVar(&transitive, "transitive", false, "keep dependency edges that are already implied by a longer path")
	DepsCmd.PersistentFlags().BoolVar(&prune, "prune", false, "also drop configured edges detection found no evidence for")

	DepsCmd.AddCommand(listCmd)
	DepsCmd.AddCommand(checkCmd)
	DepsCmd.AddCommand(syncCmd)
}

var (
	transitive bool
	prune      bool
)

var DepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Detect internal dependencies from each project's native manifest",
	Long: `Detect internal dependencies from each project's native manifest.

Go projects are resolved from the transitive import graph (including test
imports), node projects from the workspace deps in package.json, and helm
charts from the subchart list in Chart.yaml. Dependencies resolving outside the
repository are ignored.

Detection feeds the dependsOn lists in monotrack.yaml, it does not replace
them: "deps check" keeps the committed lists honest in CI and "deps sync"
regenerates them, but change detection still reads the config.`,
}

func resolve() (map[string][]string, error) {
	root, err := git.GetRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("error getting repo root: %w", err)
	}
	return dependencies.Resolve(root, app.State.Config, transitive)
}
