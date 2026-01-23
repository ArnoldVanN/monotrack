package versioning

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/utils"
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
	dry bool,
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

	projectToLatest, err := utils.GetLatestTagPerProject(projectToTags)
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

	if !dry {
		if err := pushTags(projectsToBumped); err != nil {
			return nil, err
		}
	}

	return projectsToBumped, nil
}

/*
Get changed projects.
Expects a map of projects to current versions.
Returns a map of projects that have changed, mapped to their respective versions
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

	// NOTE: maybe remove this to make sure no empty head hash gets provided during CI,
	// for example caused by failure to automatically determin the sha in the Action
	// instead error on empty head flag
	if head == "" {
		gitHead, err := git.GetHead()
		if err != nil {
			return nil, err
		}
		head = gitHead
	}

	changedProjects := make(map[string]string, 0)

	for _, c := range baseCommits {
		changed, err := ListProjectsChangedBetweenCommits(c, head)
		if err != nil {
			return nil, err
		}

		for name, c := range changed {
			if !c {
				continue
			}

			// filter out dependencies
			pr, ok := app.State.Projects[name]
			if !ok {
				continue
			}

			changedProjects[pr.Name()] = p[pr.Name()]
		}
	}

	return changedProjects, nil
}

func bump(t map[string]string, kind BumpKind, preRelease bool) (map[string]string, error) {
	out := make(map[string]string, len(t))

	for p, v := range t {
		ver, err := bumpVersion(v, kind, preRelease)
		if err != nil {
			return nil, err
		}
		out[p] = ver
	}

	return out, nil
}

// TODO: custom prefixes and separators
func bumpVersion(version string, kind BumpKind, preRelease bool) (string, error) {
	if !semver.IsValid(version) {
		return "", fmt.Errorf("invalid semver: %q", version)
	}

	v := strings.TrimPrefix(version, "v")

	// Split core and prerelease
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
		// We are already on a prerelease, bump prerelease number
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
		// Bump main version
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

		// Start prerelease if requested
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

func pushTags(tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	fullTags := make([]string, 0, len(tags))
	for project, version := range tags {
		tag := fmt.Sprintf("%s/%s", project, version)
		fullTags = append(fullTags, tag)

		// TODO: do this concurrently
		exists, err := git.TagExistsOnRemote(tag)
		if err != nil {
			return fmt.Errorf("checking remote tag %q failed: %w", tag, err)
		}
		if exists {
			return fmt.Errorf("remote tag already exists: %s", tag)
		}
	}

	for _, tag := range fullTags {
		err := git.CreateTag(tag, fmt.Sprintf("Release %s", tag))
		if err != nil {
			return err
		}
	}

	if err := git.PushTags(fullTags); err != nil {
		return err
	}

	return nil
}
