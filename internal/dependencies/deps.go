// Package dependencies detects internal dependency edges between projects by
// reading each project's native manifest (go.mod/imports, package.json,
// Chart.yaml), so `dependsOn` in monotrack.yaml can be generated and verified.
package dependencies

import (
	"fmt"
	"maps"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arnoldvann/monotrack/internal/projects"
)

// Resolve returns, per project name, the internal projects it depends on,
// sorted and deduplicated. Every project in cfg gets an entry, empty when it
// has no internal dependencies. With transitive set, edges implied by a longer
// path are kept instead of reduced away.
func Resolve(root string, cfg *projects.Config, transitive bool) (map[string][]string, error) {
	idx := newIndex(cfg)

	out := make(map[string][]string, len(cfg.Projects))
	for name, pc := range cfg.Projects {
		var (
			found []string
			err   error
		)
		switch pc.Type {
		case projects.ProjectTypeGo:
			found, err = resolveGo(root, pc, idx)
		case projects.ProjectTypeNode:
			found, err = resolveNode(name, pc, cfg)
		case projects.ProjectTypeHelm:
			found, err = resolveHelm(root, name, pc, cfg, idx)
		default:
			return nil, fmt.Errorf("project %q: no dependency resolver for type %q", name, pc.Type)
		}
		if err != nil {
			return nil, fmt.Errorf("project %q: %w", name, err)
		}
		out[name] = normalize(found, name)
	}

	if !transitive {
		out = Reduce(out)
	}
	return out, nil
}

// Change is the difference between a project's configured and detected
// dependency list. The two removal kinds are not equally safe to act on, so
// they are reported apart.
type Change struct {
	Project string   `json:"project"`
	Added   []string `json:"added,omitempty"`
	// Redundant edges are still reached through a longer detected path, so
	// dropping them leaves the closure — and change detection — unchanged.
	Redundant []Redundant `json:"redundant,omitempty"`
	// Unverified edges have no detected counterpart at all. Usually that means
	// a real dependency the resolver cannot see (a cross-language one, a file
	// read at runtime)
	Unverified []string `json:"unverified,omitempty"`
}

// Redundant is a configured edge already implied by Via.
type Redundant struct {
	Dep string `json:"dep"`
	Via string `json:"via"`
}

// Diff compares the config's dependsOn lists against detected ones, returning
// only projects that differ, sorted by name.
func Diff(cfg *projects.Config, detected map[string][]string) []Change {
	var changes []Change
	for name, pc := range cfg.Projects {
		want := detected[name]
		have := normalize(pc.DependsOn, name)

		wantSet := make(map[string]struct{}, len(want))
		for _, d := range want {
			wantSet[d] = struct{}{}
		}
		haveSet := make(map[string]struct{}, len(have))
		for _, d := range have {
			haveSet[d] = struct{}{}
		}

		c := Change{Project: name}
		for _, d := range want {
			if _, ok := haveSet[d]; !ok {
				c.Added = append(c.Added, d)
			}
		}
		for _, d := range have {
			if _, ok := wantSet[d]; ok {
				continue
			}
			if via, ok := witness(detected, name, d); ok {
				c.Redundant = append(c.Redundant, Redundant{Dep: d, Via: via})
			} else {
				c.Unverified = append(c.Unverified, d)
			}
		}
		if len(c.Added) > 0 || len(c.Redundant) > 0 || len(c.Unverified) > 0 {
			changes = append(changes, c)
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Project < changes[j].Project })
	return changes
}

// witness names a detected dependency of project through which dep is already
// reachable, explaining why the configured edge is redundant.
func witness(detected map[string][]string, project, dep string) (string, bool) {
	for _, d := range detected[project] {
		if _, ok := reachable(detected, d)[dep]; ok {
			return d, true
		}
	}
	return "", false
}

// Target is what sync should write: the detected graph, plus the configured
// edges detection could not account for. Those are kept unless prune is set.
func Target(detected map[string][]string, changes []Change, prune bool) map[string][]string {
	out := make(map[string][]string, len(detected))
	maps.Copy(out, detected)
	if prune {
		return out
	}
	for _, c := range changes {
		if len(c.Unverified) == 0 {
			continue
		}
		out[c.Project] = normalize(append(append([]string(nil), out[c.Project]...), c.Unverified...), c.Project)
	}
	return out
}

func normalize(deps []string, self string) []string {
	seen := make(map[string]struct{}, len(deps))
	out := make([]string, 0, len(deps))
	for _, d := range deps {
		if d == self || d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// index maps a repo-relative path onto the project that owns it.
type index struct {
	entries []indexEntry
}

type indexEntry struct {
	name string
	path string
}

// newIndex orders projects longest-path-first so a project nested inside
// another wins the lookup.
func newIndex(cfg *projects.Config) index {
	var idx index
	for name, pc := range cfg.Projects {
		idx.entries = append(idx.entries, indexEntry{name: name, path: cleanPath(pc.Path)})
	}
	sort.Slice(idx.entries, func(i, j int) bool {
		a, b := idx.entries[i], idx.entries[j]
		if len(a.path) != len(b.path) {
			return len(a.path) > len(b.path)
		}
		return a.name < b.name
	})
	return idx
}

func (i index) owner(rel string) (string, bool) {
	rel = cleanPath(rel)
	for _, e := range i.entries {
		if e.path == "." || rel == e.path || strings.HasPrefix(rel, e.path+"/") {
			return e.name, true
		}
	}
	return "", false
}

// ownerOfAbs maps an absolute path back to a project, ignoring anything
// outside the repo (stdlib, module cache, node_modules symlink targets).
func (i index) ownerOfAbs(root, abs string) (string, bool) {
	rel, ok := relToRoot(root, abs)
	if !ok {
		return "", false
	}
	return i.owner(rel)
}

func relToRoot(root, abs string) (string, bool) {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

func cleanPath(p string) string {
	if p == "" {
		return "."
	}
	return filepath.ToSlash(filepath.Clean(p))
}
