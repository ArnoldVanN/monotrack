package config

import (
	"fmt"

	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/spf13/viper"
)

var cfg projects.Config

func LoadConfig(configPath string) (*projects.Config, error) {
	if configPath == "" {
		configPath = "monotrack.yaml"
	}

	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", configPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate %q: %w", configPath, err)
	}

	return &cfg, nil
}

// # NOTE : probably not needed
func detectCycles(projects map[string]projects.ProjectConfig) error {
	visited := map[string]bool{}
	stack := map[string]bool{}

	var visit func(string) error
	visit = func(n string) error {
		if stack[n] {
			return fmt.Errorf("dependency cycle detected at %s", n)
		}
		if visited[n] {
			return nil
		}

		visited[n] = true
		stack[n] = true

		for _, dep := range projects[n].DependsOn {
			if err := visit(string(dep)); err != nil {
				return err
			}
		}

		stack[n] = false
		return nil
	}

	for name := range projects {
		if err := visit(name); err != nil {
			return err
		}
	}

	return nil
}
