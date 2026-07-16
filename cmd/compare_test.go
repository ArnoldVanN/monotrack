package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/projects"
)

// repo is a throwaway git repo; compare shells out to git, so these tests
// drive real history.
type repo struct {
	t   *testing.T
	dir string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}
	dir := t.TempDir()
	t.Chdir(dir)

	r := &repo{t: t, dir: dir}
	r.git("init", "-q", "-b", "main")
	r.git("config", "user.email", "test@example.com")
	r.git("config", "user.name", "Test")
	r.git("config", "commit.gpgsign", "false")
	return r
}

func (r *repo) git(args ...string) string {
	r.t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *repo) write(path, content string) {
	r.t.Helper()
	full := filepath.Join(r.dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		r.t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		r.t.Fatalf("write %s: %v", path, err)
	}
}

// commit stages everything and commits, returning the new SHA.
func (r *repo) commit(msg string) string {
	r.t.Helper()
	r.git("add", "-A")
	r.git("commit", "-q", "-m", msg)
	return r.git("rev-parse", "HEAD")
}

// initProjects builds app state via the same builder the CLI uses, so tests
// inherit its entrypoint-only filtering.
func initProjects(t *testing.T, cfg map[string]projects.ProjectConfig) {
	t.Helper()
	c := &projects.Config{Projects: cfg}
	p, err := projects.BuildProjects(c, nil)
	if err != nil {
		t.Fatalf("BuildProjects: %v", err)
	}
	app.Init(c, p)
}

// Entrypoints are reported; libraries only propagate to their dependents.
func goApp(path string, deps ...string) projects.ProjectConfig {
	return projects.ProjectConfig{
		Type:      projects.ProjectTypeGo,
		Path:      path,
		Build:     projects.BuildConfig{Entrypoint: true},
		DependsOn: deps,
	}
}

func nodeApp(path string, deps ...string) projects.ProjectConfig {
	return projects.ProjectConfig{
		Type:      projects.ProjectTypeNode,
		Path:      path,
		Build:     projects.BuildConfig{Entrypoint: true},
		DependsOn: deps,
	}
}

func goLib(path string, deps ...string) projects.ProjectConfig {
	return projects.ProjectConfig{
		Type:      projects.ProjectTypeGo,
		Path:      path,
		DependsOn: deps,
	}
}

