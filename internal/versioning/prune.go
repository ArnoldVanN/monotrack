package versioning

import (
	"fmt"
	"sort"

	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"golang.org/x/mod/semver"
)

// PruneItem is a single tag selected for deletion
type PruneItem struct {
	Project string
	Tag     string
	Version string
}

// PrunePlan is the set of prerelease tags eligible for deletion under the
// keep-policy, plus the per-project latest tag that was preserved.
type PrunePlan struct {
	Delete []PruneItem
	// Kept maps project name to the highest-semver tag retained. Projects with
	// no tags are absent.
	Kept map[string]string
}

// PlanPrune computes which prerelease tags can be safely deleted.
func PlanPrune(cfg *projects.Config, p map[string]projects.Project) (*PrunePlan, error) {
	projectToTags, err := git.GetTagsForProjects(cfg, p)
	if err != nil {
		return nil, err
	}
	return planPruneFromTags(cfg, projectToTags)
}

func planPruneFromTags(cfg *projects.Config, projectToTags map[projects.Project][]string) (*PrunePlan, error) {
	plan := &PrunePlan{Kept: make(map[string]string, len(projectToTags))}

	for proj, tags := range projectToTags {
		name := proj.Name()

		latest, err := highestVersion(cfg, name, tags)
		if err != nil {
			return nil, err
		}
		if latest == "" {
			continue // project has no tags
		}
		plan.Kept[name] = cfg.TagFor(name, latest)

		// Latest is still a prerelease: release in flight, keep everything.
		if semver.Prerelease(latest) != "" {
			continue
		}

		for _, t := range tags {
			v, ok := cfg.MatchTag(t, name)
			if !ok {
				continue
			}
			if semver.Prerelease(v) == "" {
				continue // keep every stable release
			}
			if semver.Compare(v, latest) >= 0 {
				continue // keep the latest (and, defensively, anything above it)
			}
			plan.Delete = append(plan.Delete, PruneItem{Project: name, Tag: t, Version: v})
		}
	}

	sort.Slice(plan.Delete, func(i, j int) bool {
		if plan.Delete[i].Project != plan.Delete[j].Project {
			return plan.Delete[i].Project < plan.Delete[j].Project
		}
		return semver.Compare(plan.Delete[i].Version, plan.Delete[j].Version) < 0
	})

	return plan, nil
}

// highestVersion returns the highest-semver canonical version among the tags
// that belong to name, or "" when none match.
func highestVersion(cfg *projects.Config, name string, tags []string) (string, error) {
	var best string
	for _, t := range tags {
		v, ok := cfg.MatchTag(t, name)
		if !ok {
			continue
		}
		if !semver.IsValid(v) {
			return "", fmt.Errorf("invalid semver: %q (from tag %q)", v, t)
		}
		if best == "" || semver.Compare(v, best) > 0 {
			best = v
		}
	}
	return best, nil
}

// Tags returns just the tag strings selected for deletion.
func (p *PrunePlan) Tags() []string {
	out := make([]string, 0, len(p.Delete))
	for _, item := range p.Delete {
		out = append(out, item.Tag)
	}
	return out
}
