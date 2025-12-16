package bump

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	BumpCmd.PersistentFlags().BoolVarP(&preRelease, "pre-release", "p", false, "use a pre-relelease version")
	BumpCmd.PersistentFlags().StringP("tag", "t", "", "manually specify a tag")
	BumpCmd.PersistentFlags().StringP("component", "c", "minor", "the version component to bump (major, minor, patch)")
}

var (
	preRelease bool

	BumpCmd = &cobra.Command{
		Use:   "bump",
		Short: "returns the versions for the specified apps/packages, bumped by 'component'",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := app.State.Config
			for i, p := range cfg.Projects {
				fmt.Printf("project: %v\n type: %v\n", i, p.Type)
				fmt.Printf("    dependencies: %v\n", p.DependsOn)
			}
		},
	}
)
