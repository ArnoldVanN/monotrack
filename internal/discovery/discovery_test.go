package discovery

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arnoldvann/monotrack/internal/projects"
)

func mapFS(files map[string]string) (fstest.MapFS, []string) {
	fsys := fstest.MapFS{}
	var paths []string
	for p, c := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(c)}
		paths = append(paths, p)
	}
	slices.Sort(paths)
	return fsys, paths
}

func TestDiscover(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		want      map[string]projects.ProjectConfig
		wantNotes []string
	}{
		{
			name: "one project per manifest, named after its directory",
			files: map[string]string{
				"apps/api/go.mod":          "module api",
				"apps/api/main.go":         "package main",
				"apps/web/package.json":    `{"name": "@org/web"}`,
				"charts/app/Chart.yaml":    "name: app",
				"packages/ui/package.json": `{"name": "@org/ui"}`,
			},
			want: map[string]projects.ProjectConfig{
				"api": {Type: projects.ProjectTypeGo, Path: "apps/api"},
				"web": {Type: projects.ProjectTypeNode, Path: "apps/web"},
				"app": {Type: projects.ProjectTypeHelm, Path: "charts/app"},
				"ui":  {Type: projects.ProjectTypeNode, Path: "packages/ui"},
			},
		},
		{
			name: "colliding names widen to the parent directory",
			files: map[string]string{
				"apps/api/go.mod":     "module a",
				"services/api/go.mod": "module b",
				"libs/log/go.mod":     "module c",
			},
			want: map[string]projects.ProjectConfig{
				"apps-api":     {Type: projects.ProjectTypeGo, Path: "apps/api"},
				"services-api": {Type: projects.ProjectTypeGo, Path: "services/api"},
				"log":          {Type: projects.ProjectTypeGo, Path: "libs/log"},
			},
		},
		{
			name: "names that cannot widen apart are numbered",
			files: map[string]string{
				"a b/go.mod": "module a",
				"a-b/go.mod": "module b",
			},
			want: map[string]projects.ProjectConfig{
				"a-b":   {Type: projects.ProjectTypeGo, Path: "a b"},
				"a-b-2": {Type: projects.ProjectTypeGo, Path: "a-b"},
			},
		},
		{
			name: "workspace roots are skipped",
			files: map[string]string{
				"package.json":              `{"private": true, "workspaces": ["packages/*"]}`,
				"packages/ui/package.json":  `{"name": "ui"}`,
				"tools/package.json":        `{"name": "tools"}`,
				"tools/pnpm-workspace.yaml": "packages: []",
				"tools/a/package.json":      `{"name": "a"}`,
			},
			want: map[string]projects.ProjectConfig{
				"ui": {Type: projects.ProjectTypeNode, Path: "packages/ui"},
				"a":  {Type: projects.ProjectTypeNode, Path: "tools/a"},
			},
			wantNotes: []string{
				"skipped package.json: node workspace root",
				"skipped tools/package.json: node workspace root",
			},
		},
		{
			name: "packaged subcharts, fixtures and committed dependencies are skipped",
			files: map[string]string{
				"deploy/umbrella/Chart.yaml":              "name: umbrella",
				"deploy/umbrella/charts/redis/Chart.yaml": "name: redis",
				"apps/api/go.mod":                         "module api",
				"apps/api/testdata/mod/go.mod":            "module fixture",
				"apps/web/package.json":                   `{}`,
				"apps/web/node_modules/x/package.json":    `{}`,
			},
			want: map[string]projects.ProjectConfig{
				"umbrella": {Type: projects.ProjectTypeHelm, Path: "deploy/umbrella"},
				"api":      {Type: projects.ProjectTypeGo, Path: "apps/api"},
				"web":      {Type: projects.ProjectTypeNode, Path: "apps/web"},
			},
			wantNotes: []string{
				"skipped apps/api/testdata/mod: inside a testdata directory",
				"skipped apps/web/node_modules/x: inside a node_modules directory",
				"skipped deploy/umbrella/charts/redis: packaged subchart of deploy/umbrella",
			},
		},
		{
			name: "a directory with several manifests keeps one type",
			files: map[string]string{
				"apps/api/go.mod":       "module api",
				"apps/api/package.json": `{}`,
			},
			want: map[string]projects.ProjectConfig{
				"api": {Type: projects.ProjectTypeGo, Path: "apps/api"},
			},
			wantNotes: []string{"apps/api has go.mod and package.json; treated as go"},
		},
		{
			name: "a lone root project is named after the repository",
			files: map[string]string{
				"go.mod":  "module x",
				"main.go": "package main",
			},
			want: map[string]projects.ProjectConfig{
				"my-repo": {Type: projects.ProjectTypeGo, Path: "."},
			},
		},
		{
			name: "a root project next to others is skipped",
			files: map[string]string{
				"go.mod":           "module x",
				"tools/gen/go.mod": "module gen",
			},
			want: map[string]projects.ProjectConfig{
				"gen": {Type: projects.ProjectTypeGo, Path: "tools/gen"},
			},
			wantNotes: []string{"skipped the repository root (go): a root project is only allowed when it is the only project"},
		},
		{
			name:  "nothing found",
			files: map[string]string{"README.md": "hi"},
			want:  map[string]projects.ProjectConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys, files := mapFS(tt.files)
			got, err := Discover(fsys, files, Options{RootName: "my-repo"})
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			if !reflect.DeepEqual(got.Projects, tt.want) {
				t.Errorf("Projects = %+v, want %+v", got.Projects, tt.want)
			}
			if !slices.Equal(got.Notes, tt.wantNotes) {
				t.Errorf("Notes = %q, want %q", got.Notes, tt.wantNotes)
			}
		})
	}
}

