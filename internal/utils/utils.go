package utils

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/projects"
	"golang.org/x/mod/semver"
)

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
