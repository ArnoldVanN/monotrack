package cmd

import (
	"errors"
	"fmt"
	"os"

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

	cfgFile  string
	manifest string
	projects []string

	rootCmd = &cobra.Command{
		Short: "A tool for versioning applications and packages in a monorepo",
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

	rootCmd.AddCommand(tag.TagCmd)
	rootCmd.AddCommand(bump.BumpCmd)
}
