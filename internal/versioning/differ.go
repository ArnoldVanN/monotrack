package versioning

import (
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

	reverseDeps := make(map[string]map[string]struct{})
	for n, p := range app.State.Config.Projects {
		for _, d := range p.DependsOn {
			if reverseDeps[d] == nil {
				reverseDeps[d] = make(map[string]struct{})
			}
			reverseDeps[d][n] = struct{}{}
		}
	}

	changed := map[string]bool{}
	for name, cfg := range app.State.Config.Projects {
		for _, l := range diff {
			if strings.HasPrefix(l, cfg.Path+"/") {
				changed[name] = true
			}
		}
	}

	for name := range changed {
		collectParents(name, reverseDeps, changed)
	}

	return changed, nil
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
