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
	Build      BuildConfig `mapstructure:"build"`
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
