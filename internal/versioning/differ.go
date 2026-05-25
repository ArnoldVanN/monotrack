package versioning

import (
	"maps"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/bmatcuk/doublestar/v4"
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
				if !isManaged(l) && !isIgnored(l, ".", cfg.Ignore) {
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
			if isIgnored(l, cfg.Path, cfg.Ignore) {
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

// isIgnored evaluates the ignore patterns (relative to projectPath) against
// a diff file path. Patterns are processed top-to-bottom; last match wins.
// A "!" prefix negates a pattern (re-includes the file).
func isIgnored(file, projectPath string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	var rel string
	if projectPath == "." {
		rel = file
	} else {
		rel = strings.TrimPrefix(file, projectPath+"/")
	}
	ignored := false
	for _, p := range patterns {
		negate := strings.HasPrefix(p, "!")
		if negate {
			p = p[1:]
		}
		if matched, _ := doublestar.Match(p, rel); matched {
			ignored = !negate
		}
	}
	return ignored
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
