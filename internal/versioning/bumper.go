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
) (map[string]string, error) {
	tags, err := git.GetTagsForProjects(projects)
	if err != nil {
		return nil, err
	}

	latest, err := getLatestTagPerProject(tags)
	if err != nil {
		return nil, err
	}

	bumped, err := bump(latest, kind)
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

func getLatestTagPerProject(tags []string) (map[string]string, error) {
	latestPerProject := make(map[string]string, len(tags))

	for _, i := range tags {
		splitTag := strings.Split(i, "/")
		project := splitTag[len(splitTag)-2]
		version := splitTag[len(splitTag)-1]

		if _, ok := app.State.Projects[project]; !ok {
			return nil, fmt.Errorf("project name parsed from tag doesn't match project specified in config: %q", project)
		}

		if !semver.IsValid(version) {
			return nil, fmt.Errorf("invalid semver: %q", version)
		}

		if semver.Compare(version, latestPerProject[project]) > 0 {
			latestPerProject[project] = version
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
