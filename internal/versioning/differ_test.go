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

// TestListProjectsChangedBetweenCommits_IgnoresChangelogOnly ensures that a
// commit which only touches monotrack-managed files (CHANGELOG.md) does not
// flag the project as changed. Otherwise every project whose CHANGELOG was
// rewritten gets a spurious bump on the next run.
func TestListProjectsChangedBetweenCommits_IgnoresChangelogOnly(t *testing.T) {
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

	apiDir := filepath.Join(dir, "apps", "api")
	webDir := filepath.Join(dir, "apps", "web")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.js"), []byte("// web\n"), 0o644); err != nil {
		t.Fatalf("write web: %v", err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "feat: init")
	base, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}

	// CHANGELOG-only edit for api; real source edit for web.
	if err := os.WriteFile(filepath.Join(apiDir, "CHANGELOG.md"), []byte("# Changelog\n"), 0o644); err != nil {
		t.Fatalf("write api changelog: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.js"), []byte("// web v2\n"), 0o644); err != nil {
		t.Fatalf("rewrite web: %v", err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "chore: changelog + web tweak")
	head, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead head: %v", err)
	}

	api := projects.NewGoProject("api", "apps/api", true, "go")
	web := projects.NewNodeProject("web", "apps/web", true, "node")
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api": {Type: projects.ProjectTypeGo, Path: "apps/api"},
		"web": {Type: projects.ProjectTypeNode, Path: "apps/web"},
	}}
	app.Init(cfg, map[string]projects.Project{"api": api, "web": web})

	direct, all, err := ListProjectsChangedBetweenCommits(base, head)
	if err != nil {
		t.Fatalf("ListProjectsChangedBetweenCommits: %v", err)
	}
	if direct["api"] || all["api"] {
		t.Errorf("api was flagged as changed but only its CHANGELOG.md was touched: direct=%+v all=%+v", direct, all)
	}
	if !direct["web"] || !all["web"] {
		t.Errorf("web should be flagged (source file changed): direct=%+v all=%+v", direct, all)
	}
}
