package versioning

import (
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
)

func ListChangedProjectsBetweenCommits(base string, head string) ([]string, error) {
	diff, err := git.GitDiff(base, head)
	if err != nil {
		return nil, err
	}

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

	changedMap := make(map[string]struct{})

	for _, p := range app.State.Projects {
		for _, l := range diff {
			if strings.Contains(l, p.Path()) {
				changedMap[p.Name()] = struct{}{}
				collectParents(p.Name(), reverseDeps, changedMap)
			}
		}
	}

	names := make([]string, 0, len(changedMap))
	for k := range changedMap {
		names = append(names, k)
	}

	return names, nil
}

func collectParents(
	dep string,
	reverse map[string]map[string]struct{},
	out map[string]struct{},
) {
	for parent := range reverse[dep] {
		if _, seen := out[parent]; seen {
			continue
		}
		out[parent] = struct{}{}
		collectParents(parent, reverse, out)
	}
}
