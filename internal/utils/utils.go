package utils

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"golang.org/x/mod/semver"
)

// GetLatestTagPerProject returns the highest-semver tag per project, ignoring
// any tag whose commit is not reachable from head. Tags that live only on
// side branches (e.g. a release tagged on a feature branch that was never
// merged forward) are skipped so the resulting `<tag>..head` range stays
// anchored to the actual release history of head.
func GetLatestTagPerProject(cfg *projects.Config, pToT map[projects.Project][]string, head string) (map[string]string, error) {
	latestPerProject := make(map[string]string, len(pToT))

	for p, tags := range pToT {
		for _, t := range tags {
			version, ok := cfg.MatchTag(t, p.Name())
			if !ok {
				continue
			}
			if !semver.IsValid(version) {
				return nil, fmt.Errorf("invalid semver: %q (from tag %q)", version, t)
			}

			currentLatest, exists := latestPerProject[p.Name()]
			if exists && semver.Compare(version, currentLatest) <= 0 {
				continue
			}

			reachable, err := git.IsAncestor(t, head)
			if err != nil {
				return nil, fmt.Errorf("checking ancestry of tag %q: %w", t, err)
			}
			if !reachable {
				continue
			}

			latestPerProject[p.Name()] = version
		}
	}

	return latestPerProject, nil
}
