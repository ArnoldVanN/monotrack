package projects

import (
	"fmt"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type Config struct {
	Projects  map[string]ProjectConfig `mapstructure:"projects"`
	Release   ReleaseConfig            `mapstructure:"release"`
	Tags      TagsConfig               `mapstructure:"tags"`
	Changelog ChangelogConfig          `mapstructure:"changelog"`
}

// ChangelogConfig customizes changelog output.
// The `path` field is honored only when a single changelog is used.
type ChangelogConfig struct {
	Path string `mapstructure:"path"`
}

func (c ChangelogConfig) path() string {
	if c.Path == "" {
		return "CHANGELOG.md"
	}
	return c.Path
}

// ChangelogPath returns the configured changelog path, defaulting to
// "CHANGELOG.md".
func (c *Config) ChangelogPath() string {
	return c.Changelog.path()
}

// TagsConfig customizes how project tags are rendered and parsed.
// Default scheme: `<name><separator><prefix><version>` (e.g. `api/v1.2.3`).
// single-project repos drop the `<name><separator>` part.
type TagsConfig struct {
	// Separator placed between the project name and the version prefix.
	// Defaults to "/". Ignored in single-project mode.
	Separator string `mapstructure:"separator"`
	// VersionPrefix placed immediately before the numeric version (e.g. "v"
	// in `v1.2.3`). Defaults to "v".
	VersionPrefix *string `mapstructure:"versionPrefix"`
}

func (t TagsConfig) sep() string {
	if t.Separator == "" {
		return "/"
	}
	return t.Separator
}

func (t TagsConfig) vp() string {
	if t.VersionPrefix == nil {
		return "v"
	}
	return *t.VersionPrefix
}

// ReleaseConfig controls the PR-based release flow.
type ReleaseConfig struct {
	// Branch is the long-lived release branch name. The literal "{base}" is
	// replaced with the base branch name at runtime. Defaults to
	// "monotrack/release-{base}" when empty.
	Branch string `mapstructure:"branch"`
}

// ResolveReleaseBranch expands {base} and applies the default when unset.
func (r ReleaseConfig) ResolveReleaseBranch(base string) string {
	tmpl := r.Branch
	if tmpl == "" {
		tmpl = "monotrack/release-{base}"
	}
	return strings.ReplaceAll(tmpl, "{base}", base)
}

type ProjectConfig struct {
	Type       projectType `mapstructure:"type"`
	Path       string      `mapstructure:"path"`
	Versioning string      `mapstructure:"versioning"`
	DependsOn  []string    `mapstructure:"dependsOn"`
	Build      BuildConfig `mapstructure:"build"`
	Ignore     []string    `mapstructure:"ignore"`
}

type BuildConfig struct {
	Entrypoint bool `mapstructure:"entrypoint"`
}

type projectType string

const (
	ProjectTypeNode projectType = "node"
	ProjectTypeGo   projectType = "go"
	ProjectTypeHelm projectType = "helm"
)

var validProjectTypes = map[projectType]struct{}{
	ProjectTypeNode: {},
	ProjectTypeGo:   {},
	ProjectTypeHelm: {},
}

func (t projectType) isValid() bool {
	_, ok := validProjectTypes[t]
	return ok
}

type Project interface {
	Name() string
	Path() string
	GetType() projectType
	AddDependency(Project)
	ListDependencies() []Project
	IsEntrypoint() bool
}

// IsSingleProject reports whether the config defines exactly one project.
func (c *Config) IsSingleProject() bool {
	return len(c.Projects) == 1
}

// TagFor builds a tag string for the given project name and version under
// the configured tag scheme. `version` may already include the canonical "v"
// prefix; it's stripped before re-applying the configured VersionPrefix.
func (c *Config) TagFor(projectName, version string) string {
	bare := strings.TrimPrefix(version, "v")
	rendered := c.Tags.vp() + bare
	if c.IsSingleProject() {
		return rendered
	}
	return projectName + c.Tags.sep() + rendered
}

// MatchTag reports whether `tag` belongs to `projectName` under the
// configured tag scheme. On match, the returned version is the canonical
// semver form (with leading "v") so it can be fed into golang.org/x/mod/semver.
func (c *Config) MatchTag(tag, projectName string) (version string, ok bool) {
	rest := tag
	if !c.IsSingleProject() {
		prefix := projectName + c.Tags.sep()
		if !strings.HasPrefix(tag, prefix) {
			return "", false
		}
		rest = strings.TrimPrefix(tag, prefix)
	}
	vp := c.Tags.vp()
	if vp != "" && !strings.HasPrefix(rest, vp) {
		return "", false
	}
	return "v" + strings.TrimPrefix(rest, vp), true
}

func (c *Config) Validate() error {
	for name, pc := range c.Projects {
		if !pc.Type.isValid() {
			return fmt.Errorf(
				"project %q has invalid type %q (must be one of: node, go, helm)",
				name, pc.Type,
			)
		}

		// "" and "." both mean "the repo root" and are only allowed for
		// single-project repos. Normalize "" to "." so downstream code can
		// rely on a single representation.
		if pc.Path == "" || pc.Path == "." {
			if !c.IsSingleProject() {
				return fmt.Errorf("project %q has path %q (repo root); only allowed when monotrack.yaml defines a single project", name, pc.Path)
			}
			if pc.Path == "" {
				pc.Path = "."
				c.Projects[name] = pc
			}
		}

		info, err := os.Stat(pc.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("project %q path %q does not exist", name, pc.Path)
			}
			return fmt.Errorf("project %q path %q: %w", name, pc.Path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("project %q path %q is not a directory", name, pc.Path)
		}

		for _, pattern := range pc.Ignore {
			p := strings.TrimPrefix(pattern, "!")
			if !doublestar.ValidatePattern(p) {
				return fmt.Errorf("project %q: invalid ignore pattern %q", name, pattern)
			}
		}

		for _, dep := range pc.DependsOn {
			if _, ok := c.Projects[string(dep)]; !ok {
				return fmt.Errorf(
					"project %q depends on unknown project %q",
					name,
					dep,
				)
			}
		}

		// remove for now since we're only outputting entrypoints now, so helm charts need to be entrypoint too
		// if pc.Build.Entrypoint {
		// 	switch pc.Type {
		// 	case ProjectTypeNode, ProjectTypeGo:
		// 		// ok
		// 	default:
		// 		return fmt.Errorf(
		// 			"project %q: build.entrypoint is not allowed for project type %q",
		// 			name, pc.Type,
		// 		)
		// 	}
		// }
	}

	return nil
}

func BuildProjects(config *Config, roots []string) (map[string]Project, error) {
	// if user defined projects via flag, only build those
	filteredConfigs := config.Projects
	if len(roots) > 0 {
		for _, r := range roots {
			conf, ok := config.Projects[r]
			if !ok {
				return nil, fmt.Errorf("project %q not found in config", r)
			}
			filteredConfigs[r] = conf
		}
	}

	allProjects := make(map[string]Project)
	for name := range filteredConfigs {
		proj, err := buildProject(name, config, allProjects, map[string]bool{})
		if err != nil {
			return nil, err
		}
		allProjects[name] = proj
	}

	entrypoints := make(map[string]Project)

	for name, proj := range allProjects {
		if proj.IsEntrypoint() {
			entrypoints[name] = proj
		}
	}

	return entrypoints, nil
}

func buildProject(name string, config *Config, built map[string]Project, visiting map[string]bool) (Project, error) {
	if proj, exists := built[name]; exists {
		return proj, nil // already built
	}

	if visiting[name] {
		return nil, fmt.Errorf("circular dependency detected on project %q", name)
	}
	visiting[name] = true

	cfg := config.Projects[name]

	var p Project
	switch cfg.Type {
	case ProjectTypeGo:
		p = NewGoProject(name, cfg.Path, cfg.Build.Entrypoint, cfg.Type)
	case ProjectTypeNode:
		p = NewNodeProject(name, cfg.Path, cfg.Build.Entrypoint, cfg.Type)
	case ProjectTypeHelm:
		p = NewHelmProject(name, cfg.Path, cfg.Build.Entrypoint, cfg.Type)
	default:
		return nil, fmt.Errorf("unsupported project type %q", cfg.Type)
	}

	for _, depName := range cfg.DependsOn {
		depProj, err := buildProject(string(depName), config, built, visiting)
		if err != nil {
			return nil, err
		}
		p.AddDependency(depProj)
	}

	built[name] = p
	delete(visiting, name) // finished visiting
	return p, nil
}
