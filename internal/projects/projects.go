package projects

import (
	"fmt"
	"os"
	"slices"
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

func BuildProjects(cfg *Config, p []string) (map[string]Project, error) {
	projects := make(map[string]Project)

	for name := range cfg.Projects {
		// if user defined projects via flag, only build those
		if len(p) > 0 {
			found := slices.Contains(p, name)
			if !found {
				continue
			}
		}

		proj, err := buildProject(name, cfg, projects, map[string]bool{})
		if err != nil {
			return nil, err
		}
		projects[name] = proj
	}

	return projects, nil
}

func buildProject(name string, cfg *Config, built map[string]Project, visiting map[string]bool) (Project, error) {
	if proj, exists := built[name]; exists {
		return proj, nil // already built
	}

	pc, ok := cfg.Projects[name]
	if !ok {
		return nil, fmt.Errorf("project %q not found in config", name)
	}

	if pc.Type == GoProjectType {
		if visiting[name] {
			return nil, fmt.Errorf("circular dependency detected on project %q", name)
		}
	}
	visiting[name] = true

	var p Project
	switch pc.Type {
	case GoProjectType:
		p = NewGoProject(name, pc.Path)
	case NodeProjectType:
		p = NewNodeProject(name, pc.Path)
	default:
		return nil, fmt.Errorf("unsupported project type %q", pc.Type)
	}

	for _, depName := range pc.DependsOn {
		depProj, err := buildProject(string(depName), cfg, built, visiting)
		if err != nil {
			return nil, err
		}
		p.AddDependency(depProj)
	}

	built[name] = p
	delete(visiting, name) // finished visiting
	return p, nil
}
