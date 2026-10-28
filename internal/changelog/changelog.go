package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/versioning"
	"github.com/arnoldvann/monotrack/internal/versioning/conventional"
	"golang.org/x/mod/semver"
)

// TODO: customizable header
const fileHeader = `# Changelog

`

type Entry struct {
	Project      projects.Project
	OldVersion   string
	NewVersion   string
	Date         time.Time
	Reason       versioning.BumpReason
	Commits      []conventional.ParsedCommit
	Dependencies []versioning.DepUpdate
}

type section struct {
	title string
	types []conventional.CommitType
}

var sections = []section{
	{title: "Features", types: []conventional.CommitType{conventional.TypeFeat}},
	{title: "Bug Fixes", types: []conventional.CommitType{conventional.TypeFix}},
	{title: "Performance", types: []conventional.CommitType{conventional.TypePerf}},
	{title: "Other", types: []conventional.CommitType{
		conventional.TypeRefactor,
		conventional.TypeChore,
		conventional.TypeDocs,
		conventional.TypeStyle,
		conventional.TypeTest,
		conventional.TypeBuild,
		conventional.TypeCI,
		conventional.TypeRevert,
	}},
}

// Render returns a markdown block describing this release for one project.
// Non-conventional commits are excluded. Breaking changes get their own section
// regardless of underlying type. If a project has no conventional commits at
// all, an "Other" section is appended whose content depends on Entry.Reason
// (dependency bump, promotion, or initial release).
func Render(e Entry) string {
	var b strings.Builder

	dateStr := e.Date.Format("2006-01-02")
	fmt.Fprintf(&b, "## [%s] - %s\n\n", e.NewVersion, dateStr)

	var breaking []conventional.ParsedCommit
	byType := make(map[conventional.CommitType][]conventional.ParsedCommit)
	conventionalCount := 0

	for _, c := range e.Commits {
		if c.Type == conventional.TypeUnknown {
			continue
		}
		conventionalCount++
		if c.Breaking {
			breaking = append(breaking, c)
			continue
		}
		byType[c.Type] = append(byType[c.Type], c)
	}

	if len(breaking) > 0 {
		b.WriteString("### ⚠ Breaking Changes\n\n")
		for _, c := range breaking {
			b.WriteString(renderLine(c))
		}
		b.WriteString("\n")
	}

	for _, s := range sections {
		var lines []conventional.ParsedCommit
		for _, t := range s.types {
			lines = append(lines, byType[t]...)
		}
		if len(lines) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n", s.title)
		for _, c := range lines {
			b.WriteString(renderLine(c))
		}
		b.WriteString("\n")
	}

	if conventionalCount == 0 {
		b.WriteString(renderFallback(e))
	}

	return b.String()
}

// renderFallback emits the trailing section when no conventional commits were
// rendered. With commits present it lists the non-conventional ones; with none,
// it uses the bump's Reason to pick an accurate single-line summary.
func renderFallback(e Entry) string {
	if len(e.Commits) > 0 {
		var b strings.Builder
		b.WriteString("### Other\n\n")
		for _, c := range e.Commits {
			b.WriteString(renderRawLine(c))
		}
		b.WriteString("\n")
		return b.String()
	}

	var line string
	switch e.Reason {
	case versioning.ReasonPromotion:
		if e.OldVersion != "" {
			line = fmt.Sprintf("Promoted from %s", e.OldVersion)
		} else {
			line = "Promoted to stable"
		}
	case versioning.ReasonInitial:
		line = "Initial release"
	case versioning.ReasonOrphaned:
		line = "Re-anchored after history change"
	default:
		line = "Updated internal dependencies"
		if len(e.Dependencies) > 0 {
			var b strings.Builder
			b.WriteString(line + ":")
			for _, d := range e.Dependencies {
				if d.Version != "" {
					fmt.Fprintf(&b, "\n  - %s → %s", d.Name, d.Version)
				} else {
					fmt.Fprintf(&b, "\n  - %s", d.Name)
				}
			}
			line = b.String()
		}
	}
	return fmt.Sprintf("### Other\n\n- %s\n\n", line)
}

func renderRawLine(c conventional.ParsedCommit) string {
	subject := c.Subject
	if subject == "" {
		subject = c.Description
	}
	return fmt.Sprintf("- %s (%s)\n", subject, c.ShortHash())
}

