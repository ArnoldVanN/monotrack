package versioning

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/utils"
	"github.com/arnoldvann/monotrack/internal/versioning/conventional"
	"golang.org/x/mod/semver"
)

type BumpKind string

const (
	MajorBump BumpKind = "major"
	MinorBump BumpKind = "minor"
	PatchBump BumpKind = "patch"
)

type BumpResult struct {
	Project    projects.Project
	OldVersion string
	NewVersion string
	Kind       BumpKind
	Commits    []conventional.ParsedCommit
}

type VersionBumper struct {
}

func NewBumper() VersionBumper {
	return VersionBumper{}
}

// BumpProjects bumps versions for projects that have changed since their last
// tag. If kindOverride is non-nil, it is applied uniformly. Otherwise the bump
// kind for each project is derived from its conventional commit history.
// Tags are not created or pushed; call Finalize after.
func (b *VersionBumper) BumpProjects(
	p map[string]projects.Project,
	kindOverride *BumpKind,
	preRelease bool,
	head string,
) ([]BumpResult, error) {
	cfg := app.State.Config
	projectToTags, err := git.GetTagsForProjects(cfg, p)
	if err != nil {
		return nil, err
	}

	for name, project := range p {
		if _, ok := projectToTags[project]; !ok {
			projectToTags[project] = []string{cfg.TagFor(name, "v0.0.0")}
		}
	}

	projectToLatest, err := utils.GetLatestTagPerProject(cfg, projectToTags)
	if err != nil {
		return nil, err
	}

	zeroTagProjects := make(map[string]string)
	for proj, t := range projectToLatest {
		if strings.HasPrefix(t, "v0.0.0") {
			zeroTagProjects[proj] = t
			delete(projectToLatest, proj)
		}
	}

	changed, err := getChangedProjectsVersions(projectToLatest, head)
	if err != nil {
		return nil, err
	}

	for name, version := range zeroTagProjects {
		changed[name] = changedProject{version: version, base: ""}
	}

	results := make([]BumpResult, 0, len(changed))
	for name, info := range changed {
		proj, ok := p[name]
		if !ok {
			return nil, fmt.Errorf("invalid project name: %q", name)
		}

		commits, err := commitsForProject(info.base, head, proj)
		if err != nil {
			return nil, err
		}
		parsed := parseAll(commits)

		var kind BumpKind
		if kindOverride != nil {
			kind = *kindOverride
		} else {
			kind = deriveKind(parsed)
		}

		newVer, err := bumpVersion(info.version, kind, preRelease)
		if err != nil {
			return nil, err
		}

		results = append(results, BumpResult{
			Project:    proj,
			OldVersion: info.version,
			NewVersion: newVer,
			Kind:       kind,
			Commits:    parsed,
		})
	}

	return results, nil
}

// FinalizePropose commits files onto the release branch and force-pushes
// it. Caller must have written files to disk first and ensured the working
// tree is otherwise clean.
func (b *VersionBumper) FinalizePropose(releaseBranch, baseBranch, message string, files []string) error {
	if releaseBranch == "" {
		return fmt.Errorf("empty release branch")
	}
	if len(files) == 0 {
		return fmt.Errorf("no files to commit")
	}

	if err := git.CheckoutBranchFrom(releaseBranch, baseBranch); err != nil {
		return err
	}
	defer func() { _ = git.CheckoutBranch(baseBranch) }()

	if err := git.AddFiles(files); err != nil {
		return fmt.Errorf("staging release files: %w", err)
	}
	commitSha, err := git.CommitPaths(message, files)
	if err != nil {
		return fmt.Errorf("committing release files: %w", err)
	}
	return git.ForcePushBranch(releaseBranch, commitSha)
}

// Finalize creates and pushes tags for the bumped projects. When branch is
// non-empty, the current branch tip (containing the changelog commit) is
// pushed atomically with the tags. On a non-fast-forward rejection (a
// concurrent push to the same branch), the changelog commit is rebased onto
// the new remote tip, the tags are re-pointed at the rebased commit, and the
// push is retried.
func (b *VersionBumper) Finalize(results []BumpResult, head string, branch string) error {
	if len(results) == 0 {
		return nil
	}

	tags := make([]string, 0, len(results))
	for _, r := range results {
		tag := app.State.Config.TagFor(r.Project.Name(), r.NewVersion)
		tags = append(tags, tag)

		exists, err := git.TagExistsOnRemote(tag)
		if err != nil {
			return fmt.Errorf("checking remote tag %q failed: %w", tag, err)
		}
		if exists {
			return fmt.Errorf("remote tag already exists: %s", tag)
		}
	}

	if err := createTagsAt(tags, head); err != nil {
		return err
	}

	const maxAttempts = 3
	for attempt := 1; ; attempt++ {
		err := git.PushRefsAtomic(buildRefs(tags, branch))
		if err == nil {
			return nil
		}
		if branch == "" || !errors.Is(err, git.ErrNonFastForward) || attempt >= maxAttempts {
			return err
		}

		if err := git.FetchBranch(branch); err != nil {
			return fmt.Errorf("after non-fast-forward push: %w", err)
		}
		newHead, err := git.RebaseOnto("origin/" + branch)
		if err != nil {
			return fmt.Errorf("after non-fast-forward push: %w", err)
		}
		for _, tag := range tags {
			if err := git.DeleteLocalTag(tag); err != nil {
				return err
			}
		}
		if err := createTagsAt(tags, newHead); err != nil {
			return err
		}
	}
}

