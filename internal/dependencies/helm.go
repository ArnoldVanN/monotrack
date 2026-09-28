package dependencies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arnoldvann/monotrack/internal/projects"
	"go.yaml.in/yaml/v3"
)

type chartYAML struct {
	Name         string `yaml:"name"`
	Dependencies []struct {
		Name       string `yaml:"name"`
		Repository string `yaml:"repository"`
	} `yaml:"dependencies"`
}

// resolveHelm reads the chart's subchart list. A file:// repository is
// resolved as a path relative to the chart directory; anything else is matched
// by chart name against the other helm projects, which covers umbrella charts
// pulling a sibling from a registry.
func resolveHelm(root, name string, pc projects.ProjectConfig, cfg *projects.Config, idx index) ([]string, error) {
	dir := cleanPath(pc.Path)
	self, err := readChartYAML(dir)
	if err != nil {
		return nil, err
	}

	byChartName := make(map[string]string)
	for other, opc := range cfg.Projects {
		if opc.Type != projects.ProjectTypeHelm || other == name {
			continue
		}
		chart, err := readChartYAML(cleanPath(opc.Path))
		if err != nil {
			return nil, err
		}
		if chart.Name != "" {
			byChartName[chart.Name] = other
		}
	}

	var deps []string
	for _, d := range self.Dependencies {
		if target, ok := strings.CutPrefix(d.Repository, "file://"); ok {
			abs := target
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(root, dir, target)
			}
			if owner, ok := idx.ownerOfAbs(root, filepath.Clean(abs)); ok {
				deps = append(deps, owner)
			}
			continue
		}
		if owner, ok := byChartName[d.Name]; ok {
			deps = append(deps, owner)
		}
	}
	return deps, nil
}

func readChartYAML(dir string) (*chartYAML, error) {
	path := filepath.Join(dir, "Chart.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var chart chartYAML
	if err := yaml.Unmarshal(b, &chart); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &chart, nil
}
