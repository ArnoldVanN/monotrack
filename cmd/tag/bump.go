package tag

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/changelog"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/printer"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/spf13/cobra"
)

func init() {
	bumpCmd.Flags().BoolVarP(&preRelease, "pre-release", "p", false, "use a pre-relelease version")
	bumpCmd.Flags().StringVarP(&component, "component", "c", "", "force a version component to bump (major, minor, patch). When unset, the bump kind is derived per-project from conventional commit messages")
	bumpCmd.Flags().BoolVar(&dry, "dry", false, "Run the command without making any changes")
	bumpCmd.Flags().BoolVar(&singleChangelog, "single-changelog", false, "write a single root CHANGELOG.md grouped by project instead of one per project")
	bumpCmd.Flags().BoolVar(&noChangelog, "no-changelog", false, "skip changelog generation")
	bumpCmd.Flags().BoolVar(&noCommitChangelog, "no-commit-changelog", false, "write changelog files but don't commit them; tags will point at the original HEAD")
}

var (
	preRelease        bool
	component         string
	dry               bool
	singleChangelog   bool
	noChangelog       bool
	noCommitChangelog bool

	bumpCmd = &cobra.Command{
		Use:   "bump",
		Short: "Bumps specified entrypoint tags. Defaults to v0.0.1 if no tag exists",
		RunE: func(cmd *cobra.Command, args []string) error {
			headFlag := cmd.InheritedFlags().Lookup("head")
			outFlag := cmd.InheritedFlags().Lookup("out")

			head := headFlag.Value.String()
			if head == "" {
				h, err := git.GetHead()
				if err != nil {
					return err
				}
				head = h
			}

			commitChangelog := !noCommitChangelog && !noChangelog

			var branch string
			if commitChangelog && !dry {
				b, err := git.CurrentBranch()
				if err != nil {
					return err
				}
				if b == "" {
					return fmt.Errorf("cannot commit changelog on a detached HEAD; check out a branch or pass --no-commit-changelog")
				}
				branch = b
			}

			bumper := versioning.NewBumper()

			var override *versioning.BumpKind
			if cmd.Flags().Changed("component") {
				kind, err := parseBumpKind(component)
				if err != nil {
					return err
				}
				override = &kind
			}

			results, err := bumper.BumpProjects(app.State.Projects, override, preRelease, head)
			if err != nil {
				return err
			}

			entries := make([]changelog.Entry, 0, len(results))
			now := time.Now().UTC()
			for _, r := range results {
				entries = append(entries, changelog.Entry{
					Project:    r.Project,
					OldVersion: r.OldVersion,
					NewVersion: r.NewVersion,
					Date:       now,
					Commits:    r.Commits,
				})
			}

			changelogPaths := map[string]string{}
			writtenPaths := []string{}
			if !noChangelog && len(entries) > 0 {
				if dry {
					fmt.Print(changelog.RenderForPrint(entries))
				} else if singleChangelog {
					path := "CHANGELOG.md"
					if err := changelog.Combined(path, entries); err != nil {
						return fmt.Errorf("writing combined changelog: %w", err)
					}
					for _, e := range entries {
						changelogPaths[e.Project.Name()] = path
					}
					writtenPaths = append(writtenPaths, path)
				} else {
					if err := changelog.PerProject(entries); err != nil {
						return fmt.Errorf("writing changelogs: %w", err)
					}
					for _, e := range entries {
						p := filepath.Join(e.Project.Path(), "CHANGELOG.md")
						changelogPaths[e.Project.Name()] = p
						writtenPaths = append(writtenPaths, p)
					}
				}
			}

			writtenPaths = dedupe(writtenPaths)

			if commitChangelog && len(writtenPaths) > 0 {
				msg := buildReleaseMessage(results)
				if dry {
					fmt.Println("--- would commit ---")
					fmt.Println(msg)
					fmt.Println("--- would push (atomic) ---")
					fmt.Println("refs/heads/" + currentBranchForPrint())
					for _, r := range results {
						fmt.Printf("refs/tags/%s/%s\n", r.Project.Name(), r.NewVersion)
					}
				} else {
					if err := git.AddFiles(writtenPaths); err != nil {
						return fmt.Errorf("staging changelog files: %w", err)
					}
					newHead, err := git.CommitPaths(msg, writtenPaths)
					if err != nil {
						return fmt.Errorf("committing changelog: %w", err)
					}
					head = newHead
				}
			}

			if !dry {
				var extraRefs []string
				if commitChangelog && len(writtenPaths) > 0 {
					extraRefs = []string{"refs/heads/" + branch}
				}
				if err := bumper.Finalize(results, head, extraRefs); err != nil {
					return err
				}
			}

			if outFlag.Value.String() == "json" {
				o := make([]printer.BumpOutput, 0, len(results))
				for _, r := range results {
					o = append(o, printer.BumpOutput{
						Output: printer.Output{
							Name: r.Project.Name(),
							Path: r.Project.Path(),
							Type: string(r.Project.GetType()),
						},
						Version:       r.NewVersion,
						BumpKind:      string(r.Kind),
						ChangelogPath: changelogPaths[r.Project.Name()],
					})
				}

				b, err := json.Marshal(o)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Println(string(b))
			} else {
				for _, r := range results {
					fmt.Println(r.Project.Name() + "/" + r.NewVersion)
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

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func buildReleaseMessage(results []versioning.BumpResult) string {
	sorted := make([]versioning.BumpResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Project.Name() < sorted[j].Project.Name()
	})

	subject := fmt.Sprintf("chore(release): bump %d project(s)", len(sorted))

	var body strings.Builder
	for _, r := range sorted {
		fmt.Fprintf(&body, "- %s %s -> %s\n", r.Project.Name(), r.OldVersion, r.NewVersion)
	}
	return subject + "\n\n" + body.String()
}

// currentBranchForPrint is used only in dry-run output where we may not have
// validated branch state (e.g. detached HEAD with --dry). It returns "<branch>"
// as a placeholder if detection fails so the printout is still readable.
func currentBranchForPrint() string {
	b, err := git.CurrentBranch()
	if err != nil || b == "" {
		return "<branch>"
	}
	return b
}
