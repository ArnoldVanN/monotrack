package tag

import (
	"encoding/json"
	"fmt"
	"log"

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

type Output struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

var (
	preRelease bool
	tag        string
	component  string
	out        string

	bumpCmd = &cobra.Command{
		Use:   "bump",
		Short: "Returns the bumped versions for the specified apps/packages if they changed, bumped by 'component'",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var base string
			if len(args) == 1 {
				base = args[0]
			} else {
				base = "" // will be handleled in VersionBumper
			}

			bumper := versioning.NewBumper()

			kind, err := parseBumpKind(component)
			if err != nil {
				return err
			}

			bumped, err := bumper.BumpProjects(app.State.Projects, kind, preRelease, base)
			if err != nil {
				return err
			}

			if out == "json" {
				o := make([]Output, 0, len(bumped))

				for p, v := range bumped {
					o = append(o, Output{
						Name:    p.Name(),
						Path:    p.Path(),
						Version: v,
					})
				}

				b, err := json.Marshal(o)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Println(string(b))
			} else {
				for p, v := range bumped {
					fmt.Println(p.Name() + "/" + v)
				}
			}

			return nil
		},
	}
)

func parseBumpKind(s string) (versioning.BumpKind, error) {
	switch versioning.BumpKind(s) {
	case versioning.MajorBump, versioning.MinorBump, versioning.PatchBump:
		return versioning.BumpKind(s), nil
	default:
		return "", fmt.Errorf("invalid bump kind: %q", s)
	}
}
