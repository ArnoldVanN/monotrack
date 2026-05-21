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

