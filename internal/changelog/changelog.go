package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/versioning/conventional"
)

// TODO: customizable header
const fileHeader = `# Changelog

`

type Entry struct {
	Project    projects.Project
	OldVersion string
	NewVersion string
	Date       time.Time
	Commits    []conventional.ParsedCommit
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
// all (e.g. it was bumped only because a dependency changed), an "Other"
// section is added with a single "Updated internal dependencies" entry.
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
		b.WriteString("### Other\n\n- Updated internal dependencies\n\n")
	}

	return b.String()
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
		dateStr := e.Date.Format("2006-01-02")
		fmt.Fprintf(&block, "## [%s %s] - %s\n\n", e.Project.Name(), e.NewVersion, dateStr)
		body := Render(e)
		// strip the per-entry "## [version] - date" header that Render added
		body = stripFirstHeading(body)
		block.WriteString(body)
	}

	return prepend(path, block.String())
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
			b.WriteString("### Other\n\n- Updated internal dependencies\n\n")
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
