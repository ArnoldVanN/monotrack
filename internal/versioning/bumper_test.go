package versioning

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
)

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		kind       BumpKind
		preRelease bool
		want       string
	}{
		// Stable → stable
		{"patch on stable", "v1.2.3", PatchBump, false, "v1.2.4"},
		{"minor on stable", "v1.2.3", MinorBump, false, "v1.3.0"},
		{"major on stable", "v1.2.3", MajorBump, false, "v2.0.0"},

		// Stable → start prerelease
		{"start prerelease patch", "v1.2.3", PatchBump, true, "v1.2.4-rc.1"},
		{"start prerelease minor", "v1.2.3", MinorBump, true, "v1.3.0-rc.1"},

		// Prerelease → bump rc counter
		{"bump rc counter", "v1.3.0-rc.1", MinorBump, true, "v1.3.0-rc.2"},
		{"bump rc counter patch ignored", "v1.3.0-rc.5", PatchBump, true, "v1.3.0-rc.6"},

		// Prerelease → promote to stable (kind ignored, suffix stripped)
		{"promote from rc", "v1.3.0-rc.3", PatchBump, false, "v1.3.0"},
		{"promote ignores kind", "v1.3.0-rc.3", MajorBump, false, "v1.3.0"},
		{"promote from rc.1", "v2.0.0-rc.1", MinorBump, false, "v2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bumpVersion(tt.in, tt.kind, tt.preRelease)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("bumpVersion(%q, %q, %v) = %q, want %q", tt.in, tt.kind, tt.preRelease, got, tt.want)
			}
		})
	}
}

func TestIsReleaseCommit(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"chore(release): bump 2 project(s)", true},
		{"chore(release): bump\n\n- a v0.1 -> v0.2", true},
		{"  chore(release): leading space", true},
		{"feat: add thing", false},
		{"chore: cleanup", false},
		{"chore(deps): bump foo", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isReleaseCommit(tt.msg); got != tt.want {
			t.Errorf("isReleaseCommit(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}

// TestBumpProjectsPromotesStalePrerelease covers the case where a project's
// latest tag is a prerelease and no source files have changed since that tag.
// With preRelease=false the project should still be picked up and have its
// suffix stripped; with preRelease=true it should remain skipped.
func TestBumpProjectsPromotesStalePrerelease(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}
	dir := t.TempDir()
	t.Chdir(dir)

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	run("config", "tag.gpgsign", "false")

	apiDir := filepath.Join(dir, "apps", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "feat: init api")
	run("tag", "api/v0.1.0-rc.1")

	// Second project keeps the config in multi-project mode so tags use the
	// `<name>/<version>` scheme.
	webDir := filepath.Join(dir, "apps", "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.js"), []byte("// web\n"), 0o644); err != nil {
		t.Fatalf("write web: %v", err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "feat: init web")
	run("tag", "web/v1.0.0")

	api := projects.NewGoProject("api", "apps/api", true, "go")
	web := projects.NewNodeProject("web", "apps/web", true, "node")
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api": {Type: projects.ProjectTypeGo, Path: "apps/api"},
		"web": {Type: projects.ProjectTypeNode, Path: "apps/web"},
	}}
	app.Init(cfg, map[string]projects.Project{"api": api, "web": web})

	head, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}

	b := NewBumper()

	t.Run("promotes when preRelease=false", func(t *testing.T) {
		results, err := b.BumpProjects(app.State.Projects, nil, false, head)
		if err != nil {
			t.Fatalf("BumpProjects: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("results = %d, want 1: %+v", len(results), results)
		}
		got := results[0]
		if got.Project.Name() != "api" {
			t.Errorf("project = %q, want api", got.Project.Name())
		}
		if got.OldVersion != "v0.1.0-rc.1" {
			t.Errorf("OldVersion = %q, want v0.1.0-rc.1", got.OldVersion)
		}
		if got.NewVersion != "v0.1.0" {
			t.Errorf("NewVersion = %q, want v0.1.0", got.NewVersion)
		}
	})

	t.Run("skips when preRelease=true", func(t *testing.T) {
		results, err := b.BumpProjects(app.State.Projects, nil, true, head)
		if err != nil {
			t.Fatalf("BumpProjects: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("results = %d, want 0: %+v", len(results), results)
		}
	})
}

// TestBumpProjectsUsesUnreachableTagsForVersionSelection covers the
// promotion-via-PR workflow: a release tag exists for the project but its
// commit lives on a promotion branch that never merged back into head. The
// bumper must still treat that tag as the latest version — otherwise the
// next bump would compute v0.0.1 (the bootstrap successor) and collide with
// the existing tag on every subsequent run.
func TestBumpProjectsUsesUnreachableTagsForVersionSelection(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}
	dir := t.TempDir()
	t.Chdir(dir)

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	run("config", "tag.gpgsign", "false")

	apiDir := filepath.Join(dir, "apps", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeAPI := func(content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	writeAPI("package main // v0\n")
	run("add", ".")
	run("commit", "-q", "-m", "chore: init api")

	run("checkout", "-q", "-b", "promotion")
	writeAPI("package main // promoted\n")
	run("add", ".")
	run("commit", "-q", "-m", "feat: promoted release")
	run("tag", "api/v0.0.1-rc.1")

	run("checkout", "-q", "main")
	writeAPI("package main // mainline\n")
	run("add", ".")
	run("commit", "-q", "-m", "feat: mainline work")

	// Second project keeps the config in multi-project mode so tags use the
	// `<name>/<version>` scheme.
	webDir := filepath.Join(dir, "apps", "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.js"), []byte("// web\n"), 0o644); err != nil {
		t.Fatalf("write web: %v", err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "feat: init web")
	run("tag", "web/v1.0.0")

	api := projects.NewGoProject("api", "apps/api", true, "go")
	web := projects.NewNodeProject("web", "apps/web", true, "node")
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api": {Type: projects.ProjectTypeGo, Path: "apps/api"},
		"web": {Type: projects.ProjectTypeNode, Path: "apps/web"},
	}}
	app.Init(cfg, map[string]projects.Project{"api": api, "web": web})

	head, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}

	if reachable, err := git.IsAncestor("api/v0.0.1-rc.1", head); err != nil {
		t.Fatalf("IsAncestor: %v", err)
	} else if reachable {
		t.Fatalf("test setup is wrong: api/v0.0.1-rc.1 should not be ancestor of head")
	}

	b := NewBumper()
	results, err := b.BumpProjects(app.State.Projects, nil, true, head)
	if err != nil {
		t.Fatalf("BumpProjects: %v", err)
	}
	var apiResult *BumpResult
	for i := range results {
		if results[i].Project.Name() == "api" {
			apiResult = &results[i]
			break
		}
	}
	if apiResult == nil {
		t.Fatalf("no result for api: %+v", results)
	}
	if apiResult.OldVersion != "v0.0.1-rc.1" {
		t.Errorf("OldVersion = %q, want v0.0.1-rc.1 (unreachable tag must still count)", apiResult.OldVersion)
	}
	if apiResult.NewVersion == "v0.0.1-rc.1" {
		t.Errorf("NewVersion = %q collides with existing tag", apiResult.NewVersion)
	}
}
