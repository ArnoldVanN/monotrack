package deps

import (
	"encoding/json"
	"fmt"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/dependencies"
	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/spf13/cobra"
)

func init() {
	checkCmd.SilenceUsage = true
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Fail if the configured dependsOn lists are out of date",
	Long: `Fail if the configured dependsOn lists are out of date.

Exits non-zero when running sync would change the config, so a stale dependency
graph is caught in CI rather than silently skipping a build. Unverified edges
are reported but do not fail the check unless --prune is set, matching what
sync would actually write.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := printer.Format(cmd, "plain", "json")
		if err != nil {
			return err
		}

		detected, err := resolve()
		if err != nil {
			return err
		}

		changes := dependencies.Diff(app.State.Config, detected)

		if format == "json" {
			if changes == nil {
				changes = []dependencies.Change{}
			}
			b, err := json.Marshal(changes)
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		} else {
			printChanges(changes)
		}

		if n := staleCount(changes); n > 0 {
			return fmt.Errorf("%d project(s) have stale dependsOn lists; run `monotrack deps sync`", n)
		}
		return nil
	},
}

func staleCount(changes []dependencies.Change) int {
	n := 0
	for _, c := range changes {
		if len(c.Added) > 0 || len(c.Redundant) > 0 || (prune && len(c.Unverified) > 0) {
			n++
		}
	}
	return n
}

func printChanges(changes []dependencies.Change) {
	var added, redundant, unverified int

	for _, c := range changes {
		fmt.Println(c.Project)
		for _, d := range c.Added {
			fmt.Printf("  + %s\n", d)
			added++
		}
		for _, r := range c.Redundant {
			fmt.Printf("  - %s (already reached via %s)\n", r.Dep, r.Via)
			redundant++
		}
		for _, d := range c.Unverified {
			if prune {
				fmt.Printf("  - %s (no detected dependency)\n", d)
			} else {
				fmt.Printf("  ! %s (no detected dependency, kept)\n", d)
			}
			unverified++
		}
	}

	if added+redundant+unverified == 0 {
		fmt.Println("dependsOn is up to date")
		return
	}

	fmt.Printf("\n%d added, %d redundant", added, redundant)
	if prune {
		fmt.Printf(", %d unverified removed\n", unverified)
		return
	}
	fmt.Printf(", %d unverified kept\n", unverified)
	if unverified > 0 {
		fmt.Println("unverified edges are dependencies detection cannot see (cross-language, loaded at runtime).")
		fmt.Println("confirm they are genuinely gone before removing them with --prune.")
	}
}
