package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arnoldvann/monotrack/internal/discovery"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/spf13/cobra"
)

var initDry bool

func init() {
	initCmd.Flags().BoolVar(&initDry, "dry", false, "print the generated config to stdout instead of writing any files")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Monotrack",
	Long: `Initialize Monotrack.

Writes a config listing every project found in the repository: a directory with
a go.mod, package.json or Chart.yaml becomes a project of that type. Ignored
files, node workspace roots and packaged helm subcharts are skipped. dependsOn
is filled from detected dependencies, and every project nothing else depends
on is marked as an entrypoint.

An existing config is never overwritten; use --dry to preview what would be
generated for it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := EnsureRepoRoot(); err != nil {
			return err
		}

		if initDry {
			contents, err := generateConfig()
			if err != nil {
				return err
			}
			fmt.Print(string(contents))
			return nil
		}

		if _, err := os.Stat(cfgFile); err == nil {
			fmt.Printf("%v already exists\n", cfgFile)
		} else {
			contents, err := generateConfig()
			if err != nil {
				return err
			}
			if err := createFileIfMissing(cfgFile, contents); err != nil {
				return fmt.Errorf("config file: %w", err)
			}
		}

		if err := createFileIfMissing(manifest, defaultManifestContents()); err != nil {
			return fmt.Errorf("manifest file: %w", err)
		}

		return nil
	},
}

// generateConfig discovers the repository's projects and renders them as a
// config, reporting what it skipped on stderr. It falls back to the example
// config when nothing is found.
func generateConfig() ([]byte, error) {
	root, err := git.GetRepoRoot()
	if err != nil {
		return nil, err
	}
	files, err := git.ListFiles()
	if err != nil {
		return nil, err
	}

	res, err := discovery.Discover(os.DirFS(root), files, discovery.Options{RootName: filepath.Base(root)})
	if err != nil {
		return nil, err
	}
	for _, n := range res.Notes {
		fmt.Fprintln(os.Stderr, n)
	}

	if len(res.Projects) == 0 {
		fmt.Fprintln(os.Stderr, "no go.mod, package.json or Chart.yaml found; writing an example config to edit by hand")
		return defaultConfigContents(), nil
	}

	cfg, _, err := discovery.Generate(root, res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		fmt.Fprintln(os.Stderr, "the config has no dependsOn and marks every project as an entrypoint; fix the error, then run `monotrack deps sync` and review the entrypoints")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("generated config is invalid: %w", err)
	}

	fmt.Fprintf(os.Stderr, "found %d project(s)\n", len(cfg.Projects))
	return discovery.Render(cfg)
}

func createFileIfMissing(path string, contents []byte) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("%v already exists\n", path)
		return nil // file exists, do nothing
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.WriteFile(path, contents, 0644)
}

func defaultConfigContents() []byte {
	return []byte(`projects:
  frontend-example:
    type: node
    path: apps/frontend
  backend:
    type: go
    path: apps/backend
    dependsOn:
      - shared-package
  shared-package:
    type: go
    path: packages/some-shared-package
`)
}

func defaultManifestContents() []byte {
	return []byte(`# Managed by monotrack.
# Commit this file so CI can read it across runs.
pending: []
`)
}
