package projects

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/spf13/cobra"
)

func init() {
	checkCmd.SilenceUsage = true
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Fail if the repository has projects the config does not list",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := printer.Format(cmd, "plain", "json")
		if err != nil {
			return err
		}

		_, res, err := discover(false)
		if err != nil {
			return err
		}

		names := slices.Sorted(maps.Keys(res.Projects))
		if format == "json" {
			o := make([]printer.Output, 0, len(names))
			for _, name := range names {
				pc := res.Projects[name]
				o = append(o, printer.Output{Name: name, Path: pc.Path, Type: string(pc.Type)})
			}
			b, err := json.Marshal(o)
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		} else if len(names) == 0 {
			fmt.Println("every project is configured")
		} else {
			for _, name := range names {
				pc := res.Projects[name]
				fmt.Printf("+ %s (%s, %s)\n", name, pc.Type, pc.Path)
			}
		}

		if len(names) > 0 {
			return fmt.Errorf("%d project(s) missing from the config; run `monotrack projects sync`, or list them under discovery.exclude", len(names))
		}
		return nil
	},
}
