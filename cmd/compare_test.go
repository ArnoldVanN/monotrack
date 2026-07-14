package cmd

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

// TestGetChangedProjects_StaggeredTags covers projects whose latest tags sit on
// different commits. A project tagged further back gets a wider diff range, and
// that range can span another project's changes; only the project owning the
// range may be flagged from it.
//
// Layout:
//
//	c1  init all           <- api/v0.1.0, proxy/v0.1.0
//	c2  change apps/web    <- web/v0.2.0
//	c3  change apps/api    <- head
//
// api's range (c1..c3) covers the web change at c2, but web has not changed
// since web/v0.2.0 and must not be flagged. proxy is flagged via dependsOn: api.
func TestGetChangedProjects_StaggeredTags(t *testing.T) {
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
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")

	write("apps/api/main.go", "package main\n")
	write("apps/web/index.js", "// web\n")
	write("apps/proxy/main.go", "package main\n")
	run("add", ".")
	run("commit", "-q", "-m", "feat: init")
	run("tag", "api/v0.1.0")
	run("tag", "proxy/v0.1.0")

	write("apps/web/index.js", "// web v2\n")
	run("add", ".")
	run("commit", "-q", "-m", "feat: web change")
	run("tag", "web/v0.2.0")

	write("apps/api/main.go", "package main // v2\n")
	run("add", ".")
	run("commit", "-q", "-m", "feat: api change")
	head, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}

	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api":   {Type: projects.ProjectTypeGo, Path: "apps/api"},
		"web":   {Type: projects.ProjectTypeNode, Path: "apps/web"},
		"proxy": {Type: projects.ProjectTypeGo, Path: "apps/proxy", DependsOn: []string{"api"}},
	}}
	app.Init(cfg, map[string]projects.Project{
		"api":   projects.NewGoProject("api", "apps/api", true, "go"),
		"web":   projects.NewNodeProject("web", "apps/web", true, "node"),
		"proxy": projects.NewGoProject("proxy", "apps/proxy", true, "go"),
	})

	changed, err := getChangedProjects(map[string]string{
		"api":   "v0.1.0",
		"web":   "v0.2.0",
		"proxy": "v0.1.0",
	}, head)
	if err != nil {
		t.Fatalf("getChangedProjects: %v", err)
	}

	if !changed["api"] {
		t.Errorf("api should be flagged (main.go changed since api/v0.1.0): %+v", changed)
	}
	if !changed["proxy"] {
		t.Errorf("proxy should be flagged (dependsOn api, which changed since proxy/v0.1.0): %+v", changed)
	}
	if changed["web"] {
		t.Errorf("web should not be flagged: nothing under apps/web changed since web/v0.2.0, "+
			"it only falls inside api's wider diff range: %+v", changed)
	}
}