func createTagsAt(tags []string, commit string) error {
	for _, tag := range tags {
		if err := git.CreateTag(tag, fmt.Sprintf("Release %s", tag), commit); err != nil {
			return err
		}
	}
	return nil
}

func buildRefs(tags []string, branch string) []string {
	refs := make([]string, 0, len(tags)+1)
	if branch != "" {
		refs = append(refs, "refs/heads/"+branch)
	}
	for _, tag := range tags {
		refs = append(refs, "refs/tags/"+tag)
	}
	return refs
}

type changedProject struct {
	version string
	base    string
}

func getChangedProjectsVersions(p map[string]string, head string) (map[string]changedProject, error) {
	changedProjects := make(map[string]changedProject)

	for proj, version := range p {
		base, err := git.GetBase(proj + "/" + version)
		if err != nil {
			return nil, err
		}

		changed, err := ListProjectsChangedBetweenCommits(base, head)
		if err != nil {
			return nil, err
		}

		if changed[proj] {
			changedProjects[proj] = changedProject{version: version, base: base}
		}
	}

	return changedProjects, nil
}

func commitsForProject(base, head string, proj projects.Project) ([]git.RawCommit, error) {
	if base == "" {
		return nil, nil
	}

	var paths []string
	if cfg, ok := app.State.Config.Projects[proj.Name()]; ok && cfg.Path != "" && cfg.Path != "." {
		paths = []string{cfg.Path}
	}

	raw, err := git.LogBetween(base, head, paths)
	if err != nil {
		return nil, err
	}

	filtered := raw[:0]
	for _, c := range raw {
		if isReleaseCommit(c.Message) {
			continue
		}
		filtered = append(filtered, c)
	}
	return filtered, nil
}

// isReleaseCommit returns true for commits monotrack itself authored when
// auto-committing a changelog, so they don't trigger empty re-releases.
func isReleaseCommit(message string) bool {
	subject, _, _ := strings.Cut(message, "\n")
	return strings.HasPrefix(strings.TrimSpace(subject), "chore(release)")
}

func parseAll(raw []git.RawCommit) []conventional.ParsedCommit {
	out := make([]conventional.ParsedCommit, 0, len(raw))
	for _, r := range raw {
		out = append(out, conventional.ParseAll(r.Hash, r.Message)...)
	}
	return out
}

func deriveKind(commits []conventional.ParsedCommit) BumpKind {
	switch conventional.DeriveBumpKind(commits) {
	case conventional.KindMajor:
		return MajorBump
	case conventional.KindMinor:
		return MinorBump
	default:
		return PatchBump
	}
}

func bumpVersion(version string, kind BumpKind, preRelease bool) (string, error) {
	if !semver.IsValid(version) {
		return "", fmt.Errorf("invalid semver: %q", version)
	}

	v := strings.TrimPrefix(version, "v")

	parts := strings.SplitN(v, "-", 2)
	core := parts[0]
	pre := ""
	if len(parts) == 2 {
		pre = parts[1]
	}

	nums := strings.Split(core, ".")
	if len(nums) != 3 {
		return "", fmt.Errorf("expected 3 version components")
	}

	major, _ := strconv.Atoi(nums[0])
	minor, _ := strconv.Atoi(nums[1])
	patch, _ := strconv.Atoi(nums[2])

	switch {
	case preRelease && pre != "":
		rcParts := strings.Split(pre, ".")
		if len(rcParts) == 2 {
			if n, err := strconv.Atoi(rcParts[1]); err == nil {
				pre = fmt.Sprintf("%s.%d", rcParts[0], n+1)
			} else {
				pre = fmt.Sprintf("%s.1", rcParts[0])
			}
		} else {
			pre = fmt.Sprintf("%s.1", rcParts[0])
		}
	case !preRelease && pre != "":
		pre = ""
	default:
		switch kind {
		case MajorBump:
			major++
			minor = 0
			patch = 0
		case MinorBump:
			minor++
			patch = 0
		case PatchBump:
			patch++
		default:
			return "", fmt.Errorf("unknown bump kind: %s", kind)
		}

		if preRelease {
			pre = "rc.1"
		} else {
			pre = ""
		}
	}

	newVersion := fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	if pre != "" {
		newVersion = fmt.Sprintf("%s-%s", newVersion, pre)
	}

	return newVersion, nil
}
