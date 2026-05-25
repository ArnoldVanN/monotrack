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

func TestIsIgnored(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		projectPath string
		patterns    []string
		want        bool
	}{
		{"no patterns", "src/main.go", "apps/api", nil, false},
		{"simple ignore match", "apps/api/docs/guide.md", "apps/api", []string{"docs/**"}, true},
		{"simple ignore miss", "apps/api/src/main.go", "apps/api", []string{"docs/**"}, false},
		{"negation re-includes", "apps/api/docs/api.md", "apps/api", []string{"docs/**", "!docs/api.md"}, false},
		{"last match wins", "apps/api/docs/api.md", "apps/api", []string{"docs/**", "!docs/api.md", "docs/api.md"}, true},
		{"exact file", "apps/api/package.json", "apps/api", []string{"package.json"}, true},
		{"exact file miss", "apps/api/tsconfig.json", "apps/api", []string{"package.json"}, false},
		{"recursive extension", "apps/api/nested/dir/README.md", "apps/api", []string{"**/*.md"}, true},
		{"recursive extension miss", "apps/api/nested/dir/main.go", "apps/api", []string{"**/*.md"}, false},
		{"include-only via negation", "apps/web/src/app.ts", "apps/web", []string{"**", "!src/**"}, false},
		{"include-only excludes other", "apps/web/README.md", "apps/web", []string{"**", "!src/**"}, true},
		{"root project", "src/main.go", ".", []string{"docs/**"}, false},
		{"root project match", "docs/guide.md", ".", []string{"docs/**"}, true},
		{"test file pattern", "apps/api/handler_test.go", "apps/api", []string{"**/*_test.go"}, true},
		{"test file pattern miss", "apps/api/handler.go", "apps/api", []string{"**/*_test.go"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIgnored(tt.file, tt.projectPath, tt.patterns)
			if got != tt.want {
				t.Errorf("isIgnored(%q, %q, %v) = %v, want %v", tt.file, tt.projectPath, tt.patterns, got, tt.want)
			}
		})
	}
}

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

func TestListProjectsChangedBetweenCommits_IgnorePatterns(t *testing.T) {
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
	apiDocs := filepath.Join(apiDir, "docs")
	webDir := filepath.Join(dir, "apps", "web")
	webSrc := filepath.Join(webDir, "src")
	for _, d := range []string{apiDir, apiDocs, webDir, webSrc} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webSrc, "index.js"), []byte("// web\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "feat: init")
	base, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}

	// Commit: docs-only change for api, non-src change for web.
	if err := os.WriteFile(filepath.Join(apiDocs, "guide.md"), []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "README.md"), []byte("# Web\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "docs: update")
	head, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}

	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api": {Type: projects.ProjectTypeGo, Path: "apps/api", Ignore: []string{"docs/**"}},
		"web": {Type: projects.ProjectTypeNode, Path: "apps/web", Ignore: []string{"**", "!src/**"}},
	}}
	apiProj := projects.NewGoProject("api", "apps/api", true, "go")
	webProj := projects.NewNodeProject("web", "apps/web", true, "node")
	app.Init(cfg, map[string]projects.Project{"api": apiProj, "web": webProj})

	direct, all, err := ListProjectsChangedBetweenCommits(base, head)
	if err != nil {
		t.Fatalf("ListProjectsChangedBetweenCommits: %v", err)
	}
	if direct["api"] || all["api"] {
		t.Errorf("api should not be flagged (only docs changed, which are ignored)")
	}
	if direct["web"] || all["web"] {
		t.Errorf("web should not be flagged (README.md doesn't match !src/**)")
	}

	// Now commit real source changes — both should be flagged.
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main // v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webSrc, "index.js"), []byte("// web v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "feat: real changes")
	head2, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}

	direct2, all2, err := ListProjectsChangedBetweenCommits(base, head2)
	if err != nil {
		t.Fatalf("ListProjectsChangedBetweenCommits: %v", err)
	}
	if !direct2["api"] || !all2["api"] {
		t.Errorf("api should be flagged (main.go changed)")
	}
	if !direct2["web"] || !all2["web"] {
		t.Errorf("web should be flagged (src/index.js changed)")
	}
}
