package versioning

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"golang.org/x/mod/semver"
)

type BumpKind string

const (
	MajorBump BumpKind = "major"
	MinorBump BumpKind = "minor"
	PatchBump BumpKind = "patch"
)

type VersionBumper struct {
}

func NewBumper() VersionBumper {
	return VersionBumper{}
}

func (b *VersionBumper) BumpProjects(
	p map[string]projects.Project,
	kind BumpKind,
	preRelease bool,
	base string,
) (map[projects.Project]string, error) {
	allTags, err := git.GetTagsForProjects(p)
	if err != nil {
		return nil, err
	}

	latestTags, err := getLatestTagPerProject(allTags)
	if err != nil {
		return nil, err
	}

	changedProjects, err := getChangedProjectsToVersions(latestTags, base)
	if err != nil {
		return nil, err
	}

	if len(changedProjects) == 0 {
		return nil, nil
	}

	bumped, err := bump(changedProjects, kind, preRelease)
	if err != nil {
		return nil, err
	}

	projectVersions := make(map[projects.Project]string, len(bumped))
	for name, version := range bumped {
		p, ok := app.State.Projects[name]
		if !ok {
			return nil, fmt.Errorf("project %q not found in app state", name)
		}
		projectVersions[p] = version
	}

	return projectVersions, nil
}

func bump(s map[string]string, kind BumpKind, preRelease bool) (map[string]string, error) {
	out := make(map[string]string, len(s))

	for p, v := range s {
		ver, err := bumpVersion(v, kind, preRelease)
		if err != nil {
			return nil, err
		}
		out[p] = ver
	}

	return out, nil
}

/*
Get changed projects.
Expects a map of projects to versions.
Returns a map of projects including dependencies, mapped to versions
*/
func getChangedProjectsToVersions(p map[string]string, base string) (map[string]string, error) {
	baseCommits := make(map[string]string)

	if base == "" {
		for proj, v := range p {
			base, err := git.GetBase(proj + "/" + v)
			if err != nil {
				return nil, err
			}

			baseCommits[proj] = base
		}
	} else {
		for proj := range p {
			baseCommits[proj] = base
		}
	}

	head, err := git.GetHead()
	if err != nil {
		return nil, err
	}

	changedProjects := make(map[string]string, 0)

	for _, c := range baseCommits {
		changed, err := ListChangedProjectNamesBetweenCommits(c, head)
		if err != nil {
			return nil, err
		}

		for _, name := range changed {
			pr, ok := app.State.Projects[name]
			if !ok {
				return nil, fmt.Errorf("project name not found in config: %q", name)
			}

			// skip if the name returned from the listChanged function is not in the list of input tags
			// this can happen when a project does not have any git tags yet. since the diffing function is based on the global app.State.Projects and doesnt filter by tags.
			// so we do it here
			if _, ok := p[pr.Name()]; !ok {
				continue
			}
			changedProjects[pr.Name()] = p[name]
		}
	}

	return changedProjects, nil
}

func getLatestTagPerProject(pToT map[projects.Project][]string) (map[string]string, error) {
	latestPerProject := make(map[string]string, len(pToT))

	for p, tags := range pToT {
		for _, t := range tags {
			splitTag := strings.Split(t, "/")
			// project := splitTag[len(splitTag)-2]
			version := splitTag[len(splitTag)-1]

			if !semver.IsValid(version) {
				return nil, fmt.Errorf("invalid semver: %q", version)
			}

			currentLatest, exists := latestPerProject[p.Name()]
			if !exists || semver.Compare(version, currentLatest) > 0 {
				latestPerProject[p.Name()] = version
			}
		}
	}

	return latestPerProject, nil
}

func bumpVersion(version string, kind BumpKind, preRelease bool) (string, error) {
	if !semver.IsValid(version) {
		return "", fmt.Errorf("invalid semver: %q", version)
	}

	// Remove leading "v"
	v := strings.TrimPrefix(version, "v")

	// Separate prerelease suffix
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
		return "", fmt.Errorf("unknown component: %s", kind)
	}

	// TODO: custom suffixes
	// Reattach prerelease or custom suffix if present
	newVersion := fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	if preRelease {
		newVersion = newVersion + "-" + pre
	}

	return newVersion, nil
}
