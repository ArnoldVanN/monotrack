package utils

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"golang.org/x/mod/semver"
)

// GetLatestTagPerProject returns the highest-semver tag per project across
// every tag monotrack knows about — including tags whose commits are not
// reachable from head. Reachability must not gate this lookup: Finalize
// collision-checks the computed next version against every tag on the remote,
// so a version derived from only the reachable subset would keep colliding
// with an existing tag forever (e.g. a v0.0.1-rc.N release cut on a promotion
// branch that does not merge back to head). Use this to decide *what to bump
// to*; use GetLatestReachableTagPerProject for *what to put in the changelog*.
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

// GetLatestReachableTagPerProject returns the highest-semver tag per project
// whose commit is an ancestor of head. Projects whose tags all live on
// branches that never merged into head are absent from the result.
//
// This is intended for computing the changelog range `<base>..head`. When the
// absolute latest tag is on an orphan branch, anchoring the changelog at it
// produces a wildly over-large diff (head's history minus the tiny orphan
// branch's history). Falling back to the latest *reachable* tag — or to no
// base, if none — keeps the range bounded to head's actual release history.
func GetLatestReachableTagPerProject(cfg *projects.Config, pToT map[projects.Project][]string, head string) (map[string]string, error) {
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
