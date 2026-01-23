package tag

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/spf13/cobra"
)

func init() {
	bumpCmd.Flags().BoolVarP(&preRelease, "pre-release", "p", false, "use a pre-relelease version")
	bumpCmd.Flags().StringVarP(&component, "component", "c", "patch", "the version component to bump (major, minor, patch)")
	bumpCmd.Flags().BoolVar(&dry, "dry", false, "Run the command without making any changes")
}

var (
	preRelease bool
	component  string
	dry        bool

	bumpCmd = &cobra.Command{
		Use:   "bump",
		Short: "Bumps specified entrypoint tags. Defaults to v0.0.1 if no tag exists",
		RunE: func(cmd *cobra.Command, args []string) error {
			head := cmd.InheritedFlags().Lookup("head")
			out := cmd.InheritedFlags().Lookup("out")

			bumper := versioning.NewBumper()

			kind, err := parseBumpKind(component)
			if err != nil {
				return err
			}

			bumped, err := bumper.BumpProjects(app.State.Projects, kind, preRelease, head.Value.String(), dry)
			if err != nil {
				return err
			}

			bumpedProjectsTags := make(map[projects.Project]string)
			for n, t := range bumped {
				proj, ok := app.State.Projects[n]
				if !ok {
					return fmt.Errorf("invalid project name: %q", n)
				}
				bumpedProjectsTags[proj] = t
			}

			if out.Value.String() == "json" {
				o := make([]printer.BumpOutput, 0, len(bumped))

				for p, v := range bumpedProjectsTags {
					o = append(o, printer.BumpOutput{
						Output: printer.Output{
							Name: p.Name(),
							Path: p.Path(),
							Type: string(p.GetType()),
						},
						Version: v,
					})
				}

				b, err := json.Marshal(o)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Println(string(b))
			} else {
				for p, v := range bumpedProjectsTags {
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
