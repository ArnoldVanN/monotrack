package versioning

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"golang.org/x/mod/semver"
)

// Release is a past release reconstructed from a tag.
type Release struct {
	BumpResult
	Tag  string
	Date time.Time
}

type versionedTag struct {
	git.TagRef
	version string
}

// History reconstructs every release of the given projects from their tags
// reachable from head, newest first per project. Each release covers the
// range from the previous tag, widened to the previous stable tag when a
// stable tag follows prereleases, the same ranges bump uses. The current
// config drives it, and dependency changes are always detected at project
// granularity: the import graph of a past commit isn't loaded.
func History(p map[string]projects.Project, head string) ([]Release, error) {
	cfg := app.State.Config

	refs, err := git.ListTagRefs()
	if err != nil {
		return nil, err
	}
	reachable, err := git.TagsMergedInto(head)
	if err != nil {
		return nil, err
	}

	tags := make(map[string][]versionedTag)
	for _, r := range refs {
		if !reachable[r.Name] {
			continue
		}
		for name := range cfg.Projects {
			if v, ok := cfg.MatchTag(r.Name, name); ok && semver.IsValid(v) {
				tags[name] = append(tags[name], versionedTag{r, v})
			}
		}
	}
	for _, ts := range tags {
		slices.SortFunc(ts, func(a, b versionedTag) int { return semver.Compare(a.version, b.version) })
	}

	var out []Release
	for name, proj := range p {
		ts := tags[name]
		for i, t := range ts {
			rel, err := reconstruct(cfg, proj, ts[:i], t, tags)
			if err != nil {
				return nil, err
			}
			out = append(out, rel)
		}
	}

	slices.SortFunc(out, func(a, b Release) int {
		if c := strings.Compare(a.Project.Name(), b.Project.Name()); c != 0 {
			return c
		}
		return semver.Compare(b.NewVersion, a.NewVersion)
	})
	return out, nil
}

// reconstruct rebuilds the release at t from the project's earlier tags.
func reconstruct(cfg *projects.Config, proj projects.Project, earlier []versionedTag, t versionedTag, tags map[string][]versionedTag) (Release, error) {
	name := proj.Name()
	rel := Release{
		BumpResult: BumpResult{Project: proj, NewVersion: t.version, Reason: ReasonInitial},
		Tag:        t.Name,
		Date:       t.Date,
	}
	if len(earlier) == 0 {
		return rel, nil
	}

	prev := earlier[len(earlier)-1]
	rel.OldVersion = prev.version
	rel.Reason = ReasonCommits
	base := prev.Commit
	if semver.Prerelease(t.version) == "" && semver.Prerelease(prev.version) != "" {
		rel.Reason = ReasonPromotion
		for _, e := range slices.Backward(earlier) {
			if semver.Prerelease(e.version) == "" {
				base = e.Commit
				break
			}
		}
	}

	commits, err := commitsForProject(base, t.Commit, proj)
	if err != nil {
		return Release{}, err
	}
	rel.Commits = parseAll(commits)
	rel.Kind = deriveKind(rel.Commits)
	if len(rel.Commits) > 0 || rel.Reason == ReasonPromotion {
		return rel, nil
	}

	// Without commits the changelog lists the changed dependencies instead.
	diff, err := git.GitDiff(base, t.Commit)
	if err != nil {
		return Release{}, err
	}
	all := propagate(cfg, changedFiles(cfg, diff), nil)
	rel.Reason = ReasonDependency
	for _, dep := range cfg.Projects[name].DependsOn {
		if !all[dep] {
			continue
		}
		v, err := versionAt(tags[dep], t)
		if err != nil {
			return Release{}, err
		}
		rel.Dependencies = append(rel.Dependencies, DepUpdate{Name: dep, Version: v})
	}
	slices.SortFunc(rel.Dependencies, func(a, b DepUpdate) int { return strings.Compare(a.Name, b.Name) })
	return rel, nil
}

// versionAt returns the highest version among a dependency's tags (sorted
// ascending) that was released by the time of t, or "" if none was.
func versionAt(dep []versionedTag, t versionedTag) (string, error) {
	for _, d := range slices.Backward(dep) {
		if d.Date.After(t.Date) {
			continue
		}
		ok, err := git.IsAncestor(d.Commit, t.Commit)
		if err != nil {
			return "", fmt.Errorf("checking ancestry of tag %q: %w", d.Name, err)
		}
		if ok {
			return d.version, nil
		}
	}
	return "", nil
}
