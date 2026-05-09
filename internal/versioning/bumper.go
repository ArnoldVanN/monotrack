package versioning

import (
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
	projectToTags, err := git.GetTagsForProjects(p)
	if err != nil {
		return nil, err
	}

	for name, project := range p {
		if _, ok := projectToTags[project]; !ok {
			projectToTags[project] = []string{name + "/v0.0.0"}
		}
	}

	projectToLatest, err := utils.GetLatestTagPerProject(projectToTags)
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

// Finalize creates and pushes tags for the bumped projects, atomically with
// any extra refs (e.g. a branch ref when also pushing a release commit).
func (b *VersionBumper) Finalize(results []BumpResult, head string, extraRefs []string) error {
	if len(results) == 0 {
		return nil
	}

	tagRefs := make([]string, 0, len(results))
	tags := make([]string, 0, len(results))
	for _, r := range results {
		tag := fmt.Sprintf("%s/%s", r.Project.Name(), r.NewVersion)
		tags = append(tags, tag)
		tagRefs = append(tagRefs, "refs/tags/"+tag)

		exists, err := git.TagExistsOnRemote(tag)
		if err != nil {
			return fmt.Errorf("checking remote tag %q failed: %w", tag, err)
		}
		if exists {
			return fmt.Errorf("remote tag already exists: %s", tag)
		}
	}

	for _, tag := range tags {
		if err := git.CreateTag(tag, fmt.Sprintf("Release %s", tag), head); err != nil {
			return err
		}
	}

	return git.PushRefsAtomic(append(extraRefs, tagRefs...))
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
		out = append(out, conventional.Parse(r.Hash, r.Message))
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

// TODO: custom prefixes and separators
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

	if preRelease && pre != "" {
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
	} else {
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
