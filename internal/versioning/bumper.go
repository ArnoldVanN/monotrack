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
	projects map[string]projects.Project,
	kind BumpKind,
	preRelease bool,
	base string,
) (map[string]string, error) {
	allTags, err := git.GetTagsForProjects(projects)
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

	bumped, err := bump(changedProjects, kind)
	if err != nil {
		return nil, err
	}

	if preRelease {
		appendPrSuffix(bumped)
	}

	return bumped, nil
}

func bump(s map[string]string, kind BumpKind) (map[string]string, error) {
	out := make(map[string]string, len(s))

	for p, v := range s {
		ver, err := bumpVersion(v, kind)
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
		changed, err := ListChangedProjectsBetweenCommits(c, head)
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

func getLatestTagPerProject(tags []string) (map[string]string, error) {
	latestPerProject := make(map[string]string, len(tags))

	for _, i := range tags {
		splitTag := strings.Split(i, "/")
		project := splitTag[len(splitTag)-2]
		version := splitTag[len(splitTag)-1]

		proj, ok := app.State.Projects[project]
		if !ok {
			return nil, fmt.Errorf("project name parsed from tag doesn't match project specified in config: %q", project)
		}

		if !semver.IsValid(version) {
			return nil, fmt.Errorf("invalid semver: %q", version)
		}

		if semver.Compare(version, latestPerProject[proj.Name()]) > 0 {
			latestPerProject[proj.Name()] = version
		}
	}

	return latestPerProject, nil
}

func bumpVersion(version string, kind BumpKind) (string, error) {
	if !semver.IsValid(version) {
		return "", fmt.Errorf("invalid semver: %q", version)
	}

	// Remove the leading "v"
	v := strings.TrimPrefix(version, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("expected 3 version components")
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

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

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch), nil
}

// TODO: custom suffixes
func appendPrSuffix(s map[string]string) {
	for p, v := range s {
		tag := v + "-rc"
		s[p] = tag
	}
}