func TestDiscoverIgnoresDeletedFiles(t *testing.T) {
	fsys, _ := mapFS(map[string]string{"apps/api/go.mod": "module api"})
	got, err := Discover(fsys, []string{"apps/api/go.mod", "apps/old/go.mod"}, Options{RootName: "repo"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if _, ok := got.Projects["old"]; ok {
		t.Errorf("Projects = %+v, want apps/old skipped", got.Projects)
	}
}

func TestRender(t *testing.T) {
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"web":  {Type: projects.ProjectTypeNode, Path: "apps/web", Build: projects.BuildConfig{Entrypoint: true}, DependsOn: []string{"ui"}},
		"ui":   {Type: projects.ProjectTypeNode, Path: "packages/ui"},
		"true": {Type: projects.ProjectTypeGo, Path: "odd/true"},
	}}

	got, err := Render(cfg)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := `projects:
  "true":
    type: go
    path: odd/true
  ui:
    type: node
    path: packages/ui
  web:
    type: node
    path: apps/web
    build:
      entrypoint: true
    dependsOn:
      - ui
`
	if string(got) != want {
		t.Errorf("Render =\n%s\nwant\n%s", got, want)
	}
}

func writeFiles(t *testing.T, files map[string]string) {
	t.Helper()
	for p, c := range files {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenerateFillsDependsOnAndEntrypoints(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available on PATH")
	}
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=mod")

	writeFiles(t, map[string]string{
		"apps/api/go.mod":           "module example.com/api\n\ngo 1.22\n\nrequire example.com/shared v0.0.0\n\nreplace example.com/shared => ../../packages/shared\n",
		"apps/api/main.go":          "package main\n\nimport _ \"example.com/shared\"\n\nfunc main() {}\n",
		"packages/shared/go.mod":    "module example.com/shared\n\ngo 1.22\n",
		"packages/shared/shared.go": "package shared\n",
		"apps/web/package.json":     `{"name": "web", "dependencies": {"ui": "workspace:*"}}`,
		"packages/ui/package.json":  `{"name": "ui"}`,
	})

	res := &Result{Projects: map[string]projects.ProjectConfig{
		"api":    {Type: projects.ProjectTypeGo, Path: "apps/api"},
		"shared": {Type: projects.ProjectTypeGo, Path: "packages/shared"},
		"web":    {Type: projects.ProjectTypeNode, Path: "apps/web"},
		"ui":     {Type: projects.ProjectTypeNode, Path: "packages/ui"},
	}}

	cfg, _, err := Generate(root, res)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	for name, want := range map[string]struct {
		entrypoint bool
		deps       []string
	}{
		"api":    {true, []string{"shared"}},
		"shared": {false, nil},
		"web":    {true, []string{"ui"}},
		"ui":     {false, nil},
	} {
		pc := cfg.Projects[name]
		if pc.Build.Entrypoint != want.entrypoint {
			t.Errorf("%s entrypoint = %v, want %v", name, pc.Build.Entrypoint, want.entrypoint)
		}
		if len(pc.DependsOn) != len(want.deps) || (len(want.deps) > 0 && !slices.Equal(pc.DependsOn, want.deps)) {
			t.Errorf("%s dependsOn = %v, want %v", name, pc.DependsOn, want.deps)
		}
	}
}

func TestGenerateFallsBackWhenDetectionFails(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writeFiles(t, map[string]string{
		"apps/web/package.json":    `{"name": "web", "dependencies": {"ui": "*"}}`,
		"packages/ui/package.json": `not json`,
	})

	res := &Result{Projects: map[string]projects.ProjectConfig{
		"web": {Type: projects.ProjectTypeNode, Path: "apps/web"},
		"ui":  {Type: projects.ProjectTypeNode, Path: "packages/ui"},
	}}

	cfg, _, err := Generate(root, res)
	if err == nil || !strings.Contains(err.Error(), "detecting dependencies") {
		t.Fatalf("Generate err = %v, want a detection error", err)
	}
	for name, pc := range cfg.Projects {
		if !pc.Build.Entrypoint || len(pc.DependsOn) != 0 {
			t.Errorf("%s = %+v, want an entrypoint without dependsOn", name, pc)
		}
	}
}

func TestDiscoverAgainstExistingConfig(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		opts    Options
		want    map[string]projects.ProjectConfig
		wantErr string
	}{
		{
			name: "configured directories are not reported, whatever their name",
			files: map[string]string{
				"apps/api/go.mod":       "module api",
				"apps/web/package.json": `{}`,
			},
			opts: Options{Existing: map[string]projects.ProjectConfig{
				"backend": {Type: projects.ProjectTypeGo, Path: "apps/api"},
			}},
			want: map[string]projects.ProjectConfig{
				"web": {Type: projects.ProjectTypeNode, Path: "apps/web"},
			},
		},
		{
			name: "a new project widens around a configured name",
			files: map[string]string{
				"apps/api/go.mod":     "module a",
				"services/api/go.mod": "module b",
			},
			opts: Options{Existing: map[string]projects.ProjectConfig{
				"api": {Type: projects.ProjectTypeGo, Path: "apps/api"},
			}},
			want: map[string]projects.ProjectConfig{
				"services-api": {Type: projects.ProjectTypeGo, Path: "services/api"},
			},
		},
		{
			name:  "a new project that cannot widen is numbered around a configured name",
			files: map[string]string{"api/go.mod": "module a"},
			opts: Options{Existing: map[string]projects.ProjectConfig{
				"api": {Type: projects.ProjectTypeGo, Path: "apps/api"},
			}},
			want: map[string]projects.ProjectConfig{
				"api-2": {Type: projects.ProjectTypeGo, Path: "api"},
			},
		},
		{
			name: "excluded directories are not reported",
			files: map[string]string{
				"tools/gen/go.mod":      "module gen",
				"tools/lint/go.mod":     "module lint",
				"apps/web/package.json": `{}`,
			},
			opts: Options{Exclude: []string{"tools/**"}},
			want: map[string]projects.ProjectConfig{
				"web": {Type: projects.ProjectTypeNode, Path: "apps/web"},
			},
		},
		{
			name: "a configured root project blocks additions",
			files: map[string]string{
				"go.mod":            "module x",
				"deploy/Chart.yaml": "name: x",
			},
			opts: Options{Existing: map[string]projects.ProjectConfig{
				"x": {Type: projects.ProjectTypeGo, Path: "."},
			}},
			wantErr: `project "x" is at the repository root, so no other project can be added (found deploy)`,
		},
		{
			name:  "a configured root project alone is fine",
			files: map[string]string{"go.mod": "module x"},
			opts: Options{Existing: map[string]projects.ProjectConfig{
				"x": {Type: projects.ProjectTypeGo, Path: "."},
			}},
			want: map[string]projects.ProjectConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys, files := mapFS(tt.files)
			got, err := Discover(fsys, files, tt.opts)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Discover err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			if !reflect.DeepEqual(got.Projects, tt.want) {
				t.Errorf("Projects = %+v, want %+v", got.Projects, tt.want)
			}
		})
	}
}

