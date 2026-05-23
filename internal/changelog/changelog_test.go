package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if !strings.Contains(out, "Updated internal dependencies") {
		t.Errorf("expected dep-only fallback, got: %s", out)
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
