package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Monotrack",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := createFileIfMissing(cfgFile, defaultConfigContents()); err != nil {
			return fmt.Errorf("config file: %w", err)
		}

		if err := createFileIfMissing(manifest, defaultManifestContents()); err != nil {
			return fmt.Errorf("manifest file: %w", err)
		}

		return nil
	},
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
	return []byte(`projects:
  example: TODO
`)
}
