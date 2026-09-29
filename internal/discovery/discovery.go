// Package discovery finds the projects in a repository from their native
// manifests (go.mod, package.json, Chart.yaml), so `monotrack init` can write a
// real config instead of a template.
package discovery

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/arnoldvann/monotrack/internal/dependencies"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/bmatcuk/doublestar/v4"
	"go.yaml.in/yaml/v3"
)

// Result is the set of discovered projects, keyed by project name.
type Result struct {
	Projects map[string]projects.ProjectConfig
	// Notes explains every manifest that was found but not turned into a
	// project, and anything else worth reviewing in the generated config.
	Notes []string
}

// manifests lists the recognized manifest files, in the order that decides
// the type of a directory holding more than one of them.
var manifests = []string{"go.mod", "Chart.yaml", "package.json"}

func typeOf(manifest string) projects.ProjectConfig {
	switch manifest {
	case "go.mod":
		return projects.ProjectConfig{Type: projects.ProjectTypeGo}
	case "Chart.yaml":
		return projects.ProjectConfig{Type: projects.ProjectTypeHelm}
	default:
		return projects.ProjectConfig{Type: projects.ProjectTypeNode}
	}
}

// Options narrows discovery to what a config does not already cover.
type Options struct {
	// RootName names a project found at the repository root.
	RootName string
	// Existing are the projects already configured. Their directories are not
	// reported again, and new projects are named around theirs.
	Existing map[string]projects.ProjectConfig
	// Exclude holds doublestar patterns for directories never to report.
	Exclude []string
}

