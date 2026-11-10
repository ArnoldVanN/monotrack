package projects

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/arnoldvann/monotrack/internal/versioning/conventional"
	"github.com/bmatcuk/doublestar/v4"
)

type Config struct {
	// Name labels release commits and PR titles produced by this config, so a
	// repo running several monotrack configs can tell them apart. Optional.
	Name      string                   `mapstructure:"name"`
	Projects  map[string]ProjectConfig `mapstructure:"projects"`
	Release   ReleaseConfig            `mapstructure:"release"`
	Tags      TagsConfig               `mapstructure:"tags"`
	Changelog ChangelogConfig          `mapstructure:"changelog"`
	Discovery DiscoveryConfig          `mapstructure:"discovery"`
}

// DiscoveryConfig tunes `projects check` and `projects sync`.
type DiscoveryConfig struct {
	// Exclude holds doublestar patterns, matched against repo-relative
	// directories, for projects that should never be reported as missing.
	Exclude []string `mapstructure:"exclude"`
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
	// PreMajor shifts bumps down one level while the version is below 1.0.0:
	// breaking changes bump the minor and features bump the patch.
	PreMajor  bool     `mapstructure:"preMajor"`
	DependsOn []string `mapstructure:"dependsOn"`
	// Versioned projects get their own tags and are the ones `compare` and
	// `tag bump` report. The rest only pass their changes on to the projects
	// that depend on them.
	Versioned bool `mapstructure:"versioned"`
	// Build is where Versioned used to live, as build.entrypoint.
	Build       BuildConfig `mapstructure:"build"`
	Ignore      []string    `mapstructure:"ignore"`
	Granularity string      `mapstructure:"granularity"`
	// ReleaseTypes lists the conventional commit types that trigger a
	// release. Breaking changes always do. "*" releases on any commit,
	// conventional or not. Defaults to DefaultReleaseTypes.
	ReleaseTypes []string `mapstructure:"releaseTypes"`
}

// DefaultReleaseTypes are the commit types that release a project when it
// doesn't set releaseTypes.
var DefaultReleaseTypes = []string{"feat", "fix", "perf", "revert"}

func (pc ProjectConfig) releaseTypes() []string {
	if pc.ReleaseTypes == nil {
		return DefaultReleaseTypes
	}
	return pc.ReleaseTypes
}

// ReleasesAll reports whether every commit releases the project.
func (pc ProjectConfig) ReleasesAll() bool {
	return slices.Contains(pc.releaseTypes(), "*")
}

// Releases reports whether commit c triggers a release of the project.
func (pc ProjectConfig) Releases(c conventional.ParsedCommit) bool {
	if c.Breaking || pc.ReleasesAll() {
		return true
	}
	return c.Type != conventional.TypeUnknown && slices.Contains(pc.releaseTypes(), string(c.Type))
}

const (
	GranularityProject = "project"
	GranularityPackage = "package"
)

type BuildConfig struct {
	// Deprecated: use ProjectConfig.Versioned. MigrateDeprecated moves it there.
	Entrypoint bool `mapstructure:"entrypoint"`
}

// MigrateDeprecated moves build.entrypoint onto versioned and returns the
// projects that still set it.
func (c *Config) MigrateDeprecated() []string {
	var names []string
	for name, pc := range c.Projects {
		if !pc.Build.Entrypoint {
			continue
		}
		pc.Versioned = true
		pc.Build.Entrypoint = false
		c.Projects[name] = pc
		names = append(names, name)
	}
	slices.Sort(names)
	return names
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
	IsVersioned() bool
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
	c.Name = strings.TrimSpace(c.Name)
	if strings.ContainsAny(c.Name, "[]\n") {
		return fmt.Errorf("name %q may not contain brackets or newlines", c.Name)
	}

	for _, pattern := range c.Discovery.Exclude {
		if !doublestar.ValidatePattern(pattern) {
			return fmt.Errorf("discovery: invalid exclude pattern %q", pattern)
		}
	}

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

		switch pc.Granularity {
		case "", GranularityProject:
		case GranularityPackage:
			if pc.Type != ProjectTypeGo {
				return fmt.Errorf("project %q: granularity %q is only supported for go projects", name, pc.Granularity)
			}
		default:
			return fmt.Errorf("project %q has invalid granularity %q (must be one of: project, package)", name, pc.Granularity)
		}

		for _, t := range pc.ReleaseTypes {
			if t != "*" && !conventional.IsKnownType(t) {
				return fmt.Errorf("project %q: unknown release type %q", name, t)
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

	}

	return nil
}

// BuildProjects builds the versioned projects among roots, or among every
// configured project when roots is empty. Dependencies are built to wire up
// the graph but are only returned when they are roots themselves.
func BuildProjects(config *Config, roots []string) (map[string]Project, error) {
	names := roots
	if len(names) == 0 {
		names = slices.Collect(maps.Keys(config.Projects))
	}

	built := make(map[string]Project)
	versioned := make(map[string]Project)
	for _, name := range names {
		if _, ok := config.Projects[name]; !ok {
			return nil, fmt.Errorf("project %q not found in config", name)
		}
		proj, err := buildProject(name, config, built, map[string]bool{})
		if err != nil {
			return nil, err
		}
		if proj.IsVersioned() {
			versioned[name] = proj
		}
	}

	return versioned, nil
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
		p = NewGoProject(name, cfg.Path, cfg.Versioned, cfg.Type)
	case ProjectTypeNode:
		p = NewNodeProject(name, cfg.Path, cfg.Versioned, cfg.Type)
	case ProjectTypeHelm:
		p = NewHelmProject(name, cfg.Path, cfg.Versioned, cfg.Type)
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
