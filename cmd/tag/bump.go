package tag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/changelog"
	"github.com/arnoldvann/monotrack/internal/forge"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/manifest"
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
	bumpCmd.Flags().BoolVar(&noPR, "no-pr", false, "skip the PR-based release flow; commit the changelog and push tags directly to the current branch")
	bumpCmd.Flags().StringVar(&releaseBranchFlag, "release-branch", "", "release branch name (overrides config release.branch; '{base}' is replaced with the base branch name)")
}

var (
	preRelease        bool
	component         string
	dry               bool
	singleChangelog   bool
	noChangelog       bool
	noCommitChangelog bool
	noPR              bool
	releaseBranchFlag string

	bumpCmd = &cobra.Command{
		Use:   "bump",
		Short: "Bumps specified entrypoint tags. Defaults to v0.0.1 if no tag exists",
		RunE:  runBump,
	}
)

// runBump routes between the direct (--no-pr / --dry), tag-phase, and
// propose-phase flows. Tag phase wins over propose when both could apply: if
// the manifest has unpushed pending tags, push them first so a concurrent new
// commit doesn't get folded into the same release proposal.
func runBump(cmd *cobra.Command, args []string) error {
	headFlag := cmd.InheritedFlags().Lookup("head")
	outFlag := cmd.InheritedFlags().Lookup("out")
	manifestFlag := cmd.InheritedFlags().Lookup("manifest")

	head := headFlag.Value.String()
	if head == "" {
		h, err := git.GetHead()
		if err != nil {
			return err
		}
		head = h
	}

	jsonOut := outFlag.Value.String() == "json"
	manifestPath := manifestFlag.Value.String()

	if noPR || dry {
		return runDirectBump(cmd, head, jsonOut)
	}

	baseBranch, err := git.CurrentBranch()
	if err != nil {
		return err
	}
	if baseBranch == "" {
		return fmt.Errorf("PR flow requires a checked-out branch; use --no-pr for detached HEAD")
	}

	m, err := manifest.Read(manifestPath)
	if err != nil {
		return err
	}

	if missing, err := missingPendingTags(m); err != nil {
		return err
	} else if len(missing) > 0 {
		return runTagPhase(missing, head, jsonOut)
	}

	return runProposePhase(cmd, head, baseBranch, manifestPath, jsonOut)
}

func runDirectBump(cmd *cobra.Command, head string, jsonOut bool) error {
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

	override, err := overrideFromFlag(cmd)
	if err != nil {
		return err
	}

	results, err := bumper.BumpProjects(app.State.Projects, override, preRelease, head)
	if err != nil {
		return err
	}

	entries := buildEntries(results)

	changelogPaths, writtenPaths, err := writeChangelogs(entries)
	if err != nil {
		return err
	}

	if commitChangelog && len(writtenPaths) > 0 {
		msg := buildReleaseMessage(results)
		if dry {
			fmt.Fprintln(os.Stderr, "--- would commit ---")
			fmt.Fprintln(os.Stderr, msg)
			fmt.Fprintln(os.Stderr, "--- would push (atomic) ---")
			fmt.Fprintln(os.Stderr, "refs/heads/"+currentBranchForPrint())
			for _, r := range results {
				fmt.Fprintf(os.Stderr, "refs/tags/%s/%s\n", r.Project.Name(), r.NewVersion)
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
		pushBranch := ""
		if commitChangelog && len(writtenPaths) > 0 {
			pushBranch = branch
		}
		if err := bumper.Finalize(results, head, pushBranch); err != nil {
			return err
		}
	}

	return emitResults(results, changelogPaths, "", "", jsonOut)
}

func runProposePhase(cmd *cobra.Command, head, baseBranch, manifestPath string, jsonOut bool) error {
	dirty, err := git.HasUncommittedChanges()
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("propose phase requires a clean working tree; commit or stash your changes, or use --no-pr")
	}

	bumper := versioning.NewBumper()
	override, err := overrideFromFlag(cmd)
	if err != nil {
		return err
	}
	results, err := bumper.BumpProjects(app.State.Projects, override, preRelease, head)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return emitResults(nil, nil, "", "", jsonOut)
	}

	entries := buildEntries(results)
	changelogPaths, writtenPaths, err := writeChangelogs(entries)
	if err != nil {
		return err
	}

	m := &manifest.Manifest{Pending: make([]manifest.PendingEntry, 0, len(results))}
	for _, r := range results {
		m.Pending = append(m.Pending, manifest.PendingEntry{
			Project:    r.Project.Name(),
			Tag:        app.State.Config.TagFor(r.Project.Name(), r.NewVersion),
			OldVersion: r.OldVersion,
			NewVersion: r.NewVersion,
		})
	}
	if err := manifest.Write(manifestPath, m); err != nil {
		return err
	}
	files := append([]string{manifestPath}, writtenPaths...)
	files = dedupe(files)

	releaseBranch := resolveReleaseBranch(baseBranch)
	msg := buildReleaseMessage(results)
	if err := bumper.FinalizePropose(releaseBranch, baseBranch, msg, files); err != nil {
		return err
	}

	prURL, err := ensurePR(releaseBranch, baseBranch, results)
	if err != nil {
		return fmt.Errorf("release branch %q pushed but PR creation failed: %w", releaseBranch, err)
	}

	return emitResults(results, changelogPaths, "proposed", prURL, jsonOut)
}

func runTagPhase(missing []manifest.PendingEntry, head string, jsonOut bool) error {
	results := make([]versioning.BumpResult, 0, len(missing))
	for _, p := range missing {
		proj, ok := app.State.Projects[p.Project]
		if !ok {
			return fmt.Errorf("manifest references unknown project %q", p.Project)
		}
		results = append(results, versioning.BumpResult{
			Project:    proj,
			OldVersion: p.OldVersion,
			NewVersion: p.NewVersion,
		})
	}

	bumper := versioning.NewBumper()
	if err := bumper.Finalize(results, head, ""); err != nil {
		return err
	}

	return emitResults(results, nil, "released", "", jsonOut)
}

