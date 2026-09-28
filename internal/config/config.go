package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/spf13/viper"
)

var cfg projects.Config

func LoadConfig(configPath string) (*projects.Config, error) {
	if configPath == "" {
		configPath = "monotrack.yaml"
	}

	if err := checkConfigFile(configPath); err != nil {
		return nil, err
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

// checkConfigFile front-runs viper, which reports a missing extension as
// `Unsupported Config Type ""` without naming the file it choked on.
func checkConfigFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file %q not found; run `monotrack init` to create one, or point at an existing one with -f", path)
		}
		return fmt.Errorf("config file %q: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("config file %q is a directory", path)
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if slices.Contains(viper.SupportedExts, ext) {
		return nil
	}
	if ext == "" {
		return fmt.Errorf("config file %q has no file extension; monotrack needs one to know the format (e.g. %s.yaml)", path, path)
	}
	return fmt.Errorf("config file %q has unsupported extension %q (supported: %s)", path, ext, strings.Join(viper.SupportedExts, ", "))
}
