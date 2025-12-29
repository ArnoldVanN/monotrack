package tag

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/spf13/cobra"
)

func init() {
	bumpCmd.Flags().BoolVarP(&preRelease, "pre-release", "p", false, "use a pre-relelease version")
	bumpCmd.Flags().StringVarP(&tag, "tag", "t", "", "manually specify a tag")
	bumpCmd.Flags().StringVarP(&component, "component", "c", "patch", "the version component to bump (major, minor, patch)")
	bumpCmd.Flags().StringVarP(&out, "out", "o", "plain", "output format (plain, json)")
}

var (
	preRelease bool
	tag        string
	component  string
	out        string

	bumpCmd = &cobra.Command{
		TraverseChildren: true,
		Use:              "bump",
		Short:            "returns the versions for the specified apps/packages, bumped by 'component'",
		RunE: func(cmd *cobra.Command, args []string) error {
			bumper := versioning.NewBumper()

			kind, err := ParseBumpKind(component)
			if err != nil {
				return fmt.Errorf("invalid component")
			}

			bumped, err := bumper.BumpProjects(app.State.Projects, kind, preRelease)
			if err != nil {
				return err
			}

			for p, v := range bumped {
				fmt.Println(p + "/" + v)
			}

			return nil
		},
	}
)

func ParseBumpKind(s string) (versioning.BumpKind, error) {
	switch versioning.BumpKind(s) {
	case versioning.MajorBump, versioning.MinorBump, versioning.PatchBump:
		return versioning.BumpKind(s), nil
	default:
		return "", fmt.Errorf("invalid bump kind: %q", s)
	}
}
