package versioning

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/dependencies"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
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

	direct, all := ListProjectsChangedInDiff(diff)
	return direct, all, nil
}

// ListProjectsChangedInDiff maps changed file paths onto projects. Same
// contract as ListProjectsChangedBetweenCommits, for callers holding a diff.
func ListProjectsChangedInDiff(diff []string) (map[string]bool, map[string]bool) {
	cfg := app.State.Config
	files := changedFiles(cfg, diff)

	direct := make(map[string]bool, len(files))
	for name := range files {
		direct[name] = true
	}
	return direct, propagate(cfg, files, goPackages(cfg))
}

// changedFiles groups the diff by owning project, dropping managed and
// ignored files.
func changedFiles(cfg *projects.Config, diff []string) map[string][]string {
	// Files monotrack itself writes/rewrites during a release; editing only
	// these should not flag a project as changed.
	isManaged := func(path string) bool {
		base := path
		if i := strings.LastIndex(path, "/"); i >= 0 {
			base = path[i+1:]
		}
		return base == "CHANGELOG.md"
	}

	files := map[string][]string{}
	for name, pc := range cfg.Projects {
		for _, l := range diff {
			// "." means the project IS the repo root — any diff entry counts.
			if pc.Path != "." && !strings.HasPrefix(l, pc.Path+"/") {
				continue
			}
			if isManaged(l) || isIgnored(l, pc.Path, pc.Ignore) {
				continue
			}
			files[name] = append(files[name], l)
		}
	}
	return files
}

// propagate marks every project changed by its own files or by a changed
// dependency.
func propagate(cfg *projects.Config, files map[string][]string, pkgs *dependencies.GoPackages) map[string]bool {
	memo := make(map[string]bool, len(cfg.Projects))
	var changed func(name string) bool
	changed = func(name string) bool {
		if v, ok := memo[name]; ok {
			return v
		}
		memo[name] = false
		v := len(files[name]) > 0
		if !v {
			if cfg.Projects[name].Granularity == projects.GranularityPackage && pkgs != nil && pkgs.Loaded(name) {
				v = packageChanged(cfg, name, files, pkgs, changed)
			} else {
				v = slices.ContainsFunc(cfg.Projects[name].DependsOn, changed)
			}
		}
		memo[name] = v
		return v
	}

	all := make(map[string]bool)
	for name := range cfg.Projects {
		if changed(name) {
			all[name] = true
		}
	}
	return all
}

// packageChanged reports whether project name is affected by a change in a
// project it reaches. Go imports are checked per file; declared dependencies
// that aren't Go imports count as changed whenever the dependency changed.
func packageChanged(cfg *projects.Config, name string, files map[string][]string, pkgs *dependencies.GoPackages, changed func(string) bool) bool {
	via := pkgs.Reaches(name)
	via[name] = struct{}{}
	for v := range via {
		if v != name && slices.ContainsFunc(files[v], func(f string) bool { return pkgs.Affects(name, v, f) }) {
			return true
		}
		for _, dep := range cfg.Projects[v].DependsOn {
			if !pkgs.Imports(v, dep) && changed(dep) {
				return true
			}
		}
	}
	return false
}

var goPackagesCache struct {
	cfg  *projects.Config
	pkgs *dependencies.GoPackages
}

// goPackages loads the Go import graph once per config, or returns nil when
// no project uses package granularity or loading fails.
func goPackages(cfg *projects.Config) *dependencies.GoPackages {
	if goPackagesCache.cfg == cfg {
		return goPackagesCache.pkgs
	}
	goPackagesCache.cfg = cfg
	goPackagesCache.pkgs = nil

	wanted := false
	for _, pc := range cfg.Projects {
		wanted = wanted || pc.Granularity == projects.GranularityPackage
	}
	if !wanted {
		return nil
	}

	root, err := git.GetRepoRoot()
	if err == nil {
		goPackagesCache.pkgs, err = dependencies.LoadGoPackages(root, cfg)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: package granularity unavailable, using project granularity: %v\n", err)
	}
	return goPackagesCache.pkgs
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
