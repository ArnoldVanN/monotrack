package versioning

import (
	"maps"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
)

// Returns two sets: `direct` lists projects whose own files changed in
// base..head; `all` adds transitive parents (projects that depend on a
// directly-changed project).
func ListProjectsChangedBetweenCommits(base string, head string) (map[string]bool, map[string]bool, error) {
	diff, err := git.GitDiff(base, head)
	if err != nil {
		return nil, nil, err
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

	// Files monotrack itself writes/rewrites during a release; editing only
	// these should not flag a project as changed.
	isManaged := func(path string) bool {
		base := path
		if i := strings.LastIndex(path, "/"); i >= 0 {
			base = path[i+1:]
		}
		return base == "CHANGELOG.md"
	}

	direct := map[string]bool{}
	for name, cfg := range app.State.Config.Projects {
		// "." means the project IS the repo root — any diff entry counts.
		if cfg.Path == "." {
			for _, l := range diff {
				if !isManaged(l) {
					direct[name] = true
					break
				}
			}
			continue
		}
		for _, l := range diff {
			if !strings.HasPrefix(l, cfg.Path+"/") {
				continue
			}
			if isManaged(l) {
				continue
			}
			direct[name] = true
			break
		}
	}

	all := maps.Clone(direct)
	for name := range direct {
		collectParents(name, reverseDeps, all)
	}

	return direct, all, nil
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
