package projects

import (
	"maps"
	"slices"
	"testing"
)

func TestBuildProjects(t *testing.T) {
	cfg := &Config{Projects: map[string]ProjectConfig{
		"web":    {Type: ProjectTypeNode, Path: "apps/web", Versioned: true, DependsOn: []string{"api", "lib"}},
		"api":    {Type: ProjectTypeGo, Path: "apps/api", Versioned: true, DependsOn: []string{"lib"}},
		"worker": {Type: ProjectTypeGo, Path: "apps/worker", Versioned: true},
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

func TestMigrateDeprecatedEntrypoint(t *testing.T) {
	cfg := &Config{Projects: map[string]ProjectConfig{
		"web": {Type: ProjectTypeNode, Path: "apps/web", Build: BuildConfig{Entrypoint: true}},
		"api": {Type: ProjectTypeGo, Path: "apps/api", Versioned: true},
		"lib": {Type: ProjectTypeGo, Path: "libs/lib"},
	}}
	if got := cfg.MigrateDeprecated(); !slices.Equal(got, []string{"web"}) {
		t.Errorf("MigrateDeprecated = %v, want [web]", got)
	}
	for name, want := range map[string]bool{"web": true, "api": true, "lib": false} {
		if pc := cfg.Projects[name]; pc.Versioned != want || pc.Build.Entrypoint {
			t.Errorf("%s = %+v, want versioned %v", name, pc, want)
		}
	}
}