func renderLine(c conventional.ParsedCommit) string {
	desc := c.Description
	if desc == "" {
		desc = c.Subject
	}
	if c.Scope != "" {
		return fmt.Sprintf("- **%s:** %s (%s)\n", c.Scope, desc, c.ShortHash())
	}
	return fmt.Sprintf("- %s (%s)\n", desc, c.ShortHash())
}

// PerProject writes/prepends a changelog block to each project's CHANGELOG.md
// at <project.Path>/CHANGELOG.md.
func PerProject(entries []Entry) error {
	for _, e := range entries {
		path := filepath.Join(e.Project.Path(), "CHANGELOG.md")
		if err := prepend(path, Render(e)); err != nil {
			return err
		}
	}
	return nil
}

// Combined writes a single CHANGELOG.md at the given path. Each entry is
// rendered with the project name in the heading so users can scan for the
// project they care about.
func Combined(path string, entries []Entry) error {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Project.Name() < sorted[j].Project.Name()
	})

	var block strings.Builder
	for _, e := range sorted {
		block.WriteString(renderCombined(e))
	}

	return prepend(path, block.String())
}

// renderCombined renders e with the project name in its heading.
func renderCombined(e Entry) string {
	dateStr := e.Date.Format("2006-01-02")
	// strip the per-entry "## [version] - date" header that Render added
	return fmt.Sprintf("## [%s %s] - %s\n\n", e.Project.Name(), e.NewVersion, dateStr) + stripFirstHeading(Render(e))
}

// Section is one rendered release. Key is the bracketed text of its heading,
// which identifies the release within a changelog file. Date is the exact
// release time, which orders releases from the same day.
type Section struct {
	Key  string
	Body string
	Date time.Time
}

// Sections renders entries in the given order, with the project name in each
// heading when combined.
func Sections(entries []Entry, combined bool) []Section {
	out := make([]Section, 0, len(entries))
	for _, e := range entries {
		if combined {
			out = append(out, Section{Key: e.Project.Name() + " " + e.NewVersion, Body: renderCombined(e), Date: e.Date})
		} else {
			out = append(out, Section{Key: e.NewVersion, Body: Render(e), Date: e.Date})
		}
	}
	return out
}

