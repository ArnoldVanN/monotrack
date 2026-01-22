package utils

import (
	"fmt"
	"strings"

	"github.com/arnoldvann/monotrack/internal/projects"
	"golang.org/x/mod/semver"
)

func GetLatestTagPerProject(pToT map[projects.Project][]string) (map[string]string, error) {
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
