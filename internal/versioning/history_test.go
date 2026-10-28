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

func TestHistory(t *testing.T) {
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
	commit := func(path, content, msg string) {
		t.Helper()
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		run("add", ".")
		run("commit", "-q", "-m", msg)
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	run("config", "tag.gpgsign", "false")

	commit("apps/web/index.js", "// web\n", "chore: init web")
	commit("libs/lib/index.js", "// lib\n", "chore: init lib")
	run("tag", "web/v1.0.0")
	run("tag", "lib/v1.0.0")

	commit("libs/lib/index.js", "// lib 1.1\n", "feat: lib work")
	run("tag", "lib/v1.1.0")
	run("tag", "web/v1.0.1-rc.1")

	commit("apps/web/index.js", "// web 1.1\n", "feat: web work")
	run("tag", "web/v1.1.0-rc.1")
	commit("apps/web/index.js", "// web 1.1 fixed\n", "fix: web bug")
	run("tag", "-a", "-m", "Release web/v1.1.0", "web/v1.1.0")

	// a tag on a branch that never merged back is not part of the history
	run("checkout", "-q", "-b", "side")
	commit("apps/web/side.js", "// side\n", "feat: side work")
	run("tag", "web/v9.0.0")
	run("checkout", "-q", "main")

	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"web": {Type: projects.ProjectTypeNode, Path: "apps/web", DependsOn: []string{"lib"}},
		"lib": {Type: projects.ProjectTypeNode, Path: "libs/lib"},
	}}
	web := projects.NewNodeProject("web", "apps/web", true, "node")
	app.Init(cfg, map[string]projects.Project{"web": web})

	head, err := git.GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}
	releases, err := History(app.State.Projects, head)
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	type want struct {
		version, old string
		reason       BumpReason
		subjects     []string
		deps         []DepUpdate
	}
	wants := []want{
		{version: "v1.1.0", old: "v1.1.0-rc.1", reason: ReasonPromotion, subjects: []string{"fix: web bug", "feat: web work"}},
		{version: "v1.1.0-rc.1", old: "v1.0.1-rc.1", reason: ReasonCommits, subjects: []string{"feat: web work"}},
		{version: "v1.0.1-rc.1", old: "v1.0.0", reason: ReasonDependency, deps: []DepUpdate{{Name: "lib", Version: "v1.1.0"}}},
		{version: "v1.0.0", reason: ReasonInitial},
	}
	if len(releases) != len(wants) {
		var got []string
		for _, r := range releases {
			got = append(got, r.Tag)
		}
		t.Fatalf("releases = %v, want %d", got, len(wants))
	}
	for i, w := range wants {
		r := releases[i]
		if r.NewVersion != w.version || r.OldVersion != w.old || r.Reason != w.reason {
			t.Errorf("release %d = %s from %q (%s), want %s from %q (%s)", i, r.NewVersion, r.OldVersion, r.Reason, w.version, w.old, w.reason)
		}
		var subjects []string
		for _, c := range r.Commits {
			subjects = append(subjects, c.Subject)
		}
		if strings.Join(subjects, "|") != strings.Join(w.subjects, "|") {
			t.Errorf("%s commits = %q, want %q", w.version, subjects, w.subjects)
		}
		if len(r.Dependencies) != len(w.deps) || (len(w.deps) > 0 && r.Dependencies[0] != w.deps[0]) {
			t.Errorf("%s dependencies = %+v, want %+v", w.version, r.Dependencies, w.deps)
		}
	}
	if releases[0].Date.IsZero() || releases[0].Tag != "web/v1.1.0" {
		t.Errorf("v1.1.0 tag = %q, date = %v", releases[0].Tag, releases[0].Date)
	}
}
