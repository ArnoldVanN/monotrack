package projects

import (
	"fmt"
	"os"
)

type Config struct {
	Projects map[string]ProjectConfig `mapstructure:"projects"`
}

type ProjectConfig struct {
	Type       projectType `mapstructure:"type"`
	Path       string      `mapstructure:"path"`
	Versioning string      `mapstructure:"versioning"`
	DependsOn  []string    `mapstructure:"dependsOn"`
}

type BumpKind string

const (
	SemVerBumpKind     BumpKind = "semver"
	PreReleaseBumpKind BumpKind = "pr"
)

type projectType string

const (
	NodeProjectType projectType = "node"
	GoProjectType   projectType = "go"
	HelmProjectType projectType = "helm"
)

var validProjectTypes = map[projectType]struct{}{
	NodeProjectType: {},
	GoProjectType:   {},
	HelmProjectType: {},
}

func (t projectType) isValid() bool {
	_, ok := validProjectTypes[t]
	return ok
}

type Project interface {
	Name() string
	Path() string
	GetVersion() (string, error)
	GetType() projectType
	GetTags() ([]string, error)
	Bump(kind BumpKind) error
	AddDependency(Project)
	ListDependencies() []Project
}

func (c *Config) Validate() error {
	for name, pc := range c.Projects {
		if !pc.Type.isValid() {
			return fmt.Errorf(
				"project %q has invalid type %q (must be one of: node, go, helm)",
				name, pc.Type,
			)
		}

		if pc.Path == "" {
			return fmt.Errorf("project %q missing path", name)
		}

		_, err := os.Stat(pc.Path)
		if err != nil {
			return err
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

func BuildProjects(config *Config, roots []string) (map[string]Project, error) {
	projects := make(map[string]Project)

	filteredConfigs := make(map[string]ProjectConfig, 0)

	// if user defined projects via flag, only build those
	if len(roots) > 0 {
		for _, r := range roots {
			conf, ok := config.Projects[r]
			if !ok {
				return nil, fmt.Errorf("project %q not found in config", r)
			}
			filteredConfigs[r] = conf
		}
	}

	if len(filteredConfigs) == 0 {
		filteredConfigs = config.Projects
	}

	for name := range filteredConfigs {
		proj, err := buildProject(name, config, projects, map[string]bool{})
		if err != nil {
			return nil, err
		}
		projects[name] = proj
	}

	return projects, nil
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
	case GoProjectType:
		p = NewGoProject(name, cfg.Path)
	case NodeProjectType:
		p = NewNodeProject(name, cfg.Path)
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
