package deps

import (
	"fmt"
	"slices"
	"strings"

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

// outFormat resolves -o against the formats a subcommand actually implements,
// so an unsupported value errors instead of silently falling back to plain.
func outFormat(cmd *cobra.Command, supported ...string) (string, error) {
	format := "plain"
	if f := cmd.InheritedFlags().Lookup("out"); f != nil && f.Value.String() != "" {
		format = f.Value.String()
	}
	if slices.Contains(supported, format) {
		return format, nil
	}
	return "", fmt.Errorf("unsupported output format %q, want one of: %s", format, strings.Join(supported, ", "))
}
