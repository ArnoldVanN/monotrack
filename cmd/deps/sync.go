package deps

import (
	"fmt"
	"os"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/dependencies"
	"github.com/spf13/cobra"
)

func init() {
	syncCmd.Flags().BoolVar(&dry, "dry", false, "print the changes without writing the config file")
	syncCmd.SilenceUsage = true
}

var (
	dry bool

	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Rewrite the dependsOn lists in the config to match detection",
		Long: `Rewrite the dependsOn lists in the config to match detection.

Only the dependsOn lists are touched; the rest of the document, comments
included, is preserved.

Configured edges detection cannot account for are kept, since they are usually
a blind spot in the resolver rather than a stale entry — a cross-language
dependency, or a file loaded by path at runtime. --prune removes them instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := outFormat(cmd, "plain"); err != nil {
				return err
			}

			detected, err := resolve()
			if err != nil {
				return err
			}

			changes := dependencies.Diff(app.State.Config, detected)
			printChanges(changes)

			if staleCount(changes) == 0 {
				return nil
			}
			if dry {
				return nil
			}

			path := cmd.InheritedFlags().Lookup("config").Value.String()
			src, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %q: %w", path, err)
			}

			out, err := dependencies.Sync(src, dependencies.Target(detected, changes, prune))
			if err != nil {
				return err
			}

			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, out, info.Mode().Perm()); err != nil {
				return fmt.Errorf("writing %q: %w", path, err)
			}

			fmt.Printf("updated %s\n", path)
			return nil
		},
	}
)
