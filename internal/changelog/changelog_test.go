package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/arnoldvann/monotrack/internal/versioning/conventional"
)

func TestRender_SectionsAndOrdering(t *testing.T) {
	e := Entry{
		NewVersion: "v1.2.0",
		Date:       time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
		Commits: []conventional.ParsedCommit{
			{Hash: "1111111", Type: conventional.TypeFeat, Description: "add endpoint", Scope: "api"},
			{Hash: "2222222", Type: conventional.TypeFix, Description: "handle nil"},
			{Hash: "3333333", Type: conventional.TypeFeat, Description: "drop v1", Breaking: true},
			{Hash: "4444444", Type: conventional.TypeUnknown, Subject: "WIP"},
		},
	}

	out := Render(e)

	if !strings.Contains(out, "## [v1.2.0] - 2026-05-09") {
		t.Errorf("missing version header: %s", out)
	}
	if !strings.Contains(out, "Breaking Changes") {
		t.Error("missing breaking changes section")
	}
	if !strings.Contains(out, "### Features") {
		t.Error("missing features section")
	}
	if !strings.Contains(out, "### Bug Fixes") {
		t.Error("missing bug fixes section")
	}
	if strings.Contains(out, "WIP") {
		t.Error("non-conventional commit should be excluded from changelog")
	}

	bIdx := strings.Index(out, "Breaking Changes")
	fIdx := strings.Index(out, "Features")
	if bIdx >= fIdx {
		t.Error("breaking changes should appear before features")
	}
}

func TestRender_DependencyOnly(t *testing.T) {
	e := Entry{
		NewVersion: "v0.0.2",
		Date:       time.Now(),
		Reason:     versioning.ReasonDependency,
		Commits:    nil,
	}
	out := Render(e)
	if !strings.Contains(out, "- Updated internal dependencies\n") {
		t.Errorf("expected generic dep-only fallback, got: %s", out)
	}
}

func TestRender_DependencyOnlyListsDeps(t *testing.T) {
	e := Entry{
		NewVersion: "v0.0.2",
		Date:       time.Now(),
		Reason:     versioning.ReasonDependency,
		Dependencies: []versioning.DepUpdate{
			{Name: "go-shared", Version: "v0.4.1"},
			{Name: "ui"},
		},
	}
	out := Render(e)
	want := "### Other\n\n- Updated internal dependencies:\n  - go-shared → v0.4.1\n  - ui\n\n"
	if !strings.HasSuffix(out, want) {
		t.Errorf("expected dep list %q, got: %s", want, out)
	}
}

func TestRender_PromotionFallback(t *testing.T) {
	e := Entry{
		OldVersion: "v1.2.0-rc.3",
		NewVersion: "v1.2.0",
		Date:       time.Now(),
		Reason:     versioning.ReasonPromotion,
		Commits:    nil,
	}
	out := Render(e)
	if strings.Contains(out, "Updated internal dependencies") {
		t.Errorf("promotion should not render dependency fallback: %s", out)
	}
	if !strings.Contains(out, "Promoted from v1.2.0-rc.3") {
		t.Errorf("expected promotion line referencing old version, got: %s", out)
	}
}

func TestRender_InitialFallback(t *testing.T) {
	e := Entry{
		NewVersion: "v0.1.0",
		Date:       time.Now(),
		Reason:     versioning.ReasonInitial,
		Commits:    nil,
	}
	out := Render(e)
	if !strings.Contains(out, "Initial release") {
		t.Errorf("expected initial-release fallback, got: %s", out)
	}
}

func TestRender_OrphanedFallback(t *testing.T) {
	e := Entry{
		NewVersion: "v1.1.0-rc.1",
		Date:       time.Now(),
		Reason:     versioning.ReasonOrphaned,
		Commits:    nil,
	}
	out := Render(e)
	if !strings.Contains(out, "Re-anchored after history change") {
		t.Errorf("expected orphaned fallback, got: %s", out)
	}
}

func TestRender_OnlyNonConventional(t *testing.T) {
	e := Entry{
		NewVersion: "v0.0.2",
		Date:       time.Now(),
		Commits: []conventional.ParsedCommit{
			{Hash: "abcdef1", Type: conventional.TypeUnknown, Subject: "fix changelogs"},
		},
	}
	out := Render(e)
	if strings.Contains(out, "Updated internal dependencies") {
		t.Errorf("should not use dep-only fallback when commits exist: %s", out)
	}
	if !strings.Contains(out, "fix changelogs") {
		t.Errorf("expected non-conventional subject in output: %s", out)
	}
	if !strings.Contains(out, "### Other") {
		t.Errorf("expected Other section: %s", out)
	}
}

func TestPrepend_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	if err := prepend(path, "## [v1.0.0] - 2026-05-09\n\n### Features\n\n- thing (abc1234)\n\n"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if !strings.HasPrefix(s, "# Changelog") {
		t.Errorf("missing header: %s", s)
	}
	if !strings.Contains(s, "## [v1.0.0]") {
		t.Errorf("missing release block: %s", s)
	}
}

func TestPrepend_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	original := "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n## [v1.0.0] - 2026-01-01\n\n- old\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := prepend(path, "## [v1.1.0] - 2026-05-09\n\n- new\n\n"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	v110 := strings.Index(s, "v1.1.0")
	v100 := strings.Index(s, "v1.0.0")
	if v110 < 0 || v100 < 0 || v110 > v100 {
		t.Errorf("new entry should appear before old: %s", s)
	}
}

