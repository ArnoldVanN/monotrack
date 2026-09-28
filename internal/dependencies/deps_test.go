package dependencies

import (
	"reflect"
	"testing"

	"github.com/arnoldvann/monotrack/internal/projects"
)

func TestDiffSeparatesRedundantFromUnverified(t *testing.T) {
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api":       {Type: projects.ProjectTypeGo, Path: "apps/api", DependsOn: []string{"go-shared", "nested", "protos"}},
		"go-shared": {Type: projects.ProjectTypeGo, Path: "packages/go-shared", DependsOn: []string{"nested"}},
		"nested":    {Type: projects.ProjectTypeGo, Path: "packages/nested"},
		"protos":    {Type: projects.ProjectTypeGo, Path: "packages/protos"},
	}}

	// api reaches nested through go-shared, but nothing detects protos.
	got := Diff(cfg, map[string][]string{
		"api":       {"go-shared"},
		"go-shared": {"nested"},
		"nested":    {},
		"protos":    {},
	})

	want := []Change{{
		Project:    "api",
		Redundant:  []Redundant{{Dep: "nested", Via: "go-shared"}},
		Unverified: []string{"protos"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff = %+v, want %+v", got, want)
	}
}

func TestDiffReportsAdditions(t *testing.T) {
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"web": {Type: projects.ProjectTypeNode, Path: "apps/web"},
		"ui":  {Type: projects.ProjectTypeNode, Path: "packages/ui"},
	}}

	got := Diff(cfg, map[string][]string{"web": {"ui"}, "ui": {}})
	want := []Change{{Project: "web", Added: []string{"ui"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff = %+v, want %+v", got, want)
	}
}

func TestTargetKeepsUnverifiedEdgesUnlessPruning(t *testing.T) {
	detected := map[string][]string{"frontend": {}, "protos": {}}
	changes := []Change{{Project: "frontend", Unverified: []string{"protos"}}}

	kept := Target(detected, changes, false)
	if !reflect.DeepEqual(kept["frontend"], []string{"protos"}) {
		t.Errorf("frontend = %v, want [protos] preserved", kept["frontend"])
	}

	pruned := Target(detected, changes, true)
	if len(pruned["frontend"]) != 0 {
		t.Errorf("frontend = %v, want empty under --prune", pruned["frontend"])
	}
}

func TestTargetDropsRedundantEdges(t *testing.T) {
	detected := map[string][]string{"api": {"go-shared"}, "go-shared": {"nested"}, "nested": {}}
	changes := []Change{{Project: "api", Redundant: []Redundant{{Dep: "nested", Via: "go-shared"}}}}

	if got := Target(detected, changes, false)["api"]; !reflect.DeepEqual(got, []string{"go-shared"}) {
		t.Errorf("api = %v, want [go-shared]", got)
	}
}

func TestOwnerPrefersNestedProject(t *testing.T) {
	idx := newIndex(&projects.Config{Projects: map[string]projects.ProjectConfig{
		"outer": {Path: "packages"},
		"inner": {Path: "packages/inner"},
	}})

	if got, _ := idx.owner("packages/inner/sub"); got != "inner" {
		t.Errorf("owner = %q, want inner", got)
	}
	if got, _ := idx.owner("packages/other"); got != "outer" {
		t.Errorf("owner = %q, want outer", got)
	}
}

func TestOwnerOfAbsIgnoresPathsOutsideRepo(t *testing.T) {
	idx := newIndex(&projects.Config{Projects: map[string]projects.ProjectConfig{
		"api": {Path: "apps/api"},
	}})

	if _, ok := idx.ownerOfAbs("/repo", "/home/x/go/pkg/mod/example.com/dep"); ok {
		t.Error("a module-cache path should not map to a project")
	}
	if got, ok := idx.ownerOfAbs("/repo", "/repo/apps/api/internal/svc"); !ok || got != "api" {
		t.Errorf("ownerOfAbs = %q,%v want api,true", got, ok)
	}
}

func TestNormalizeDropsSelfAndDuplicates(t *testing.T) {
	got := normalize([]string{"b", "api", "a", "b", ""}, "api")
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("normalize = %v, want [a b]", got)
	}
}
