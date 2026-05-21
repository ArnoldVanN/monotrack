package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/arnoldvann/monotrack/internal/projects"
)

func TestFilterTagsForProjects(t *testing.T) {
	api := projects.NewGoProject("api", "apps/api", true, "go")
	web := projects.NewNodeProject("web", "apps/web", true, "node")
	shared := projects.NewGoProject("shared", "packages/shared", false, "go")

	projMap := map[string]projects.Project{
		"api":    api,
		"web":    web,
		"shared": shared,
	}
	tags := []string{
		"api/v0.1.0",
		"api/v0.2.0",
		"web/v1.0.0",
		"unrelated",
		"apiv1.0.0", // missing slash, must not match api
		"shared/v0.0.1",
	}

	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api": {}, "web": {}, "shared": {},
	}}
	got := filterTagsForProjects(cfg, tags, projMap)

	want := map[string][]string{
		"api":    {"api/v0.1.0", "api/v0.2.0"},
		"web":    {"web/v1.0.0"},
		"shared": {"shared/v0.0.1"},
	}
	if len(got) != len(want) {
		t.Fatalf("filterTagsForProjects: got %d entries, want %d (%v)", len(got), len(want), got)
	}
	for p, tags := range got {
		w := want[p.Name()]
		sort.Strings(tags)
		sort.Strings(w)
		if !reflect.DeepEqual(tags, w) {
			t.Errorf("project %q: got %v, want %v", p.Name(), tags, w)
		}
	}
}

func TestFilterTagsForProjects_CustomScheme(t *testing.T) {
	api := projects.NewGoProject("api", "apps/api", true, "go")
	web := projects.NewNodeProject("web", "apps/web", true, "node")
	projMap := map[string]projects.Project{"api": api, "web": web}

	empty := ""
	cfg := &projects.Config{
		Projects: map[string]projects.ProjectConfig{"api": {}, "web": {}},
		Tags:     projects.TagsConfig{Separator: "@", VersionPrefix: &empty},
	}
	tags := []string{
		"api@1.0.0",
		"api@1.1.0",
		"web@2.0.0",
		"api/v1.0.0", // old scheme, must not match under new one
		"random",
	}
	got := filterTagsForProjects(cfg, tags, projMap)
	sort.Strings(got[api])
	if !reflect.DeepEqual(got[api], []string{"api@1.0.0", "api@1.1.0"}) {
		t.Errorf("api = %v", got[api])
	}
	if !reflect.DeepEqual(got[web], []string{"web@2.0.0"}) {
		t.Errorf("web = %v", got[web])
	}
}

func TestFilterTagsForProjects_SingleProject(t *testing.T) {
	only := projects.NewGoProject("only", ".", true, "go")
	projMap := map[string]projects.Project{"only": only}
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{"only": {}}}
	tags := []string{"v1.0.0", "v1.1.0", "other/v1.0.0", "1.2.3" /* missing v prefix */}
	got := filterTagsForProjects(cfg, tags, projMap)
	sort.Strings(got[only])
	if !reflect.DeepEqual(got[only], []string{"v1.0.0", "v1.1.0"}) {
		t.Errorf("single project tags = %v", got[only])
	}
}

func TestIsNonFastForward(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want bool
	}{
		{"non-fast-forward phrase", "Updates were rejected because the tip of your current branch is behind\nhint: non-fast-forward\n", true},
		{"fetch first phrase", "! [rejected] main -> main (fetch first)\n", true},
		{"rejected marker", " ! [rejected]        refs/tags/v1 -> refs/tags/v1\n", true},
		{"unrelated error", "fatal: unable to access 'https://...': could not resolve host\n", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNonFastForward([]byte(tc.out)); got != tc.want {
				t.Errorf("isNonFastForward(%q) = %v, want %v", tc.out, got, tc.want)
			}
		})
	}
}