// Discover picks projects out of files, the repo-relative paths of every file
// in the repository, reading manifests from fsys (rooted at the repository).
// Only projects missing from opts.Existing are returned.
func Discover(fsys fs.FS, files []string, opts Options) (*Result, error) {
	found := make(map[string][]string) // dir -> manifests present
	for _, f := range files {
		base := path.Base(f)
		if !slices.Contains(manifests, base) {
			continue
		}
		if _, err := fs.Stat(fsys, f); err != nil {
			continue // tracked but deleted from the working tree
		}
		found[path.Dir(f)] = append(found[path.Dir(f)], base)
	}

	res := &Result{Projects: make(map[string]projects.ProjectConfig)}
	notef := func(format string, args ...any) {
		res.Notes = append(res.Notes, fmt.Sprintf(format, args...))
	}

	dirs := make([]string, 0, len(found))
	for dir := range found {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	configured := make(map[string]string, len(opts.Existing))
	for name, pc := range opts.Existing {
		configured[cleanPath(pc.Path)] = name
	}

	candidates := make(map[string]projects.ProjectConfig)
	for _, dir := range dirs {
		if _, ok := configured[dir]; ok {
			continue
		}
		if excluded(dir, opts.Exclude) {
			continue
		}
		if seg, ok := skippedSegment(dir); ok {
			notef("skipped %s: inside a %s directory", dir, seg)
			continue
		}
		if parent, ok := subchartOf(dir, found); ok {
			notef("skipped %s: packaged subchart of %s", dir, parent)
			continue
		}

		present := found[dir]
		if slices.Contains(present, "package.json") {
			ws, err := isWorkspaceRoot(fsys, dir)
			if err != nil {
				return nil, err
			}
			if ws {
				notef("skipped %s: node workspace root", path.Join(dir, "package.json"))
				present = slices.DeleteFunc(slices.Clone(present), func(m string) bool { return m == "package.json" })
				if len(present) == 0 {
					continue
				}
			}
		}

		var chosen string
		for _, m := range manifests {
			if slices.Contains(present, m) {
				chosen = m
				break
			}
		}
		pc := typeOf(chosen)
		pc.Path = dir
		candidates[dir] = pc
		if len(present) > 1 {
			var others []string
			for _, m := range present {
				if m != chosen {
					others = append(others, m)
				}
			}
			sort.Strings(others)
			notef("%s has %s and %s; treated as %s", dir, chosen, strings.Join(others, ", "), pc.Type)
		}
	}

	// The repo root may only be a project when it is the only one.
	if root, ok := candidates["."]; ok {
		if len(candidates) == 1 && len(opts.Existing) == 0 {
			res.Projects[sanitize(opts.RootName)] = root
			return res, nil
		}
		notef("skipped the repository root (%s): a root project is only allowed when it is the only project", root.Type)
		delete(candidates, ".")
	}
	if name, ok := configured["."]; ok && len(candidates) > 0 {
		dirs := make([]string, 0, len(candidates))
		for dir := range candidates {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		return nil, fmt.Errorf("project %q is at the repository root, so no other project can be added (found %s); move it to its own path, or list the others under discovery.exclude",
			name, strings.Join(dirs, ", "))
	}

	taken := make(map[string]bool, len(opts.Existing))
	for name := range opts.Existing {
		taken[name] = true
	}
	for name, dir := range nameDirs(candidates, taken) {
		res.Projects[name] = candidates[dir]
	}
	return res, nil
}

// skippedSegment reports a directory that never holds a real project: test
// fixtures, and dependency trees committed to the repo.
func skippedSegment(dir string) (string, bool) {
	for seg := range strings.SplitSeq(dir, "/") {
		switch seg {
		case "testdata", "node_modules", "vendor":
			return seg, true
		}
	}
	return "", false
}

// subchartOf reports a chart sitting under another chart's charts/ directory,
// which helm treats as a packaged dependency of that chart.
func subchartOf(dir string, found map[string][]string) (string, bool) {
	for p := dir; p != "." && p != "/"; p = path.Dir(p) {
		if path.Base(p) != "charts" {
			continue
		}
		parent := path.Dir(p)
		if slices.Contains(found[parent], "Chart.yaml") {
			return parent, true
		}
	}
	return "", false
}

func isWorkspaceRoot(fsys fs.FS, dir string) (bool, error) {
	if _, err := fs.Stat(fsys, path.Join(dir, "pnpm-workspace.yaml")); err == nil {
		return true, nil
	}

	p := path.Join(dir, "package.json")
	b, err := fs.ReadFile(fsys, p)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", p, err)
	}
	var pkg struct {
		Workspaces json.RawMessage `json:"workspaces"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		return false, fmt.Errorf("parsing %s: %w", p, err)
	}
	return len(pkg.Workspaces) > 0 && string(pkg.Workspaces) != "null", nil
}

func excluded(dir string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := doublestar.Match(strings.TrimSuffix(p, "/"), dir); ok {
			return true
		}
	}
	return false
}

func cleanPath(p string) string {
	if p == "" {
		return "."
	}
	return path.Clean(filepath.ToSlash(p))
}

// nameDirs names each directory after its last path segment, widening to more
// segments (`apps-api`, `services-api`) only for names that would collide with
// each other or with a taken name.
func nameDirs(candidates map[string]projects.ProjectConfig, taken map[string]bool) map[string]string {
	depth := make(map[string]int, len(candidates))
	for dir := range candidates {
		depth[dir] = 1
	}

	nameFor := func(dir string) string {
		segs := strings.Split(dir, "/")
		n := min(depth[dir], len(segs))
		parts := segs[len(segs)-n:]
		for i, p := range parts {
			parts[i] = sanitize(p)
		}
		return strings.Join(parts, "-")
	}

	for {
		byName := make(map[string][]string)
		for dir := range candidates {
			byName[nameFor(dir)] = append(byName[nameFor(dir)], dir)
		}

		widened := false
		for _, group := range byName {
			if len(group) < 2 && !taken[nameFor(group[0])] {
				continue
			}
			for _, dir := range group {
				if depth[dir] < strings.Count(dir, "/")+1 {
					depth[dir]++
					widened = true
				}
			}
		}
		if widened {
			continue
		}

		// Nothing left to widen, e.g. `a b` next to `a-b`: number the rest.
		out := make(map[string]string, len(candidates))
		names := make([]string, 0, len(byName))
		for name := range byName {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			group := byName[name]
			sort.Strings(group)
			i := 1
			for _, dir := range group {
				n := name
				for taken[n] || out[n] != "" {
					i++
					n = name + "-" + strconv.Itoa(i)
				}
				out[n] = dir
			}
		}
		return out
	}
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// sanitize keeps a name usable as a tag prefix.
func sanitize(s string) string {
	s = strings.Trim(unsafeName.ReplaceAllString(s, "-"), "-.")
	if s == "" {
		return "project"
	}
	return s
}

// Generate turns discovered projects into a new config, completed as by
// Complete.
func Generate(root string, res *Result) (*projects.Config, []string, error) {
	cfg := &projects.Config{Projects: make(map[string]projects.ProjectConfig, len(res.Projects))}
	maps.Copy(cfg.Projects, res.Projects)
	notes, err := Complete(root, cfg, slices.Collect(maps.Keys(res.Projects)))
	return cfg, notes, err
}

// Complete fills dependsOn and build.entrypoint for the added projects of cfg.
// Detection needs the repository at root to be the working directory. When it
// fails, the added projects get no dependsOn and are all marked as
// entrypoints, so none goes unreported, and the error is returned alongside.
func Complete(root string, cfg *projects.Config, added []string) ([]string, error) {
	detected, err := dependencies.Resolve(root, cfg, false)
	if err != nil {
		for _, name := range added {
			pc := cfg.Projects[name]
			pc.Build.Entrypoint = true
			cfg.Projects[name] = pc
		}
		return nil, fmt.Errorf("detecting dependencies: %w", err)
	}

	depended := make(map[string]bool)
	for _, deps := range detected {
		for _, d := range deps {
			depended[d] = true
		}
	}
	for _, name := range added {
		pc := cfg.Projects[name]
		pc.DependsOn = detected[name]
		pc.Build.Entrypoint = !depended[name]
		cfg.Projects[name] = pc
	}

	var notes []string
	for _, name := range slices.Sorted(maps.Keys(cfg.Projects)) {
		if slices.Contains(added, name) {
			continue
		}
		for _, d := range detected[name] {
			if slices.Contains(added, d) && !slices.Contains(cfg.Projects[name].DependsOn, d) {
				notes = append(notes, fmt.Sprintf("%s now depends on %s; run `monotrack deps sync` to record it", name, d))
			}
		}
	}
	return notes, nil
}

// Render writes cfg's projects as a monotrack.yaml document, sorted by name.
func Render(cfg *projects.Config) ([]byte, error) {
	projectsNode := &yaml.Node{Kind: yaml.MappingNode}
	for _, name := range slices.Sorted(maps.Keys(cfg.Projects)) {
		projectsNode.Content = append(projectsNode.Content, scalar(name), projectNode(cfg.Projects[name]))
	}
	doc := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{scalar("projects"), projectsNode}}
	return encode(doc, 2)
}

// AddProjects appends the named projects of cfg to the `projects` mapping of
// the config source, sorted by name, leaving the rest of the document intact.
func AddProjects(src []byte, cfg *projects.Config, names []string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config is not a mapping")
	}

	top := doc.Content[0]
	var projectsNode *yaml.Node
	for i := 0; i+1 < len(top.Content); i += 2 {
		if top.Content[i].Value == "projects" {
			projectsNode = top.Content[i+1]
		}
	}
	if projectsNode == nil || projectsNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config has no `projects` mapping")
	}

	for _, name := range slices.Sorted(slices.Values(names)) {
		projectsNode.Content = append(projectsNode.Content, scalar(name), projectNode(cfg.Projects[name]))
	}
	return encode(&doc, dependencies.DetectIndent(src))
}

func projectNode(pc projects.ProjectConfig) *yaml.Node {
	p := &yaml.Node{Kind: yaml.MappingNode}
	p.Content = append(p.Content, scalar("type"), scalar(string(pc.Type)), scalar("path"), scalar(pc.Path))
	if pc.Build.Entrypoint {
		p.Content = append(p.Content, scalar("build"), &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{scalar("entrypoint"), {Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}},
		})
	}
	if len(pc.DependsOn) > 0 {
		seq := &yaml.Node{Kind: yaml.SequenceNode}
		for _, d := range pc.DependsOn {
			seq.Content = append(seq.Content, scalar(d))
		}
		p.Content = append(p.Content, scalar("dependsOn"), seq)
	}
	return p
}

func encode(doc *yaml.Node, indent int) ([]byte, error) {
	var sb strings.Builder
	enc := yaml.NewEncoder(&sb)
	enc.SetIndent(indent)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encoding config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encoding config: %w", err)
	}
	return []byte(sb.String()), nil
}

func scalar(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}
