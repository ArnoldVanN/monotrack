package projects

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/discovery"
	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/spf13/cobra"
)

func init() {
	syncCmd.Flags().BoolVar(&dry, "dry", false, "print the projects that would be added without writing the config file")
	syncCmd.SilenceUsage = true
}

var (
	dry bool

	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Add the projects the config does not list",
		Long: `Add the projects the config does not list.

New projects get their dependsOn from detection, and are entrypoints when
nothing else depends on them. Configured projects are left alone, even when
they now depend on a new project; run "deps sync" to record that. The rest of
the document, comments included, is preserved.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := printer.Format(cmd, "plain"); err != nil {
				return err
			}

			root, res, err := discover(true)
			if err != nil {
				return err
			}
			if len(res.Projects) == 0 {
				fmt.Println("every project is configured")
				return nil
			}

			// Detect over the whole config, so a new project's dependencies
			// on configured ones are found too.
			merged := *app.State.Config
			merged.Projects = maps.Clone(app.State.Config.Projects)
			maps.Copy(merged.Projects, res.Projects)
			added := slices.Sorted(maps.Keys(res.Projects))

			notes, err := discovery.Complete(root, &merged, added)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", err)
				fmt.Fprintln(os.Stderr, "new projects are added without dependsOn and as entrypoints; fix the error, then run `monotrack deps sync` and review the entrypoints")
			}

			for _, name := range added {
				printAdded(name, merged.Projects[name])
			}
			for _, n := range notes {
				fmt.Println(n)
			}
			if dry {
				return nil
			}

			if err := merged.Validate(); err != nil {
				return fmt.Errorf("config would be invalid: %w", err)
			}

			path := cmd.InheritedFlags().Lookup("config").Value.String()
			src, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %q: %w", path, err)
			}
			out, err := discovery.AddProjects(src, &merged, added)
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

func printAdded(name string, pc projects.ProjectConfig) {
	fmt.Printf("+ %s (%s, %s)", name, pc.Type, pc.Path)
	if pc.Build.Entrypoint {
		fmt.Print(" entrypoint")
	}
	if len(pc.DependsOn) > 0 {
		fmt.Printf(" dependsOn %v", pc.DependsOn)
	}
	fmt.Println()
}