func TestSplice(t *testing.T) {
	all := func(string) bool { return true }
	none := func(string) bool { return false }

	existing := fileHeader + "Some preamble.\n\n" +
		"## [v1.1.0] - 2026-05-02\n\n- stale\n\n" +
		"## [v1.0.0] - 2026-05-01\n\n- hand edited\n"
	sections := []Section{
		{Key: "v1.2.0", Body: "## [v1.2.0] - 2026-05-03\n\n- new\n\n", Date: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)},
		{Key: "v1.1.0", Body: "## [v1.1.0] - 2026-05-02\n\n- fresh\n\n", Date: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)},
	}

	want := fileHeader + "Some preamble.\n\n" +
		"## [v1.2.0] - 2026-05-03\n\n- new\n\n" +
		"## [v1.1.0] - 2026-05-02\n\n- fresh\n\n" +
		"## [v1.0.0] - 2026-05-01\n\n- hand edited\n"
	if got := Splice(existing, sections, all); got != want {
		t.Errorf("Splice keep:\n%s\nwant:\n%s", got, want)
	}

	want = fileHeader +
		"## [v1.2.0] - 2026-05-03\n\n- new\n\n" +
		"## [v1.1.0] - 2026-05-02\n\n- fresh\n"
	if got := Splice(existing, sections, none); got != want {
		t.Errorf("Splice replace:\n%s\nwant:\n%s", got, want)
	}
}

// Regenerated sections from the same day keep their release order, even
// across projects, since they carry the exact time.
func TestSplice_SameDayByTime(t *testing.T) {
	sections := []Section{
		{Key: "web v1.0.1", Body: "## [web v1.0.1] - 2026-05-03\n\n- web\n\n", Date: time.Date(2026, 5, 3, 15, 0, 0, 0, time.UTC)},
		{Key: "api v1.1.0", Body: "## [api v1.1.0] - 2026-05-03\n\n- api\n\n", Date: time.Date(2026, 5, 3, 9, 0, 0, 0, time.UTC)},
	}
	want := fileHeader +
		"## [web v1.0.1] - 2026-05-03\n\n- web\n\n" +
		"## [api v1.1.0] - 2026-05-03\n\n- api\n"
	if got := Splice("", sections, func(string) bool { return false }); got != want {
		t.Errorf("Splice:\n%s\nwant:\n%s", got, want)
	}
}

// A kept section newer than the regenerated ones, e.g. a prerelease left out
// by --skip-prereleases, lands in date order rather than below them.
func TestSplice_OrdersByDate(t *testing.T) {
	existing := fileHeader +
		"## [v1.1.0-rc.1] - 2026-05-05\n\n- rc\n\n" +
		"## [v1.0.0] - 2026-05-01\n\n- old\n"
	sections := []Section{
		{Key: "v1.0.1", Body: "## [v1.0.1] - 2026-05-03\n\n- patch\n\n", Date: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)},
		{Key: "v1.0.2", Body: "## [v1.0.2] - 2026-05-03\n\n- same day\n\n", Date: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)},
	}
	want := fileHeader +
		"## [v1.1.0-rc.1] - 2026-05-05\n\n- rc\n\n" +
		"## [v1.0.2] - 2026-05-03\n\n- same day\n\n" +
		"## [v1.0.1] - 2026-05-03\n\n- patch\n\n" +
		"## [v1.0.0] - 2026-05-01\n\n- old\n"
	if got := Splice(existing, sections, func(string) bool { return true }); got != want {
		t.Errorf("Splice:\n%s\nwant:\n%s", got, want)
	}
}

// Regenerating one project of a combined changelog keeps the others.
func TestSplice_KeepsOtherProjects(t *testing.T) {
	existing := fileHeader +
		"## [web v1.1.0] - 2026-05-04\n\n- web\n\n" +
		"## [api v2.0.0] - 2026-05-02\n\n- api stale\n\n" +
		"## [api v1.9.0] - 2026-05-01\n\n- api removed tag\n"
	sections := []Section{
		{Key: "api v2.0.0", Body: "## [api v2.0.0] - 2026-05-02\n\n- api fresh\n\n", Date: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)},
	}
	keep := func(key string) bool { return SectionProject(key) != "api" }
	want := fileHeader +
		"## [web v1.1.0] - 2026-05-04\n\n- web\n\n" +
		"## [api v2.0.0] - 2026-05-02\n\n- api fresh\n"
	if got := Splice(existing, sections, keep); got != want {
		t.Errorf("Splice:\n%s\nwant:\n%s", got, want)
	}
}

func TestSections_CombinedKeys(t *testing.T) {
	e := Entry{
		Project:    projects.NewNodeProject("web", "apps/web", true, "node"),
		NewVersion: "v1.0.0",
		Date:       time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
		Reason:     versioning.ReasonInitial,
	}
	s := Sections([]Entry{e}, true)[0]
	if s.Key != "web v1.0.0" || !strings.HasPrefix(s.Body, "## [web v1.0.0] - 2026-05-09\n") {
		t.Errorf("combined section = %+v", s)
	}
	if key, ok := sectionKey(s.Body); !ok || key != s.Key {
		t.Errorf("sectionKey = %q, %v; want %q", key, ok, s.Key)
	}
}
