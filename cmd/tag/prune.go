package tag

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/spf13/cobra"
)

func init() {
	pruneCmd.SilenceUsage = true
	pruneCmd.Flags().BoolVar(&pruneApply, "apply", false, "actually delete the tags; without this flag prune only prints what it would delete")
	pruneCmd.Flags().BoolVar(&pruneLocal, "local", true, "delete matching local tags")
	pruneCmd.Flags().BoolVar(&pruneRemote, "remote", true, "delete matching tags on origin")
}

var (
	pruneApply  bool
	pruneLocal  bool
	pruneRemote bool

	pruneCmd = &cobra.Command{
		Use:   "prune",
		Short: "Delete stale prerelease tags, keeping every stable release and the latest tag per project",
		Long: "Removes prerelease tags (e.g. -rc) that sit strictly below their project's " +
			"highest tag, so the accumulated release-candidate tags from CI don't grow unbounded. " +
			"Every stable release is kept, as is the latest tag of each project; a project whose " +
			"latest tag is still a prerelease is left untouched so in-flight releases keep versioning " +
			"correctly.\n\n" +
			"Defaults to a dry run: pass --apply to actually delete.",
		RunE: runPrune,
	}
)

func runPrune(cmd *cobra.Command, args []string) error {
	jsonOut := cmd.InheritedFlags().Lookup("out").Value.String() == "json"

	plan, err := versioning.PlanPrune(app.State.Config, app.State.Projects)
	if err != nil {
		return err
	}

	if pruneApply && len(plan.Delete) > 0 {
		tags := plan.Tags()
		if pruneRemote {
			if err := git.DeleteRemoteTags(tags); err != nil {
				return err
			}
		}
		if pruneLocal {
			if err := git.DeleteLocalTags(tags); err != nil {
				return err
			}
		}
	}

	return emitPrune(plan, pruneApply, jsonOut)
}

func emitPrune(plan *versioning.PrunePlan, applied, jsonOut bool) error {
	if jsonOut {
		o := make([]printer.PruneOutput, 0, len(plan.Delete))
		for _, item := range plan.Delete {
			o = append(o, printer.PruneOutput{
				Project: item.Project,
				Tag:     item.Tag,
				Version: item.Version,
				Deleted: applied,
			})
		}
		b, err := json.Marshal(o)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	if len(plan.Delete) == 0 {
		fmt.Fprintln(os.Stderr, "nothing to prune")
		return nil
	}

	verb := "would delete"
	if applied {
		verb = "deleted"
	}
	fmt.Fprintf(os.Stderr, "%s %d tag(s):\n", verb, len(plan.Delete))
	for _, item := range plan.Delete {
		fmt.Println(item.Tag)
	}
	return nil
}