func missingPendingTags(m *manifest.Manifest) ([]manifest.PendingEntry, error) {
	if !m.HasPending() {
		return nil, nil
	}
	missing := make([]manifest.PendingEntry, 0, len(m.Pending))
	for _, p := range m.Pending {
		exists, err := git.TagExistsOnRemote(p.Tag)
		if err != nil {
			return nil, err
		}
		if !exists {
			missing = append(missing, p)
		}
	}
	return missing, nil
}

func ensurePR(head, base string, results []versioning.BumpResult) (string, error) {
	f, err := forge.Detect()
	if err != nil {
		return "", err
	}
	title, body := buildPRTitleBody(results)
	res, err := f.EnsurePR(context.Background(), forge.PRRequest{
		Base: base, Head: head, Title: title, Body: body,
	})
	if err != nil {
		return "", err
	}
	if res.URL != "" {
		action := "updated"
		if res.Created {
			action = "opened"
		}
		fmt.Fprintf(os.Stderr, "release PR %s: %s\n", action, res.URL)
	}
	return res.URL, nil
}

func buildPRTitleBody(results []versioning.BumpResult) (string, string) {
	title := fmt.Sprintf("chore(release): bump %d project(s)", len(results))
	sorted := make([]versioning.BumpResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Project.Name() < sorted[j].Project.Name() })

	var body strings.Builder
	body.WriteString("Automated release proposal from monotrack.\n\n")
	body.WriteString("| Project | Old | New |\n")
	body.WriteString("|---|---|---|\n")
	for _, r := range sorted {
		fmt.Fprintf(&body, "| %s | %s | %s |\n", r.Project.Name(), r.OldVersion, r.NewVersion)
	}
	body.WriteString("\nMerge this PR to publish the listed versions. To override changelog wording for a commit, edit the source commit/PR body with a `BEGIN_COMMIT_OVERRIDE` ... `END_COMMIT_OVERRIDE` block before merging.\n")
	return title, body.String()
}

// resolveReleaseBranch applies the --release-branch flag override on top of
// the config value and substitutes {base}.
func resolveReleaseBranch(base string) string {
	cfg := app.State.Config.Release
	if releaseBranchFlag != "" {
		cfg.Branch = releaseBranchFlag
	}
	return cfg.ResolveReleaseBranch(base)
}

func overrideFromFlag(cmd *cobra.Command) (*versioning.BumpKind, error) {
	if !cmd.Flags().Changed("component") {
		return nil, nil
	}
	kind, err := parseBumpKind(component)
	if err != nil {
		return nil, err
	}
	return &kind, nil
}

func buildEntries(results []versioning.BumpResult) []changelog.Entry {
	now := time.Now().UTC()
	out := make([]changelog.Entry, 0, len(results))
	for _, r := range results {
		out = append(out, changelog.Entry{
			Project:    r.Project,
			OldVersion: r.OldVersion,
			NewVersion: r.NewVersion,
			Date:       now,
			Commits:    r.Commits,
		})
	}
	return out
}

func writeChangelogs(entries []changelog.Entry) (map[string]string, []string, error) {
	changelogPaths := map[string]string{}
	var writtenPaths []string

	if noChangelog || len(entries) == 0 {
		return changelogPaths, writtenPaths, nil
	}
	if dry {
		fmt.Fprint(os.Stderr, changelog.RenderForPrint(entries))
		return changelogPaths, writtenPaths, nil
	}
	cfg := app.State.Config
	useSingle := singleChangelog || cfg.IsSingleProject()
	if useSingle {
		path := cfg.ChangelogPath()
		if err := changelog.Combined(path, entries); err != nil {
			return nil, nil, fmt.Errorf("writing combined changelog: %w", err)
		}
		for _, e := range entries {
			changelogPaths[e.Project.Name()] = path
		}
		writtenPaths = append(writtenPaths, path)
	} else {
		if err := changelog.PerProject(entries); err != nil {
			return nil, nil, fmt.Errorf("writing changelogs: %w", err)
		}
		for _, e := range entries {
			p := filepath.Join(e.Project.Path(), "CHANGELOG.md")
			changelogPaths[e.Project.Name()] = p
			writtenPaths = append(writtenPaths, p)
		}
	}
	return changelogPaths, dedupe(writtenPaths), nil
}

func emitResults(results []versioning.BumpResult, changelogPaths map[string]string, status, prURL string, jsonOut bool) error {
	if jsonOut {
		o := make([]printer.BumpOutput, 0, len(results))
		for _, r := range results {
			o = append(o, printer.BumpOutput{
				Output: printer.Output{
					Name: r.Project.Name(),
					Path: r.Project.Path(),
					Type: string(r.Project.GetType()),
				},
				Version:       r.NewVersion,
				OldVersion:    r.OldVersion,
				BumpKind:      string(r.Kind),
				ChangelogPath: changelogPaths[r.Project.Name()],
				Status:        status,
				PRUrl:         prURL,
			})
		}
		b, err := json.Marshal(o)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
		return nil
	}

	for _, r := range results {
		fmt.Println(app.State.Config.TagFor(r.Project.Name(), r.NewVersion))
	}
	return nil
}

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

// currentBranchForPrint returns a "<branch>" placeholder when detection
// fails — dry runs may be on a detached HEAD and we still want readable
// output.
func currentBranchForPrint() string {
	b, err := git.CurrentBranch()
	if err != nil || b == "" {
		return "<branch>"
	}
	return b
}
