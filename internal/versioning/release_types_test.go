package versioning

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
)

func TestBumpProjectsReleaseTypes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}

	type commit struct{ path, msg string }
	tests := []struct {
		name         string
		commits      []commit
		releaseTypes []string
		override     *BumpKind
		want         []string
	}{
		{
			name:    "chore alone does not release",
			commits: []commit{{"apps/api/a.go", "chore: tidy"}},
		},
		{
			name:    "non-conventional commit does not release",
			commits: []commit{{"apps/api/a.go", "tidy things"}},
		},
		{
			name:    "fix releases",
			commits: []commit{{"apps/api/a.go", "chore: tidy"}, {"apps/api/b.go", "fix: bug"}},
			want:    []string{"api"},
		},
		{
			name:    "breaking chore releases",
			commits: []commit{{"apps/api/a.go", "chore!: drop flag"}},
			want:    []string{"api"},
		},
		{
			name:    "fix in another project does not release this one",
			commits: []commit{{"apps/api/a.go", "chore: tidy"}, {"apps/web/a.js", "fix: bug"}},
			want:    []string{"web"},
		},
		{
			name:    "fix in a dependency releases the dependent",
			commits: []commit{{"libs/lib/a.go", "fix: bug"}},
			want:    []string{"api", "lib"},
		},
		{
			name:    "chore in a dependency does not release the dependent",
			commits: []commit{{"libs/lib/a.go", "chore: tidy"}},
		},
		{
			name:         "custom release types",
			commits:      []commit{{"apps/api/a.go", "chore: tidy"}, {"apps/web/a.js", "chore: tidy"}},
			releaseTypes: []string{"chore"},
			want:         []string{"api", "web"},
		},
		{
			name:         "wildcard releases on any commit",
			commits:      []commit{{"apps/api/a.go", "tidy things"}},
			releaseTypes: []string{"*"},
			want:         []string{"api"},
		},
		{
			name:     "explicit bump kind releases any change",
			commits:  []commit{{"apps/api/a.go", "chore: tidy"}},
			override: func() *BumpKind { k := PatchBump; return &k }(),
			want:     []string{"api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)

			run := func(args ...string) {
				t.Helper()
				out, err := exec.Command("git", args...).CombinedOutput()
				if err != nil {
					t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
				}
			}
			write := func(path, content string) {
				t.Helper()
				full := filepath.Join(dir, path)
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
					t.Fatalf("write: %v", err)
				}
			}
			run("init", "-q", "-b", "main")
			run("config", "user.email", "test@example.com")
			run("config", "user.name", "Test")
			run("config", "commit.gpgsign", "false")
			run("config", "tag.gpgsign", "false")

			write("apps/api/main.go", "package main\n")
			write("apps/web/index.js", "// web\n")
			write("libs/lib/lib.go", "package lib\n")
			run("add", ".")
			run("commit", "-q", "-m", "feat: init")
			run("tag", "api/v1.0.0")
			run("tag", "web/v1.0.0")
			run("tag", "lib/v1.0.0")

			for _, c := range tt.commits {
				write(c.path, c.msg+"\n")
				run("add", ".")
				run("commit", "-q", "-m", c.msg)
			}

			cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
				"api": {Type: projects.ProjectTypeGo, Path: "apps/api", DependsOn: []string{"lib"}, ReleaseTypes: tt.releaseTypes},
				"web": {Type: projects.ProjectTypeNode, Path: "apps/web", ReleaseTypes: tt.releaseTypes},
				"lib": {Type: projects.ProjectTypeGo, Path: "libs/lib", ReleaseTypes: tt.releaseTypes},
			}}
			app.Init(cfg, map[string]projects.Project{
				"api": projects.NewGoProject("api", "apps/api", true, "go"),
				"web": projects.NewNodeProject("web", "apps/web", true, "node"),
				"lib": projects.NewGoProject("lib", "libs/lib", true, "go"),
			})

			head, err := git.GetHead()
			if err != nil {
				t.Fatalf("GetHead: %v", err)
			}
			b := NewBumper()
			results, err := b.BumpProjects(app.State.Projects, tt.override, false, head)
			if err != nil {
				t.Fatalf("BumpProjects: %v", err)
			}

			var got []string
			for _, r := range results {
				got = append(got, r.Project.Name())
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("released %v, want %v", got, tt.want)
			}
		})
	}
}
