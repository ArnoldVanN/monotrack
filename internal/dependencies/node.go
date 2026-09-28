package dependencies

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arnoldvann/monotrack/internal/projects"
)

type packageJSON struct {
	Name                 string            `json:"name"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

// resolveNode matches the project's declared dependencies against the package
// names of the other node projects in the config. Workspace deps are named by
// package name under every package manager, so the version range (workspace:*,
// file:, a semver range) doesn't need interpreting.
func resolveNode(name string, pc projects.ProjectConfig, cfg *projects.Config, idx index) ([]string, error) {
	self, err := readPackageJSON(cleanPath(pc.Path))
	if err != nil {
		return nil, err
	}

	byPackageName := make(map[string]string)
	for other, opc := range cfg.Projects {
		if opc.Type != projects.ProjectTypeNode || other == name {
			continue
		}
		pkg, err := readPackageJSON(cleanPath(opc.Path))
		if err != nil {
			return nil, err
		}
		if pkg.Name != "" {
			byPackageName[pkg.Name] = other
		}
	}

	var deps []string
	for _, set := range []map[string]string{
		self.Dependencies, self.DevDependencies,
		self.PeerDependencies, self.OptionalDependencies,
	} {
		for dep := range set {
			if owner, ok := byPackageName[dep]; ok {
				deps = append(deps, owner)
			}
		}
	}
	return deps, nil
}

func readPackageJSON(dir string) (*packageJSON, error) {
	path := filepath.Join(dir, "package.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var pkg packageJSON
	if err := json.Unmarshal(b, &pkg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &pkg, nil
}