// Splice merges sections into the existing changelog. A section of existing
// whose key is among sections is replaced; any other is kept when keep
// returns true for its key. keep is called with "" for text that isn't a
// release section, such as a preamble. Undated text stays on top in its
// original order, followed by every release, newest first.
func Splice(existing string, sections []Section, keep func(key string) bool) string {
	replaced := make(map[string]bool, len(sections))
	for _, s := range sections {
		replaced[s.Key] = true
	}

	var undated []string
	dated := slices.Clone(sections)
	rest, _ := strings.CutPrefix(existing, fileHeader)
	for _, chunk := range splitSections(rest) {
		key, _ := sectionKey(chunk)
		if (key != "" && replaced[key]) || !keep(key) {
			continue
		}
		// Only the day is in the file, so a kept section sorts below a
		// regenerated one from the same day.
		if date, ok := sectionDate(chunk); ok {
			dated = append(dated, Section{Key: key, Body: chunk, Date: date})
		} else {
			undated = append(undated, chunk)
		}
	}
	slices.SortStableFunc(dated, compareSections)

	var b strings.Builder
	b.WriteString(fileHeader)
	for _, chunk := range slices.Concat(undated, bodies(dated)) {
		// The file's last section may lack the blank line that separates
		// sections, and needn't stay last.
		b.WriteString(strings.TrimRight(chunk, "\n") + "\n\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// compareSections orders release sections newest first. Sections released
// at the same time are ordered by version, highest first, then by project.
func compareSections(a, b Section) int {
	if c := b.Date.Compare(a.Date); c != 0 {
		return c
	}
	na, va := splitKey(a.Key)
	nb, vb := splitKey(b.Key)
	if c := semver.Compare(vb, va); c != 0 {
		return c
	}
	return strings.Compare(na, nb)
}

func bodies(sections []Section) []string {
	out := make([]string, len(sections))
	for i, s := range sections {
		out[i] = s.Body
	}
	return out
}

// SectionProject returns the project name in a combined section key
// ("name version"), or "" for a per-project key, which is just the version.
func SectionProject(key string) string {
	name, _ := splitKey(key)
	return name
}

// splitKey splits a section key into its project name, empty for a
// per-project key, and its version, normalized to a "v" prefix for semver.
func splitKey(key string) (name, version string) {
	if i := strings.LastIndexByte(key, ' '); i >= 0 {
		name, version = key[:i], key[i+1:]
	} else {
		version = key
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return name, version
}

// sectionDate returns the date of a "## [key] - 2006-01-02" heading.
func sectionDate(chunk string) (time.Time, bool) {
	if _, ok := sectionKey(chunk); !ok {
		return time.Time{}, false
	}
	heading, _, _ := strings.Cut(chunk, "\n")
	_, date, ok := strings.Cut(heading, "] - ")
	if !ok {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(date))
	return t, err == nil
}

// splitSections splits s before every "## " heading line.
func splitSections(s string) []string {
	var chunks []string
	start := 0
	for i := 0; i < len(s); {
		end := strings.IndexByte(s[i:], '\n')
		if end < 0 {
			end = len(s)
		} else {
			end += i + 1
		}
		if i > start && strings.HasPrefix(s[i:], "## ") {
			chunks = append(chunks, s[start:i])
			start = i
		}
		i = end
	}
	if start < len(s) {
		chunks = append(chunks, s[start:])
	}
	return chunks
}

// sectionKey returns the bracketed text of a "## [key] - date" heading.
func sectionKey(chunk string) (string, bool) {
	rest, ok := strings.CutPrefix(chunk, "## [")
	if !ok {
		return "", false
	}
	key, _, ok := strings.Cut(rest, "]")
	return key, ok
}

func stripFirstHeading(s string) string {
	_, rest, ok := strings.Cut(s, "\n")
	if !ok {
		return ""
	}
	return strings.TrimLeft(rest, "\n")
}

func prepend(path, block string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var content string
	if len(existing) == 0 {
		content = fileHeader + block
	} else {
		body := string(existing)
		if rest, ok := strings.CutPrefix(body, fileHeader); ok {
			content = fileHeader + block + rest
		} else {
			content = fileHeader + block + body
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// PRBody renders a release PR body covering multiple projects. Breaking
// changes from all projects are collected into a single top-level section
// (with per-project subsections); each project then gets its own section
// containing the remaining changes.
func PRBody(entries []Entry) string {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Project.Name() < sorted[j].Project.Name()
	})

	var b strings.Builder

	type projBreaking struct {
		name    string
		commits []conventional.ParsedCommit
	}
	var allBreaking []projBreaking
	for _, e := range sorted {
		var breaking []conventional.ParsedCommit
		for _, c := range e.Commits {
			if c.Type == conventional.TypeUnknown {
				continue
			}
			if c.Breaking {
				breaking = append(breaking, c)
			}
		}
		if len(breaking) > 0 {
			allBreaking = append(allBreaking, projBreaking{e.Project.Name(), breaking})
		}
	}

	if len(allBreaking) > 0 {
		b.WriteString("## ⚠ BREAKING CHANGES\n\n")
		for _, pb := range allBreaking {
			fmt.Fprintf(&b, "### %s\n\n", pb.name)
			for _, c := range pb.commits {
				b.WriteString(renderLine(c))
			}
			b.WriteString("\n")
		}
	}

	for _, e := range sorted {
		dateStr := e.Date.Format("2006-01-02")
		fmt.Fprintf(&b, "## %s %s (%s)\n\n", e.Project.Name(), e.NewVersion, dateStr)

		byType := make(map[conventional.CommitType][]conventional.ParsedCommit)
		conventionalCount := 0
		for _, c := range e.Commits {
			if c.Type == conventional.TypeUnknown {
				continue
			}
			conventionalCount++
			if c.Breaking {
				continue
			}
			byType[c.Type] = append(byType[c.Type], c)
		}

		nonBreakingRendered := false
		for _, s := range sections {
			var lines []conventional.ParsedCommit
			for _, t := range s.types {
				lines = append(lines, byType[t]...)
			}
			if len(lines) == 0 {
				continue
			}
			nonBreakingRendered = true
			fmt.Fprintf(&b, "### %s\n\n", s.title)
			for _, c := range lines {
				b.WriteString(renderLine(c))
			}
			b.WriteString("\n")
		}

		if conventionalCount == 0 {
			b.WriteString(renderFallback(e))
		} else if !nonBreakingRendered {
			// only breaking commits — point readers to the section above
			b.WriteString("_See breaking changes above._\n\n")
		}
	}

	return b.String()
}

// RenderForPrint returns a stdout-friendly representation for dry runs.
func RenderForPrint(entries []Entry) string {
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "--- changelog for %s ---\n", e.Project.Name())
		b.WriteString(Render(e))
	}
	return b.String()
}