// initRepo creates a fresh git repo in a temp dir, chdirs into it for the
// duration of the test, and returns the repo path.
func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}
	dir := t.TempDir()
	t.Chdir(dir)

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	run("config", "tag.gpgsign", "false")
	return dir
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func gitDo(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit stages `path` with the given contents and creates a commit, returning
// the resulting HEAD sha.
func commit(t *testing.T, path, contents, message string) string {
	t.Helper()
	writeFile(t, path, contents)
	gitDo(t, "add", path)
	gitDo(t, "commit", "-q", "-m", message)
	return gitDo(t, "rev-parse", "HEAD")
}

func TestGetRepoRootAndHead(t *testing.T) {
	dir := initRepo(t)
	sha := commit(t, "a.txt", "hello", "init")

	root, err := GetRepoRoot()
	if err != nil {
		t.Fatalf("GetRepoRoot: %v", err)
	}
	// Resolve symlinks on macOS/Linux temp dirs so comparison is stable.
	wantRoot, _ := filepath.EvalSymlinks(dir)
	gotRoot, _ := filepath.EvalSymlinks(root)
	if gotRoot != wantRoot {
		t.Errorf("GetRepoRoot = %q, want %q", gotRoot, wantRoot)
	}

	head, err := GetHead()
	if err != nil {
		t.Fatalf("GetHead: %v", err)
	}
	if head != sha {
		t.Errorf("GetHead = %q, want %q", head, sha)
	}
}

func TestIsShallowRepo(t *testing.T) {
	initRepo(t)
	commit(t, "a.txt", "x", "init")

	shallow, err := IsShallowRepo()
	if err != nil {
		t.Fatalf("IsShallowRepo: %v", err)
	}
	if shallow {
		t.Errorf("fresh repo reported shallow")
	}
}

func TestCurrentBranchAndDetached(t *testing.T) {
	initRepo(t)
	sha := commit(t, "a.txt", "x", "init")

	branch, err := CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "main" {
		t.Errorf("CurrentBranch = %q, want %q", branch, "main")
	}

	// Detach HEAD and confirm we get "".
	gitDo(t, "checkout", "-q", "--detach", sha)
	branch, err = CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch detached: %v", err)
	}
	if branch != "" {
		t.Errorf("CurrentBranch detached = %q, want empty", branch)
	}
}

func TestCreateAndDeleteTag(t *testing.T) {
	initRepo(t)
	sha := commit(t, "a.txt", "x", "init")

	if err := CreateTag("api/v0.1.0", "release api v0.1.0", sha); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	tags := gitDo(t, "tag", "--list")
	if !strings.Contains(tags, "api/v0.1.0") {
		t.Errorf("tag not created; got %q", tags)
	}

	if err := DeleteLocalTag("api/v0.1.0"); err != nil {
		t.Fatalf("DeleteLocalTag: %v", err)
	}
	tags = gitDo(t, "tag", "--list")
	if strings.Contains(tags, "api/v0.1.0") {
		t.Errorf("tag not deleted; got %q", tags)
	}
}

func TestGetTagsForProjects(t *testing.T) {
	initRepo(t)
	commit(t, "a.txt", "x", "init")

	for _, ref := range []string{"api/v0.1.0", "api/v0.2.0", "web/v1.0.0", "unrelated"} {
		if err := CreateTag(ref, ref, "HEAD"); err != nil {
			t.Fatalf("CreateTag %s: %v", ref, err)
		}
	}

	api := projects.NewGoProject("api", "apps/api", true, "go")
	web := projects.NewNodeProject("web", "apps/web", true, "node")
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{"api": {}, "web": {}}}
	got, err := GetTagsForProjects(cfg, map[string]projects.Project{"api": api, "web": web})
	if err != nil {
		t.Fatalf("GetTagsForProjects: %v", err)
	}

	sort.Strings(got[api])
	if !reflect.DeepEqual(got[api], []string{"api/v0.1.0", "api/v0.2.0"}) {
		t.Errorf("api tags = %v", got[api])
	}
	if !reflect.DeepEqual(got[web], []string{"web/v1.0.0"}) {
		t.Errorf("web tags = %v", got[web])
	}
}

func TestGetBase(t *testing.T) {
	initRepo(t)
	sha := commit(t, "a.txt", "x", "init")
	if err := CreateTag("api/v0.1.0", "msg", sha); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	got, err := GetBase("api/v0.1.0")
	if err != nil {
		t.Fatalf("GetBase: %v", err)
	}
	if got != sha {
		t.Errorf("GetBase = %q, want %q", got, sha)
	}
}

