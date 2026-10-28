package projects

import (
	"maps"
	"slices"
	"testing"
)

func TestBuildProjects(t *testing.T) {
	entry := BuildConfig{Entrypoint: true}
	cfg := &Config{Projects: map[string]ProjectConfig{
		"web":    {Type: ProjectTypeNode, Path: "apps/web", Build: entry, DependsOn: []string{"api", "lib"}},
		"api":    {Type: ProjectTypeGo, Path: "apps/api", Build: entry, DependsOn: []string{"lib"}},
		"worker": {Type: ProjectTypeGo, Path: "apps/worker", Build: entry},
		"lib":    {Type: ProjectTypeGo, Path: "libs/lib"},
	}}

	tests := []struct {
		roots []string
		want  []string
	}{
		{nil, []string{"api", "web", "worker"}},
		{[]string{"web"}, []string{"web"}},
		{[]string{"api", "worker"}, []string{"api", "worker"}},
		{[]string{"lib"}, nil},
	}
	for _, tt := range tests {
		got, err := BuildProjects(cfg, tt.roots)
		if err != nil {
			t.Fatalf("BuildProjects(%v): %v", tt.roots, err)
		}
		if names := slices.Sorted(maps.Keys(got)); !slices.Equal(names, tt.want) {
			t.Errorf("BuildProjects(%v) = %v, want %v", tt.roots, names, tt.want)
		}
	}

	got, err := BuildProjects(cfg, []string{"web"})
	if err != nil {
		t.Fatal(err)
	}
	var deps []string
	for _, d := range got["web"].ListDependencies() {
		deps = append(deps, d.Name())
	}
	if !slices.Equal(deps, []string{"api", "lib"}) {
		t.Errorf("web dependencies = %v, want [api lib]", deps)
	}
	if len(cfg.Projects) != 4 {
		t.Errorf("config was modified: %v", slices.Sorted(maps.Keys(cfg.Projects)))
	}

	if _, err := BuildProjects(cfg, []string{"nope"}); err == nil {
		t.Error("unknown project: want error")
	}
}
