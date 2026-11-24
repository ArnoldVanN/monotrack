package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	// Tag and PrevTag are this release's tag and the one it follows, which
	// link the heading to the diff between them. PrevTag is empty for a
	// release without a comparable predecessor.
	Tag     string
	PrevTag string
	Links   Links
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
	return heading(e, false) + renderBody(e)
}

// heading renders "## v1.2.0 - 2006-01-02", with the project name before the
// version when withName is set, and the title linked to the diff from the
// previous release when there's a link for it.
func heading(e Entry, withName bool) string {
	title := e.NewVersion
	if withName {
		title = e.Project.Name() + " " + title
	}
	if url := e.Links.compare(e.PrevTag, e.Tag); url != "" {
		title = "[" + title + "](" + url + ")"
	}
	return fmt.Sprintf("## %s - %s\n\n", title, e.Date.Format("2006-01-02"))
}

func renderBody(e Entry) string {
	var b strings.Builder

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
			b.WriteString(renderBreakingLine(e.Links, c))
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
			b.WriteString(renderLine(e.Links, c))
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
			b.WriteString(renderRawLine(e.Links, c))
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

func renderRawLine(l Links, c conventional.ParsedCommit) string {
	subject := c.Subject
	if subject == "" {
		subject = c.Description
	}
	return fmt.Sprintf("- %s (%s)\n", l.prRefs(subject), l.commit(c.Hash, c.ShortHash()))
}

func renderLine(l Links, c conventional.ParsedCommit) string {
	desc := c.Description
	if desc == "" {
		desc = c.Subject
	}
	desc = l.prRefs(desc)
	hash := l.commit(c.Hash, c.ShortHash())
	if c.Scope != "" {
		return fmt.Sprintf("- **%s:** %s (%s)\n", c.Scope, desc, hash)
	}
	return fmt.Sprintf("- %s (%s)\n", desc, hash)
}

// renderBreakingLine is renderLine with the BREAKING CHANGE footer's text
// quoted under it, inside the same list item.
func renderBreakingLine(l Links, c conventional.ParsedCommit) string {
	if c.BreakingNote == "" {
		return renderLine(l, c)
	}
	return renderLine(l, c) + "  > " + c.BreakingNote + "\n"
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

// Combined writes a single CHANGELOG.md at the given path. With withName,
// each entry is rendered with the project name in the heading so users can
// scan for the project they care about.
func Combined(path string, entries []Entry, withName bool) error {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Project.Name() < sorted[j].Project.Name()
	})

	var block strings.Builder
	for _, e := range sorted {
		block.WriteString(heading(e, withName) + renderBody(e))
	}

	return prepend(path, block.String())
}

// Section is one rendered release. Key is the text of its heading without
// the date, "name version" or just the version, which identifies the release
// within a changelog file. Date is the exact release time, which orders
// releases from the same day.
type Section struct {
	Key  string
	Body string
	Date time.Time
}

// Sections renders entries in the given order, with the project name in each
// heading when withName is set.
func Sections(entries []Entry, withName bool) []Section {
	out := make([]Section, 0, len(entries))
	for _, e := range entries {
		key := e.NewVersion
		if withName {
			key = e.Project.Name() + " " + key
		}
		out = append(out, Section{Key: key, Body: heading(e, withName) + renderBody(e), Date: e.Date})
	}
	return out
}

