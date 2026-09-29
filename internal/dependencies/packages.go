package dependencies

import (
	"fmt"
	"path"

	"github.com/arnoldvann/monotrack/internal/projects"
)

// GoPackages is the package-level import graph of a config's Go projects,
// keyed by repo-relative package directory.
type GoPackages struct {
	reached map[string]map[string]struct{}
	owned   map[string]map[string]struct{}
	idx     index
}

// LoadGoPackages runs `go list` for every Go project in cfg.
func LoadGoPackages(root string, cfg *projects.Config) (*GoPackages, error) {
	g := &GoPackages{
		reached: make(map[string]map[string]struct{}),
		owned:   make(map[string]map[string]struct{}),
		idx:     newIndex(cfg),
	}
	for name, pc := range cfg.Projects {
		if pc.Type != projects.ProjectTypeGo {
			continue
		}
		dirs, err := goPackageDirs(root, pc)
		if err != nil {
			return nil, fmt.Errorf("project %q: %w", name, err)
		}
		set := make(map[string]struct{}, len(dirs))
		for _, dir := range dirs {
			set[dir] = struct{}{}
			if owner, ok := g.idx.owner(dir); ok {
				if g.owned[owner] == nil {
					g.owned[owner] = make(map[string]struct{})
				}
				g.owned[owner][dir] = struct{}{}
			}
		}
		g.reached[name] = set
	}
	return g, nil
}

// Loaded reports whether project's packages were listed.
func (g *GoPackages) Loaded(project string) bool {
	_, ok := g.reached[project]
	return ok
}

// Reaches returns the projects owning a package that project loads, itself
// included.
func (g *GoPackages) Reaches(project string) map[string]struct{} {
	out := make(map[string]struct{})
	for dir := range g.reached[project] {
		if owner, ok := g.idx.owner(dir); ok {
			out[owner] = struct{}{}
		}
	}
	return out
}

// Imports reports whether project loads any package owned by dep.
func (g *GoPackages) Imports(project, dep string) bool {
	for dir := range g.reached[project] {
		if owner, ok := g.idx.owner(dir); ok && owner == dep {
			return true
		}
	}
	return false
}

// Affects reports whether a change to file, owned by project owner, can
// affect project.
func (g *GoPackages) Affects(project, owner, file string) bool {
	dir := path.Dir(file)
	if _, ok := g.reached[project][dir]; ok {
		return true
	}
	_, known := g.owned[owner][dir]
	return !known
}
