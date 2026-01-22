package versioning

import (
	"fmt"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
)

// Returns all projects including dependencies
func ListProjectsChangedBetweenCommits(base string, head string) (map[string]bool, error) {
	diff, err := git.GitDiff(base, head)
	if err != nil {
		return nil, err
	}

	fmt.Printf("diff: %v\n", diff)

	// set of dependency names to parent projects
	reverseDeps := make(map[string]map[string]struct{})
	for _, p := range app.State.Projects {
		for _, d := range p.ListDependencies() {
			if reverseDeps[d.Name()] == nil {
				reverseDeps[d.Name()] = make(map[string]struct{})
			}
			reverseDeps[d.Name()][p.Name()] = struct{}{}
		}
	}

	changedMap := make(map[string]bool)

	for name, cfg := range app.State.Config.Projects {
		for _, l := range diff {
			if strings.Contains(l, cfg.Path) {
				changedMap[name] = true
				collectParents(name, reverseDeps, changedMap)
			}
		}
	}

	fmt.Printf("changed: %v\n", changedMap)

	return changedMap, nil
}

func collectParents(
	name string,
	reverse map[string]map[string]struct{},
	out map[string]bool,
) {
	for parent := range reverse[name] {
		if out[parent] {
			continue
		}
		out[parent] = true
		collectParents(parent, reverse, out)
	}
}