// Splice merges sections into the existing changelog. A section of existing
// whose key is among sections is replaced; any other is kept when keep
// returns true for its key. keep is called with "" for text that isn't a
// release section, such as a preamble. Undated text stays on top in its
// original order, followed by every release, newest first.
func Splice(existing string, sections []Section, keep func(key string) bool) string {
	replaced := func(key string) bool {
		return slices.ContainsFunc(sections, func(s Section) bool { return sameRelease(s.Key, key) })
	}

	var undated []string
	dated := slices.Clone(sections)
	rest, _ := strings.CutPrefix(existing, fileHeader)
	for _, chunk := range splitSections(rest) {
		key, date, hasDate := parseHeading(chunk)
		if (key != "" && replaced(key)) || !keep(key) {
			continue
		}
		// Only the day is in the file, so a kept section sorts below a
		// regenerated one from the same day.
		if hasDate {
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

// sameRelease reports whether two section keys name the same release. A key
// without a project name matches any project's, so headings written before a
// changelog dropped or gained project names are still replaced.
func sameRelease(a, b string) bool {
	na, va := splitKey(a)
	nb, vb := splitKey(b)
	return va == vb && (na == nb || na == "" || nb == "")
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

var (
	headingLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	headingDateRe = regexp.MustCompile(`^(.*?)\s+(?:-\s+(\d{4}-\d{2}-\d{2})|\((\d{4}-\d{2}-\d{2})\))\s*$`)
)

// parseHeading returns the key and date of a release heading: "## v1.2.0 -
// 2006-01-02", optionally with the title linked, as well as the older
// "## [v1.2.0] - 2006-01-02" and release-please's "## [1.2.0](url) (2006-01-02)". The key is empty for any other text, except a
// bracketed title without a date, which still has a key.
func parseHeading(chunk string) (key string, date time.Time, hasDate bool) {
	line, _, _ := strings.Cut(chunk, "\n")
	title, ok := strings.CutPrefix(line, "## ")
	if !ok {
		return "", time.Time{}, false
	}
	title = headingLinkRe.ReplaceAllString(strings.TrimSpace(title), "$1")
	m := headingDateRe.FindStringSubmatch(title)
	if m == nil {
		if rest, ok := strings.CutPrefix(title, "["); ok {
			if key, _, ok := strings.Cut(rest, "]"); ok {
				return key, time.Time{}, false
			}
		}
		return "", time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", m[2]+m[3])
	if err != nil {
		return "", time.Time{}, false
	}
	key = strings.TrimSuffix(strings.TrimPrefix(m[1], "["), "]")
	return key, t, key != ""
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

// PRBody renders the changelog part of a release PR body. A single project
// gets its release notes directly under a version heading. Multiple projects
// get a version table, then breaking changes from all projects collected
// into one section (with per-project subsections), then a section per
// project with the remaining changes.
func PRBody(entries []Entry) string {
	if len(entries) == 1 {
		e := entries[0]
		return strings.TrimRight(Render(e), "\n") + "\n"
	}

	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Project.Name() < sorted[j].Project.Name()
	})

	var b strings.Builder

	b.WriteString("| Project | Version |\n|---|---|\n")
	for _, e := range sorted {
		fmt.Fprintf(&b, "| %s | %s |\n", e.Project.Name(), versionChange(e))
	}
	b.WriteString("\n")

	type projBreaking struct {
		name    string
		commits []conventional.ParsedCommit
	}
	var allBreaking []projBreaking
	links := make(map[string]Links, len(sorted))
	for _, e := range sorted {
		links[e.Project.Name()] = e.Links
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
		b.WriteString("## ⚠ Breaking Changes\n\n")
		for _, pb := range allBreaking {
			fmt.Fprintf(&b, "### %s\n\n", pb.name)
			for _, c := range pb.commits {
				b.WriteString(renderBreakingLine(links[pb.name], c))
			}
			b.WriteString("\n")
		}
	}

	for _, e := range sorted {
		fmt.Fprintf(&b, "## %s %s\n\n", e.Project.Name(), e.NewVersion)

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
				b.WriteString(renderLine(e.Links, c))
			}
			b.WriteString("\n")
		}

		if conventionalCount == 0 {
			b.WriteString(renderFallback(e))
		} else if !nonBreakingRendered {
			// only breaking commits — point readers to the section above
			b.WriteString("_Breaking changes only, see above._\n\n")
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// versionChange renders "`old` → `new`", or just "`new` (new)" for a
// project's first release.
func versionChange(e Entry) string {
	if e.OldVersion == "" || e.Reason == versioning.ReasonInitial {
		return fmt.Sprintf("`%s` (new)", e.NewVersion)
	}
	return fmt.Sprintf("`%s` → `%s`", e.OldVersion, e.NewVersion)
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
