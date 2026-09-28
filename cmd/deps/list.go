package deps

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func init() {
	listCmd.SilenceUsage = true
}

type listEntry struct {
	Project   string   `json:"project"`
	DependsOn []string `json:"dependsOn"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the detected internal dependencies of each project",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := outFormat(cmd, "plain", "json")
		if err != nil {
			return err
		}

		detected, err := resolve()
		if err != nil {
			return err
		}

		names := make([]string, 0, len(detected))
		for name := range detected {
			names = append(names, name)
		}
		sort.Strings(names)

		if format == "json" {
			entries := make([]listEntry, 0, len(names))
			for _, name := range names {
				d := detected[name]
				if d == nil {
					d = []string{}
				}
				entries = append(entries, listEntry{Project: name, DependsOn: d})
			}
			b, err := json.Marshal(entries)
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		for _, name := range names {
			fmt.Println(name)
			for _, d := range detected[name] {
				fmt.Printf("  - %s\n", d)
			}
		}
		return nil
	},
}
