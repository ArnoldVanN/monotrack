package tag

import (
	"testing"

	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/versioning"
)

func TestParseReleaseCommit(t *testing.T) {
	tagFor := func(name, version string) string {
		return name + "/v" + version[1:] // strip leading v, re-add with prefix
	}

	t.Run("single project", func(t *testing.T) {
		msg := "chore(release): bump 1 project(s)\n\n- api v0.1.0 -> v0.2.0\n"
		entries, err := parseReleaseCommit(msg, "", tagFor)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].project != "api" {
			t.Errorf("project = %q, want %q", entries[0].project, "api")
		}
		if entries[0].oldVersion != "v0.1.0" {
			t.Errorf("oldVersion = %q, want %q", entries[0].oldVersion, "v0.1.0")
		}
		if entries[0].newVersion != "v0.2.0" {
			t.Errorf("newVersion = %q, want %q", entries[0].newVersion, "v0.2.0")
		}
		if entries[0].tag != "api/v0.2.0" {
			t.Errorf("tag = %q, want %q", entries[0].tag, "api/v0.2.0")
		}
	})

	t.Run("multiple projects", func(t *testing.T) {
		msg := "chore(release): bump 2 project(s)\n\n- api v0.1.0 -> v0.2.0\n- web v1.0.0 -> v1.1.0\n"
		entries, err := parseReleaseCommit(msg, "", tagFor)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
		if entries[1].project != "web" {
			t.Errorf("project = %q, want %q", entries[1].project, "web")
		}
	})

	t.Run("not a release commit", func(t *testing.T) {
		msg := "fix: some bug"
		_, err := parseReleaseCommit(msg, "", tagFor)
		if err == nil {
			t.Fatal("expected error for non-release commit")
		}
	})

	t.Run("release commit with empty body", func(t *testing.T) {
		msg := "chore(release): bump 1 project(s)"
		_, err := parseReleaseCommit(msg, "", tagFor)
		if err == nil {
			t.Fatal("expected error for empty body")
		}
	})

	t.Run("release commit with unparseable body", func(t *testing.T) {
		msg := "chore(release): bump 1 project(s)\n\nsome random text\n"
		_, err := parseReleaseCommit(msg, "", tagFor)
		if err == nil {
			t.Fatal("expected error for unparseable body")
		}
	})

	// A bullet that looks like an entry but doesn't parse must fail the whole
	// operation, not silently drop that project and undo the rest.
	t.Run("malformed bullet fails loudly", func(t *testing.T) {
		msg := "chore(release): bump 2 project(s)\n\n- api v0.1.0 -> v0.2.0\n- web v1.0.0 v1.1.0\n"
		_, err := parseReleaseCommit(msg, "", tagFor)
		if err == nil {
			t.Fatal("expected error for malformed bullet line, got nil (partial undo hazard)")
		}
	})
}

// TestParseReleaseCommit_RoundTrip ties the producer (buildReleaseMessage) to
// the consumer (parseReleaseCommit): if the release-commit format drifts on one
// side, this fails.
func TestParseReleaseCommit_RoundTrip(t *testing.T) {
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"charts/gateway": {},
		"worker":         {},
		"api":            {},
	}}
	results := []versioning.BumpResult{
		{Project: projects.NewHelmProject("charts/gateway", "charts/gateway", true, "helm"), OldVersion: "v0.0.1", NewVersion: "v0.0.2-rc.1"},
		{Project: projects.NewGoProject("worker", "apps/worker", true, "go"), OldVersion: "v0.2.2", NewVersion: "v0.2.3-rc.1"},
		{Project: projects.NewGoProject("api", "apps/api", true, "go"), OldVersion: "v1.0.0", NewVersion: "v1.1.0"},
	}

	msg := buildReleaseMessage(results, cfg.Name)
	entries, err := parseReleaseCommit(msg, cfg.Name, cfg.TagFor)
	if err != nil {
		t.Fatalf("round-trip parse failed:\n%s\nerror: %v", msg, err)
	}
	if len(entries) != len(results) {
		t.Fatalf("parsed %d entries, want %d\nmessage:\n%s", len(entries), len(results), msg)
	}

	got := make(map[string]releaseEntry, len(entries))
	for _, e := range entries {
		got[e.project] = e
	}
	for _, r := range results {
		e, ok := got[r.Project.Name()]
		if !ok {
			t.Errorf("project %q missing from parsed entries", r.Project.Name())
			continue
		}
		if e.oldVersion != r.OldVersion || e.newVersion != r.NewVersion {
			t.Errorf("project %q: parsed %s -> %s, want %s -> %s",
				r.Project.Name(), e.oldVersion, e.newVersion, r.OldVersion, r.NewVersion)
		}
		if want := cfg.TagFor(r.Project.Name(), r.NewVersion); e.tag != want {
			t.Errorf("project %q: tag = %q, want %q", r.Project.Name(), e.tag, want)
		}
	}
}

func TestParseReleaseCommit_ConfigName(t *testing.T) {
	tagFor := func(name, version string) string {
		return name + "/" + version
	}

	t.Run("matching name parses", func(t *testing.T) {
		msg := "chore(release): bump 1 project(s) [charts]\n\n- charts/api v0.1.0 -> v0.2.0\n"
		entries, err := parseReleaseCommit(msg, "charts", tagFor)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].project != "charts/api" {
			t.Fatalf("unexpected entries: %+v", entries)
		}
	})

	t.Run("other config's commit is refused", func(t *testing.T) {
		msg := "chore(release): bump 1 project(s) [charts]\n\n- charts/api v0.1.0 -> v0.2.0\n"
		if _, err := parseReleaseCommit(msg, "apps", tagFor); err == nil {
			t.Fatal("expected error when undoing another config's release commit")
		}
	})

	t.Run("unnamed commit is refused by a named config", func(t *testing.T) {
		msg := "chore(release): bump 1 project(s)\n\n- api v0.1.0 -> v0.2.0\n"
		if _, err := parseReleaseCommit(msg, "apps", tagFor); err == nil {
			t.Fatal("expected error for unlabeled release commit")
		}
	})

	t.Run("named commit is refused by an unnamed config", func(t *testing.T) {
		msg := "chore(release): bump 1 project(s) [charts]\n\n- charts/api v0.1.0 -> v0.2.0\n"
		if _, err := parseReleaseCommit(msg, "", tagFor); err == nil {
			t.Fatal("expected error for labeled release commit under unnamed config")
		}
	})
}

func TestReleaseSubject(t *testing.T) {
	if got, want := releaseSubject(3, ""), "chore(release): bump 3 project(s)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := releaseSubject(3, "charts"), "chore(release): bump 3 project(s) [charts]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The label must survive the producer/consumer round trip the same way the
// project bullets do.
func TestParseReleaseCommit_RoundTripNamed(t *testing.T) {
	cfg := &projects.Config{Name: "charts", Projects: map[string]projects.ProjectConfig{"charts/api": {}}}
	results := []versioning.BumpResult{
		{Project: projects.NewHelmProject("charts/api", "charts/api", true, "helm"), OldVersion: "v0.0.1", NewVersion: "v0.0.2"},
	}

	msg := buildReleaseMessage(results, cfg.Name)
	entries, err := parseReleaseCommit(msg, cfg.Name, cfg.TagFor)
	if err != nil {
		t.Fatalf("round-trip parse failed:\n%s\nerror: %v", msg, err)
	}
	if len(entries) != 1 {
		t.Fatalf("parsed %d entries, want 1\nmessage:\n%s", len(entries), msg)
	}
}