func TestAddProjectsPreservesDocument(t *testing.T) {
	src := `name: main # label
projects:
    # the api
    api:
        type: go
        path: apps/api
`
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api": {Type: projects.ProjectTypeGo, Path: "apps/api"},
		"web": {Type: projects.ProjectTypeNode, Path: "apps/web", Build: projects.BuildConfig{Entrypoint: true}, DependsOn: []string{"ui"}},
		"ui":  {Type: projects.ProjectTypeNode, Path: "packages/ui"},
	}}

	got, err := AddProjects([]byte(src), cfg, []string{"web", "ui"})
	if err != nil {
		t.Fatalf("AddProjects: %v", err)
	}
	want := src + `    ui:
        type: node
        path: packages/ui
    web:
        type: node
        path: apps/web
        build:
            entrypoint: true
        dependsOn:
            - ui
`
	if string(got) != want {
		t.Errorf("AddProjects =\n%s\nwant\n%s", got, want)
	}
}

func TestCompleteLeavesConfiguredProjectsAlone(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writeFiles(t, map[string]string{
		"apps/web/package.json":        `{"name": "web", "dependencies": {"ui": "workspace:*"}}`,
		"apps/docs/package.json":       `{"name": "docs"}`,
		"packages/ui/package.json":     `{"name": "ui", "dependencies": {"tokens": "workspace:*"}}`,
		"packages/tokens/package.json": `{"name": "tokens"}`,
	})

	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"web":    {Type: projects.ProjectTypeNode, Path: "apps/web", Build: projects.BuildConfig{Entrypoint: true}},
		"docs":   {Type: projects.ProjectTypeNode, Path: "apps/docs", Build: projects.BuildConfig{Entrypoint: true}},
		"ui":     {Type: projects.ProjectTypeNode, Path: "packages/ui"},
		"tokens": {Type: projects.ProjectTypeNode, Path: "packages/tokens"},
	}}

	notes, err := Complete(root, cfg, []string{"ui", "tokens"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := cfg.Projects["ui"]; got.Build.Entrypoint || !slices.Equal(got.DependsOn, []string{"tokens"}) {
		t.Errorf("ui = %+v, want a non-entrypoint depending on tokens", got)
	}
	if got := cfg.Projects["tokens"]; got.Build.Entrypoint || len(got.DependsOn) != 0 {
		t.Errorf("tokens = %+v, want a non-entrypoint leaf", got)
	}
	if got := cfg.Projects["web"]; !got.Build.Entrypoint || len(got.DependsOn) != 0 {
		t.Errorf("web = %+v, want it untouched", got)
	}
	want := []string{"web now depends on ui; run `monotrack deps sync` to record it"}
	if !slices.Equal(notes, want) {
		t.Errorf("notes = %q, want %q", notes, want)
	}
}
