package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arnoldvann/monotrack/cmd/bump"
	"github.com/arnoldvann/monotrack/cmd/tag"
	"github.com/arnoldvann/monotrack/config"
	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	proj "github.com/arnoldvann/monotrack/internal/projects"
	"github.com/spf13/cobra"
)

var (
	version string
	commit  string
	date    string

	cfgFile    string
	manifest   string
	projects   []string
	preRelease bool

	rootCmd = &cobra.Command{
		Use:   "monotrack [base] [head]",
		Short: "A tool for versioning applications and packages in a monorepo",
		Args:  cobra.ExactArgs(2),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "init" {
				return nil // skip
			}

			if err := EnsureRepoRoot(); err != nil {
				return err
			}

			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return err
			}

			projectsFlag, err := cmd.Root().PersistentFlags().GetStringSlice("projects")
			if err != nil {
				return err
			}

			p, err := proj.BuildProjects(cfg, projectsFlag)
			if err != nil {
				return err
			}

			app.Init(cfg, p)

			return nil
		},
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

func collectParents(
	start string,
	reverse map[string]map[string]struct{},
	out map[string]struct{},
) {
	for parent := range reverse[start] {
		if _, seen := out[parent]; seen {
			continue
		}
		out[parent] = struct{}{}
		collectParents(parent, reverse, out)
	}
}

func EnsureRepoRoot() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	root, err := git.GetRepoRoot()
	if err != nil {
		return errors.New("not inside a git repository")
	}

	if cwd != root {
		return fmt.Errorf("please run this command from the repository root: %s", root)
	}
	return nil
}

func Execute(v string, c string, d string) error {
	version = v
	commit = c
	date = d
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "monotrack.yaml", "config file")
	rootCmd.PersistentFlags().StringVarP(&manifest, "manifest", "m", ".monotrack-manifest.yaml", "manifest containing projects/tags")
	rootCmd.PersistentFlags().StringSliceVar(&projects, "projects", make([]string, 0), "projects to include in operation")
	rootCmd.PersistentFlags().BoolVarP(&preRelease, "pre-release", "p", false, "use a pre-relelease version")

	rootCmd.AddCommand(tag.TagCmd)
	rootCmd.AddCommand(bump.BumpCmd)
}
