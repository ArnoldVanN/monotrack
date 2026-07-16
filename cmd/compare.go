package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/utils"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/spf13/cobra"
)

func init() {
	compareCmd.Flags().BoolVar(&unreleased, "unreleased", false, "list projects with changes since their own latest tag, instead of comparing a base..head range")
	rootCmd.AddCommand(compareCmd)
}

var (
	unreleased bool

	compareCmd = &cobra.Command{
		Use:   "compare",
		Short: "List which projects changed between a base and a head",
		Long: `List which projects changed between a base and a head.

By default the range runs from the default branch to your working tree,
including uncommitted and untracked (non-ignored) files:

  monotrack compare                        # origin/main -> working tree
  monotrack compare --base origin/feat-x   # unpushed work vs your remote branch
  monotrack compare --base <sha> --head <sha>

The base is anchored at its merge base with the head, so commits landed on the
base branch after you diverged are not reported as your changes.

--unreleased instead reports, per project, whether it changed since its own
latest tag. "what has not been released yet" instead of "what did this range
touch".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			headFlag := cmd.InheritedFlags().Lookup("head")
			base := cmd.InheritedFlags().Lookup("base")

			if !unreleased {
				changes, err := compareRange(base.Value.String(), headFlag.Value.String())
				if err != nil {
					return err
				}
				return emitChanged(changes)
			}

			// Same default as `tag bump`: measure up to the current commit.
			head := headFlag.Value.String()
			if head == "" {
				h, err := git.GetHead()
				if err != nil {
					return err
				}
				head = h
			}

			projectToTags, err := git.GetTagsForProjects(app.State.Config, app.State.Projects)
			if err != nil {
				return err
			}

			// Set default tags if they dont exist
			for name, project := range app.State.Projects {
				if _, ok := projectToTags[project]; !ok {
					projectToTags[project] = []string{name + "/v0.0.0"}
				}
			}

			// Use the latest *reachable* tag as the changelog/diff baseline:
			// when the absolute latest is orphan-tagged the diff range explodes,
			// so we fall back to the highest reachable tag (or to "no base" if
			// none, treating the project as freshly-bootstrapped).
			projectToLatest, err := utils.GetLatestReachableTagPerProject(app.State.Config, projectToTags, head)
			if err != nil {
				return err
			}

			// Diff projects with existing tags only
			zeroTagProjects := make(map[string]bool)
			for proj, t := range projectToLatest {
				// TODO: if it's a preRelease, will have to do extra checks here
				if strings.HasPrefix(t, "v0.0.0") {
					zeroTagProjects[proj] = true
					delete(projectToLatest, proj)
				}
			}
			// Projects whose only tags are orphaned (no reachable tag) are
			// absent from projectToLatest; treat them as changed/bootstrap.
			for name := range app.State.Projects {
				if _, ok := projectToLatest[name]; ok {
					continue
				}
				zeroTagProjects[name] = true
			}

			changes, err := getChangedProjects(projectToLatest, head)
			if err != nil {
				return err
			}

			// assume project has changed if it doesnt have tags yet
			maps.Copy(changes, zeroTagProjects)

			return emitChanged(changes)
		},
	}
)

// emitChanged renders the changed set in the requested output format.
func emitChanged(changes map[string]bool) error {
	changedProjects := make(map[string]projects.ProjectConfig)
	for n := range changes {
		proj, ok := app.State.Config.Projects[n]
		if !ok {
			return fmt.Errorf("invalid project name: %q", n)
		}
		changedProjects[n] = proj
	}

	if out == "json" {
		o := make([]printer.Output, 0, len(changedProjects))

		for k, v := range changedProjects {
			o = append(o, printer.Output{
				Name: k,
				Path: v.Path,
				Type: string(v.Type),
			})
		}

		b, err := json.Marshal(o)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(b))
	} else {
		for k := range changedProjects {
			fmt.Printf("%s\n", k)
		}
	}
	return nil
}

// compareRange flags projects that changed between base and head, plus their
// transitive dependents. Empty base means the default branch; empty head means
// the working tree. The base is anchored at its merge base with head, so the
// base branch's own later commits aren't attributed to head.
func compareRange(base, head string) (map[string]bool, error) {
	if base == "" {
		b, err := git.DefaultBranch()
		if err != nil {
			return nil, err
		}
		base = b
	}

	// The working tree hangs off HEAD.
	headRev := head
	if headRev == "" {
		headRev = "HEAD"
	}
	mergeBase, err := git.MergeBase(base, headRev)
	if err != nil {
		return nil, err
	}

	var diff []string
	if head == "" {
		diff, err = git.DiffWorkingTree(mergeBase)
	} else {
		diff, err = git.GitDiff(mergeBase, head)
	}
	if err != nil {
		return nil, err
	}

	_, all := versioning.ListProjectsChangedInDiff(diff)

	changed := make(map[string]bool, len(all))
	for name := range app.State.Projects {
		if all[name] {
			changed[name] = true
		}
	}
	return changed, nil
}

// Get changed projects.
// Expects a map of projects to versions.
// Returns a map of all projects including dependencies and whether they changed
func getChangedProjects(p map[string]string, head string) (map[string]bool, error) {
	baseCommits := make(map[string]string)

	for proj, v := range p {
		base, err := git.GetBase(proj + "/" + v)
		if err != nil {
			return nil, err
		}

		baseCommits[proj] = base
	}

	changedByBase := make(map[string]map[string]bool)
	allProjects := make(map[string]bool, 0)

	for proj, c := range baseCommits {
		pr, ok := changedByBase[c]
		if !ok {
			_, all, err := versioning.ListProjectsChangedBetweenCommits(c, head)
			if err != nil {
				return nil, err
			}
			changedByBase[c] = all
			pr = all
		}

		if pr[proj] {
			allProjects[proj] = true
		}
	}

	return allProjects, nil
}