// assertChanged compares the set exactly, so an over-eager result fails too.
func assertChanged(t *testing.T, changed map[string]bool, want ...string) {
	t.Helper()
	got := make([]string, 0, len(changed))
	for name, ok := range changed {
		if ok {
			got = append(got, name)
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Errorf("flagged projects = %v, want %v", got, want)
	}
}

// The base bounds the diff even though the tags sit further back: the api
// change landed on main before base, so it isn't the PR's (nor proxy's, via it).
func TestCompareRange_IgnoresCommitsBeforeBase(t *testing.T) {
	r := newRepo(t)

	r.write("apps/api/main.go", "package main\n")
	r.write("apps/web/index.js", "// web\n")
	r.write("apps/proxy/main.go", "package main\n")
	r.commit("feat: init")
	r.git("tag", "api/v0.1.0")
	r.git("tag", "web/v0.1.0")
	r.git("tag", "proxy/v0.1.0")

	r.write("apps/api/main.go", "package main // v2\n")
	base := r.commit("feat: api change already on main")

	r.write("apps/web/index.js", "// web v2\n")
	head := r.commit("feat: web change in the PR")

	initProjects(t, map[string]projects.ProjectConfig{
		"api":   goApp("apps/api"),
		"web":   nodeApp("apps/web"),
		"proxy": goApp("apps/proxy", "api"),
	})

	changed, err := compareRange(base, head)
	if err != nil {
		t.Fatalf("compareRange: %v", err)
	}
	assertChanged(t, changed, "web")
}

// A changed library fans out to dependent entrypoints, transitively, but is
// never itself reported.
func TestCompareRange_FlagsDependentsOfChangedLibrary(t *testing.T) {
	r := newRepo(t)

	r.write("packages/protos/api.proto", "// v1\n")
	r.write("apps/api/main.go", "package main\n")
	r.write("apps/web/index.js", "// web\n")
	r.write("apps/unrelated/main.go", "package main\n")
	base := r.commit("feat: init")

	r.write("packages/protos/api.proto", "// v2\n")
	head := r.commit("feat: proto change")

	initProjects(t, map[string]projects.ProjectConfig{
		"protos":    goLib("packages/protos"),
		"api":       goApp("apps/api", "protos"),
		"web":       nodeApp("apps/web", "api"),
		"unrelated": goApp("apps/unrelated"),
	})

	changed, err := compareRange(base, head)
	if err != nil {
		t.Fatalf("compareRange: %v", err)
	}
	assertChanged(t, changed, "api", "web")
}

// Empty head means the working tree: uncommitted and untracked files count,
// ignored ones don't. The branch is left behind base, which the anchor absorbs.
func TestCompareRange_WorkingTree(t *testing.T) {
	r := newRepo(t)

	r.write(".gitignore", "apps/web/dist/\n")
	r.write("apps/api/main.go", "package main\n")
	r.write("apps/web/index.js", "// web\n")
	r.write("apps/proxy/main.go", "package main\n")
	r.commit("feat: init")

	// main moves on after the feature branch diverges.
	r.git("checkout", "-q", "-b", "feature")
	r.git("checkout", "-q", "main")
	r.write("apps/proxy/main.go", "package main // main moved on\n")
	r.commit("feat: proxy change on main")

	r.git("checkout", "-q", "feature")
	r.write("apps/api/main.go", "package main // api v2\n")
	r.commit("feat: api change on branch")

	r.write("apps/web/index.js", "// edited, not committed\n")
	r.write("apps/web/newthing.ts", "// untracked\n")
	r.write("apps/web/dist/bundle.js", "// ignored build output\n")

	initProjects(t, map[string]projects.ProjectConfig{
		"api":   goApp("apps/api"),
		"web":   nodeApp("apps/web"),
		"proxy": goApp("apps/proxy"),
	})

	// Base is explicit: the fixture has no remote to resolve a default from.
	changed, err := compareRange("main", "")
	if err != nil {
		t.Fatalf("compareRange: %v", err)
	}
	assertChanged(t, changed, "api", "web")
}

// A gitignored file alone must never flag a project.
func TestCompareRange_WorkingTreeIgnoresGitignored(t *testing.T) {
	r := newRepo(t)

	r.write(".gitignore", "apps/web/dist/\n")
	r.write("apps/web/index.js", "// web\n")
	r.commit("feat: init")

	r.write("apps/web/dist/bundle.js", "// ignored build output\n")

	initProjects(t, map[string]projects.ProjectConfig{
		"web": nodeApp("apps/web"),
	})

	changed, err := compareRange("HEAD", "")
	if err != nil {
		t.Fatalf("compareRange: %v", err)
	}
	assertChanged(t, changed)
}

// --unreleased path: staggered tags give each project a different range, and a
// wider one can span another's changes. Only the range's owner may be flagged.
func TestGetChangedProjects_StaggeredTags(t *testing.T) {
	r := newRepo(t)

	r.write("apps/api/main.go", "package main\n")
	r.write("apps/web/index.js", "// web\n")
	r.write("apps/proxy/main.go", "package main\n")
	r.commit("feat: init")
	r.git("tag", "api/v0.1.0")
	r.git("tag", "proxy/v0.1.0")

	r.write("apps/web/index.js", "// web v2\n")
	r.commit("feat: web change")
	r.git("tag", "web/v0.2.0")

	r.write("apps/api/main.go", "package main // v2\n")
	head := r.commit("feat: api change")

	initProjects(t, map[string]projects.ProjectConfig{
		"api":   goApp("apps/api"),
		"web":   nodeApp("apps/web"),
		"proxy": goApp("apps/proxy", "api"),
	})

	changed, err := getChangedProjects(map[string]string{
		"api":   "v0.1.0",
		"web":   "v0.2.0",
		"proxy": "v0.1.0",
	}, head)
	if err != nil {
		t.Fatalf("getChangedProjects: %v", err)
	}
	assertChanged(t, changed, "api", "proxy")
}
