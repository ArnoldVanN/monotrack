package utils

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/projects"
	"golang.org/x/mod/semver"
)

// GetLatestTagPerProject returns the highest-semver tag per project across all
// tags monotrack knows about — including tags whose commits are not reachable
// from head. Reachability must not gate this lookup: Finalize collision-checks
// the computed next version against every tag on the remote, so a version
// derived from only the reachable subset will keep colliding with an existing
// tag forever (e.g. a v0.0.1 release cut on a promotion branch that does not
// merge back to head).
func GetLatestTagPerProject(cfg *projects.Config, pToT map[projects.Project][]string) (map[string]string, error) {
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
			if !exists || semver.Compare(version, currentLatest) > 0 {
				latestPerProject[p.Name()] = version
			}
		}
	}

	return latestPerProject, nil
}
