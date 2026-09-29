package dependencies

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arnoldvann/monotrack/internal/projects"
)

type goPackage struct {
	ImportPath string
	Dir        string
	Standard   bool
	Error      *goPackageError
	DepsErrors []*goPackageError
}

type goPackageError struct {
	Err string
}

// resolveGo walks the transitive import graph of every package under the
// project, including test-only imports, and maps each package's directory back
// onto the project that owns it. Packages outside the repo (stdlib, module
// cache) drop out of the mapping.
func resolveGo(root string, pc projects.ProjectConfig, idx index) ([]string, error) {
	dirs, err := goPackageDirs(root, pc)
	if err != nil {
		return nil, err
	}
	var deps []string
	for _, dir := range dirs {
		if owner, ok := idx.owner(dir); ok {
			deps = append(deps, owner)
		}
	}
	return deps, nil
}

// goPackageDirs returns the repo-relative directory of every in-repo package
// loaded by `go list -deps -test ./...` in the project.
//
// -e keeps go list from bailing on the first broken package so all load errors
// surface at once; they are still fatal, since a package that failed to load
// contributes no import edges and would silently narrow the result.
func goPackageDirs(root string, pc projects.ProjectConfig) ([]string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "list", "-deps", "-test", "-e", "-json", "./...")
	cmd.Dir = filepath.Join(root, cleanPath(pc.Path))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("go list: %s", msg)
	}

	var (
		dirs    []string
		loadErr []string
	)
	dec := json.NewDecoder(&stdout)
	for {
		var pkg goPackage
		if err := dec.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("parsing go list output: %w", err)
		}

		if pkg.Error != nil {
			loadErr = append(loadErr, fmt.Sprintf("%s: %s", pkg.ImportPath, pkg.Error.Err))
			continue
		}
		if pkg.Standard || pkg.Dir == "" {
			continue
		}
		if rel, ok := relToRoot(root, pkg.Dir); ok {
			dirs = append(dirs, rel)
		}
	}

	if len(loadErr) > 0 {
		return nil, fmt.Errorf("go list reported %d package error(s):\n  %s", len(loadErr), strings.Join(loadErr, "\n  "))
	}
	return dirs, nil
}
