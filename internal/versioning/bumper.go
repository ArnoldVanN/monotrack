package versioning

import (
	"fmt"
	"maps"
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

// Returns a map of projects that have changed, to bumped versions. Defaults to v0.0.1 for projects that don't have a tag
func (b *VersionBumper) BumpProjects(
	p map[string]projects.Project,
	kind BumpKind,
	preRelease bool,
	head string,
) (map[string]string, error) {
	projectToTags, err := git.GetTagsForProjects(p)
	if err != nil {
		return nil, err
	}

	// Set default tags if they dont exist
	for name, project := range p {
		if _, ok := projectToTags[project]; !ok {
			projectToTags[project] = []string{name + "/v0.0.0"}
		}
	}

	projectToLatest, err := getLatestTagPerProject(projectToTags)
	if err != nil {
		return nil, err
	}

	// Diff projects with existing tags only
	zeroTagProjects := make(map[string]string)
	for proj, t := range projectToLatest {
		// TODO: if it's a preRelease, will have to do extra checks here
		if strings.HasPrefix(t, "v0.0.0") {
			zeroTagProjects[proj] = t
			delete(projectToLatest, proj)
		}
	}

	changedProjNameToVersion, err := getChangedProjectsVersions(projectToLatest, head)
	if err != nil {
		return nil, err
	}

	maps.Copy(changedProjNameToVersion, zeroTagProjects)

	projectsToBumped, err := bump(changedProjNameToVersion, kind, preRelease)
	if err != nil {
		return nil, err
	}

	return projectsToBumped, nil
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
Returns a map of projects that have changed including dependencies, mapped to their respective versions
*/
func getChangedProjectsVersions(p map[string]string, head string) (map[string]string, error) {
	baseCommits := make(map[string]string)

	for proj, v := range p {
		base, err := git.GetBase(proj + "/" + v)
		if err != nil {
			return nil, err
		}

		baseCommits[proj] = base
	}

	if head == "" {
		gitHead, err := git.GetHead()
		if err != nil {
			return nil, err
		}
		head = gitHead
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

	// TODO: custom suffixes and separators
	if preRelease {
		if pre == "" {
			pre = "rc.1"
		} else {
			// Try to bump existing numeric suffix, e.g., rc.1 -> rc.2
			parts := strings.Split(pre, ".")
			if len(parts) == 2 {
				num, err := strconv.Atoi(parts[1])
				if err == nil {
					num++
					pre = fmt.Sprintf("%s.%d", parts[0], num)
				} else {
					// fallback: append .1 if existing suffix isn't numeric
					pre = fmt.Sprintf("%s.1", pre)
				}
			} else {
				// fallback: append .1 if no numeric suffix
				pre = fmt.Sprintf("%s.1", pre)
			}
		}
	}

	newVersion := fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	if pre != "" {
		newVersion = fmt.Sprintf("%s-%s", newVersion, pre)
	}

	return newVersion, nil
}
