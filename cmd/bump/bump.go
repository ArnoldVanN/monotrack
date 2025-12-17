package bump

import (
	// "fmt"
	//
	// "github.com/arnoldvann/monotrack/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	BumpCmd.PersistentFlags().BoolVarP(&preRelease, "pre-release", "p", false, "use a pre-relelease version")
	BumpCmd.PersistentFlags().StringVarP(&tag, "tag", "t", "", "manually specify a tag")
	BumpCmd.PersistentFlags().StringVarP(&component, "component", "c", "minor", "the version component to bump (major, minor, patch)")
}

var (
	preRelease bool
	tag        string
	component  string

	BumpCmd = &cobra.Command{
		Use:   "bump",
		Short: "returns the versions for the specified apps/packages, bumped by 'component'",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
)
