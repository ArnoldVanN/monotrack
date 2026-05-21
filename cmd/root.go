package cmd

import (
	"fmt"
	"os"

	"github.com/arnoldvann/monotrack/cmd/tag"
	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/config"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/spf13/cobra"
)

var (
	version string
	date    string

	cfgFile  string
	manifest string
	proj     []string
	out      string

	base string
	head string

	rootCmd = &cobra.Command{
		Short: "A tool for versioning applications and packages in a monorepo",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "init" || cmd.Name() == "version" {
				return nil // skip
			}

			if err := EnsureRepoRoot(); err != nil {
				return err
			}

			shallow, err := git.IsShallowRepo()
			if err != nil {
				return err
			}

			if shallow {
				return fmt.Errorf("shallow repository detected, fetch full history before running")
			}

			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return err
			}

			projectsFlag, err := cmd.Root().PersistentFlags().GetStringSlice("projects")
			if err != nil {
				return err
			}

			p, err := projects.BuildProjects(cfg, projectsFlag)
			if err != nil {
				return err
			}

			app.Init(cfg, p)

			return nil
		},
	}
)

func EnsureRepoRoot() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	root, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("error getting repo root: %v", err)
	}

	if cwd != root {
		return fmt.Errorf("please run this command from the repository root: %s", root)
	}
	return nil
}

func Execute(v string, _ string, d string) error {
	version = v
	date = d
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "monotrack.yaml", "config file")
	rootCmd.PersistentFlags().StringVarP(&manifest, "manifest", "m", ".monotrack-manifest.yaml", "manifest containing projects/tags")
	rootCmd.PersistentFlags().StringSliceVar(&proj, "projects", make([]string, 0), "projects to include in operation")
	rootCmd.PersistentFlags().StringVarP(&out, "out", "o", "plain", "output format (plain, json)")

	rootCmd.PersistentFlags().StringVar(&base, "base", "", "base commit SHA")
	rootCmd.PersistentFlags().StringVar(&head, "head", "", "head commit SHA")

	rootCmd.AddCommand(tag.TagCmd)
}