func TestGitDiff(t *testing.T) {
	initRepo(t)
	base := commit(t, "a.txt", "1", "init")
	commit(t, "b.txt", "2", "add b")
	head := commit(t, "a.txt", "1-updated", "modify a")

	files, err := GitDiff(base, head)
	if err != nil {
		t.Fatalf("GitDiff: %v", err)
	}
	sort.Strings(files)
	if !reflect.DeepEqual(files, []string{"a.txt", "b.txt"}) {
		t.Errorf("GitDiff = %v", files)
	}

	// No-change diff returns nil, not an error.
	files, err = GitDiff(head, head)
	if err != nil {
		t.Fatalf("GitDiff equal: %v", err)
	}
	if files != nil {
		t.Errorf("GitDiff equal = %v, want nil", files)
	}
}

func TestAddAndCommitPaths(t *testing.T) {
	initRepo(t)
	commit(t, "a.txt", "x", "init")

	writeFile(t, "tracked.txt", "tracked content")
	writeFile(t, "other.txt", "should remain unstaged")

	if err := AddFiles([]string{"tracked.txt"}); err != nil {
		t.Fatalf("AddFiles: %v", err)
	}
	// CommitPaths takes paths it commits directly (without prior add).
	sha, err := CommitPaths("add tracked", []string{"tracked.txt"})
	if err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}
	if sha == "" {
		t.Errorf("CommitPaths returned empty sha")
	}

	// other.txt should still appear as untracked.
	status := gitDo(t, "status", "--porcelain")
	if !strings.Contains(status, "other.txt") {
		t.Errorf("expected other.txt to remain untracked, got status:\n%s", status)
	}

	// Empty paths is an error.
	if _, err := CommitPaths("nope", nil); err == nil {
		t.Error("CommitPaths with empty paths should error")
	}

	// AddFiles with empty paths is a no-op.
	if err := AddFiles(nil); err != nil {
		t.Errorf("AddFiles(nil) = %v, want nil", err)
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	initRepo(t)
	commit(t, "a.txt", "x", "init")

	dirty, err := HasUncommittedChanges()
	if err != nil {
		t.Fatalf("HasUncommittedChanges clean: %v", err)
	}
	if dirty {
		t.Error("clean tree reported dirty")
	}

	writeFile(t, "a.txt", "modified")
	dirty, err = HasUncommittedChanges()
	if err != nil {
		t.Fatalf("HasUncommittedChanges dirty: %v", err)
	}
	if !dirty {
		t.Error("modified tree reported clean")
	}
}

func TestCheckoutBranchFromAndCheckoutBranch(t *testing.T) {
	initRepo(t)
	commit(t, "a.txt", "x", "init")

	if err := CheckoutBranchFrom("feature", "main"); err != nil {
		t.Fatalf("CheckoutBranchFrom: %v", err)
	}
	if got := gitDo(t, "rev-parse", "--abbrev-ref", "HEAD"); got != "feature" {
		t.Errorf("on branch %q, want feature", got)
	}

	if err := CheckoutBranch("main"); err != nil {
		t.Fatalf("CheckoutBranch: %v", err)
	}
	if got := gitDo(t, "rev-parse", "--abbrev-ref", "HEAD"); got != "main" {
		t.Errorf("on branch %q, want main", got)
	}
}

