package changelog

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/changelog"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func init() {
	ChangelogCmd.Flags().StringVar(&since, "since", "", "only regenerate releases tagged after this ref (a tag, or e.g. HEAD~50); older sections in the file are kept")
	ChangelogCmd.Flags().BoolVar(&write, "write", false, "write the changelog files instead of printing the sections")
	ChangelogCmd.Flags().BoolVar(&skipPre, "skip-prereleases", false, "leave out prerelease sections, as for a flow that only writes changelogs on stable releases")
	ChangelogCmd.Flags().BoolVar(&single, "single-changelog", false, "write a single root CHANGELOG.md grouped by project instead of one per project")
	ChangelogCmd.SilenceUsage = true
}

var (
	since   string
	write   bool
	single  bool
	skipPre bool
)

var ChangelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Regenerate changelogs from the tag and commit history",
	Long: `Regenerate changelogs from the tag and commit history.

Every tag reachable from head becomes a section, covering the commits since
the project's previous tag, as "tag bump" would have written it. Commit
overrides (BEGIN_COMMIT_OVERRIDE) added since are picked up.

The current config is used for every release, and a release without commits
lists the dependencies that changed at project granularity, so history from
before a config change can differ from what was written at the time.

Without --since every section is regenerated and the files are rewritten.
With --since only releases tagged after that ref are regenerated, and the
other sections already in the file are kept as they are. With --projects
and a combined changelog, only those projects' sections are regenerated.
Sections are ordered by date, newest first.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		head := cmd.InheritedFlags().Lookup("head").Value.String()
		if head == "" {
			h, err := git.GetHead()
			if err != nil {
				return err
			}
			head = h
		}

		releases, err := versioning.History(app.State.Projects, head)
		if err != nil {
			return err
		}
		if skipPre {
			releases = slices.DeleteFunc(releases, func(r versioning.Release) bool { return semver.Prerelease(r.NewVersion) != "" })
		}
		if since != "" {
			old, err := git.TagsMergedInto(since)
			if err != nil {
				return err
			}
			releases = slices.DeleteFunc(releases, func(r versioning.Release) bool { return old[r.Tag] })
		}

		cfg := app.State.Config
		combined := single || cfg.IsSingleProject()
		files := map[string][]changelog.Entry{}
		for _, r := range releases {
			path := cfg.ChangelogPath()
			if !combined {
				path = filepath.Join(r.Project.Path(), "CHANGELOG.md")
			}
			files[path] = append(files[path], changelog.Entry{
				Project:      r.Project,
				OldVersion:   r.OldVersion,
				NewVersion:   r.NewVersion,
				Date:         r.Date.UTC(),
				Reason:       r.Reason,
				Commits:      r.Commits,
				Dependencies: r.Dependencies,
			})
		}
		if combined {
			for _, entries := range files {
				slices.SortStableFunc(entries, func(a, b changelog.Entry) int {
					if c := b.Date.Compare(a.Date); c != 0 {
						return c
					}
					return strings.Compare(a.Project.Name(), b.Project.Name())
				})
			}
		}

		// A section already in a file is kept when it's older than --since,
		// or when it belongs to a project left out by --projects, which only
		// happens in a combined file. Otherwise the file is regenerated whole.
		partial := cmd.InheritedFlags().Changed("projects")
		keep := func(key string) bool {
			if since != "" {
				return true
			}
			if !partial || !combined {
				return false
			}
			_, selected := app.State.Projects[changelog.SectionProject(key)]
			return key == "" || !selected
		}

		for _, path := range slices.Sorted(maps.Keys(files)) {
			sections := changelog.Sections(files[path], combined)
			if !write {
				fmt.Printf("--- %s ---\n", path)
				for _, s := range sections {
					fmt.Print(s.Body)
				}
				continue
			}

			existing, err := os.ReadFile(path)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			content := changelog.Splice(string(existing), sections, keep)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}
			fmt.Println(path)
		}
		return nil
	},
}
