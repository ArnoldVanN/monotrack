package versioning

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arnoldvann/monotrack/internal/dependencies"
	"github.com/arnoldvann/monotrack/internal/projects"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPropagatePackageGranularity(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available on PATH")
	}
	root := t.TempDir()
	t.Setenv("GOWORK", "off")

	apiMain := "package main\n\nimport _ \"example.com/shared/a\"\n\nfunc main() {}\n"
	apiMod := "module example.com/%s\n\ngo 1.22\n\nrequire example.com/shared v0.0.0\n\nreplace example.com/shared => ../../packages/shared\n"
	writeTree(t, root, map[string]string{
		"packages/shared/go.mod": "module example.com/shared\n\ngo 1.22\n",
		"packages/shared/a/a.go": "package a\n\nimport _ \"example.com/shared/c\"\n",
		"packages/shared/b/b.go": "package b\n",
		"packages/shared/c/c.go": "package c\n",
		"packages/other/go.mod":  "module example.com/other\n\ngo 1.22\n",
		"packages/other/o.go":    "package other\n",
		"apps/api/go.mod":        fmt.Sprintf(apiMod, "api"),
		"apps/api/main.go":       apiMain,
		"apps/legacy/go.mod":     fmt.Sprintf(apiMod, "legacy"),
		"apps/legacy/main.go":    apiMain,
	})

	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api":    {Type: projects.ProjectTypeGo, Path: "apps/api", DependsOn: []string{"shared", "other"}, Granularity: projects.GranularityPackage},
		"legacy": {Type: projects.ProjectTypeGo, Path: "apps/legacy", DependsOn: []string{"shared"}},
		"shared": {Type: projects.ProjectTypeGo, Path: "packages/shared", DependsOn: []string{"chart"}},
		"other":  {Type: projects.ProjectTypeGo, Path: "packages/other"},
		"chart":  {Type: projects.ProjectTypeHelm, Path: "packages/chart"},
	}}
	pkgs, err := dependencies.LoadGoPackages(root, cfg)
	if err != nil {
		t.Fatalf("LoadGoPackages: %v", err)
	}

	tests := []struct {
		name string
		diff []string
		api  bool
	}{
		{"unimported package", []string{"packages/shared/b/b.go"}, false},
		{"imported package", []string{"packages/shared/a/a.go"}, true},
		{"transitively imported package", []string{"packages/shared/c/c.go"}, true},
		{"module file", []string{"packages/shared/go.mod"}, true},
		{"file outside any package", []string{"packages/shared/a/assets/logo.txt"}, true},
		{"deleted package", []string{"packages/shared/gone/gone.go"}, true},
		{"edge without import", []string{"packages/other/o.go"}, true},
		{"non-go edge behind an import", []string{"packages/chart/Chart.yaml"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := changedFiles(cfg, tt.diff)
			all := propagate(cfg, files, pkgs)
			if all["api"] != tt.api {
				t.Errorf("api changed = %v, want %v (all=%v)", all["api"], tt.api, all)
			}
			if tt.diff[0] != "packages/other/o.go" && !all["legacy"] {
				t.Errorf("legacy uses project granularity and should always change: %v", all)
			}
			if fallback := propagate(cfg, files, nil); !fallback["api"] {
				t.Errorf("without package data api should fall back to project granularity: %v", fallback)
			}
		})
	}
}