func TestLogBetween(t *testing.T) {
	initRepo(t)
	base := commit(t, "a.txt", "1", "init")
	commit(t, "apps/api/main.go", "x", "feat(api): hello")
	commit(t, "apps/web/index.html", "x", "feat(web): page")
	head := commit(t, "apps/api/main.go", "y", "fix(api): bug\n\nbody line")

	// All commits in range.
	all, err := LogBetween(base, head, nil)
	if err != nil {
		t.Fatalf("LogBetween all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("LogBetween all = %d commits, want 3", len(all))
	}
	if !strings.HasPrefix(all[0].Message, "fix(api): bug") {
		t.Errorf("first commit message = %q", all[0].Message)
	}
	if !strings.Contains(all[0].Message, "body line") {
		t.Errorf("commit body not captured: %q", all[0].Message)
	}

	// Path-filtered: only api commits.
	apiOnly, err := LogBetween(base, head, []string{"apps/api"})
	if err != nil {
		t.Fatalf("LogBetween api: %v", err)
	}
	if len(apiOnly) != 2 {
		t.Errorf("api commits = %d, want 2", len(apiOnly))
	}
	for _, c := range apiOnly {
		if !strings.Contains(c.Message, "api") {
			t.Errorf("unexpected non-api commit in api filter: %q", c.Message)
		}
	}

	// Empty range returns nil without error.
	empty, err := LogBetween(head, head, nil)
	if err != nil {
		t.Fatalf("LogBetween empty: %v", err)
	}
	if empty != nil {
		t.Errorf("LogBetween empty = %v, want nil", empty)
	}
}

func TestRemoteURL(t *testing.T) {
	initRepo(t)
	commit(t, "a.txt", "x", "init")
	gitDo(t, "remote", "add", "origin", "https://example.com/foo/bar.git")

	got, err := RemoteURL("origin")
	if err != nil {
		t.Fatalf("RemoteURL: %v", err)
	}
	if got != "https://example.com/foo/bar.git" {
		t.Errorf("RemoteURL = %q", got)
	}

	if _, err := RemoteURL("nope"); err == nil {
		t.Error("RemoteURL for missing remote should error")
	}
}

func TestPushRefsAtomicEmpty(t *testing.T) {
	// No refs => no-op success even without a remote configured.
	if err := PushRefsAtomic(nil); err != nil {
		t.Errorf("PushRefsAtomic(nil) = %v, want nil", err)
	}
}

func TestPushRefsAtomicNonFastForward(t *testing.T) {
	// Create two repos: "remote" (bare-ish) and "local" that pushes to it.
	// Simulate divergence so the push is rejected as non-fast-forward.
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "--bare", "-q", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatalf("init bare: %v: %s", err, out)
	}

	// Seed remote with one commit by pushing from a scratch clone.
	seed := filepath.Join(root, "seed")
	mustRun := func(dir string, args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git -C %s %s: %v: %s", dir, strings.Join(args, " "), err, out)
		}
	}
	if err := os.Mkdir(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	mustRun(seed, "init", "-q", "-b", "main")
	mustRun(seed, "config", "user.email", "t@e.com")
	mustRun(seed, "config", "user.name", "t")
	mustRun(seed, "config", "commit.gpgsign", "false")
	writeFile(t, filepath.Join(seed, "x.txt"), "1")
	mustRun(seed, "add", "x.txt")
	mustRun(seed, "commit", "-q", "-m", "init")
	mustRun(seed, "remote", "add", "origin", remote)
	mustRun(seed, "push", "-q", "origin", "main")

	// Now advance remote past local by committing once more from seed.
	writeFile(t, filepath.Join(seed, "x.txt"), "2")
	mustRun(seed, "add", "x.txt")
	mustRun(seed, "commit", "-q", "-m", "advance")
	mustRun(seed, "push", "-q", "origin", "main")

	// Local repo: clone the older state by cloning then resetting back one commit.
	local := filepath.Join(root, "local")
	if out, err := exec.Command("git", "clone", "-q", remote, local).CombinedOutput(); err != nil {
		t.Fatalf("clone: %v: %s", err, out)
	}
	mustRun(local, "config", "user.email", "t@e.com")
	mustRun(local, "config", "user.name", "t")
	mustRun(local, "config", "commit.gpgsign", "false")
	mustRun(local, "reset", "--hard", "HEAD~1")
	// Make a divergent commit locally.
	writeFile(t, filepath.Join(local, "x.txt"), "diverged")
	mustRun(local, "add", "x.txt")
	mustRun(local, "commit", "-q", "-m", "diverge")

	t.Chdir(local)
	err := PushRefsAtomic([]string{"refs/heads/main:refs/heads/main"})
	if err == nil {
		t.Fatal("expected non-fast-forward error, got nil")
	}
	if !errors.Is(err, ErrNonFastForward) {
		t.Errorf("err = %v, want wraps ErrNonFastForward", err)
	}
}
