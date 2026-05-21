package conventional

import (
	"regexp"
	"strings"
)

type CommitType string

const (
	TypeFeat     CommitType = "feat"
	TypeFix      CommitType = "fix"
	TypePerf     CommitType = "perf"
	TypeRefactor CommitType = "refactor"
	TypeChore    CommitType = "chore"
	TypeDocs     CommitType = "docs"
	TypeStyle    CommitType = "style"
	TypeTest     CommitType = "test"
	TypeBuild    CommitType = "build"
	TypeCI       CommitType = "ci"
	TypeRevert   CommitType = "revert"
	TypeUnknown  CommitType = ""
)

type ParsedCommit struct {
	Hash        string
	Subject     string
	Type        CommitType
	Scope       string
	Description string
	Breaking    bool
	Body        string
}

func (c ParsedCommit) ShortHash() string {
	if len(c.Hash) > 7 {
		return c.Hash[:7]
	}
	return c.Hash
}

var headerRe = regexp.MustCompile(`^(?P<type>[a-zA-Z]+)(?:\((?P<scope>[^)]+)\))?(?P<bang>!)?: (?P<desc>.+)$`)

// ParseAll returns one ParsedCommit per entry in a
// BEGIN_COMMIT_OVERRIDE/END_COMMIT_OVERRIDE block (blank-line separated), or
// the single Parse result when no override is present. All entries share the
// original hash. Lets users override changelog wording by editing the source
// commit body — works reliably with squash-merge.
func ParseAll(hash, raw string) []ParsedCommit {
	override, ok := extractOverride(raw)
	if !ok {
		return []ParsedCommit{Parse(hash, raw)}
	}

	parts := splitOverrideEntries(override)
	if len(parts) == 0 {
		return []ParsedCommit{Parse(hash, raw)}
	}
	out := make([]ParsedCommit, 0, len(parts))
	for _, p := range parts {
		out = append(out, Parse(hash, p))
	}
	return out
}

const (
	overrideBegin = "BEGIN_COMMIT_OVERRIDE"
	overrideEnd   = "END_COMMIT_OVERRIDE"
)

func extractOverride(raw string) (string, bool) {
	lines := strings.Split(raw, "\n")
	start, end := -1, -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == overrideBegin && start == -1 {
			start = i
		} else if t == overrideEnd && start != -1 {
			end = i
			break
		}
	}
	if start == -1 || end == -1 || end <= start+1 {
		return "", false
	}
	return strings.Join(lines[start+1:end], "\n"), true
}

func splitOverrideEntries(block string) []string {
	var entries []string
	var cur strings.Builder
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			entries = append(entries, s)
		}
		cur.Reset()
	}
	for l := range strings.SplitSeq(block, "\n") {
		if strings.TrimSpace(l) == "" {
			flush()
			continue
		}
		if cur.Len() > 0 {
			cur.WriteByte('\n')
		}
		cur.WriteString(l)
	}
	flush()
	return entries
}

func Parse(hash, raw string) ParsedCommit {
	pc := ParsedCommit{Hash: hash}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return pc
	}

	lines := strings.SplitN(raw, "\n", 2)
	pc.Subject = strings.TrimSpace(lines[0])
	if len(lines) == 2 {
		pc.Body = strings.TrimSpace(lines[1])
	}

	m := headerRe.FindStringSubmatch(pc.Subject)
	if m == nil {
		return pc
	}

	pc.Type = CommitType(strings.ToLower(m[headerRe.SubexpIndex("type")]))
	pc.Scope = m[headerRe.SubexpIndex("scope")]
	pc.Description = strings.TrimSpace(m[headerRe.SubexpIndex("desc")])
	if m[headerRe.SubexpIndex("bang")] == "!" {
		pc.Breaking = true
	}

	for line := range strings.SplitSeq(pc.Body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "BREAKING CHANGE:") || strings.HasPrefix(trimmed, "BREAKING-CHANGE:") {
			pc.Breaking = true
			break
		}
	}

	if !pc.Type.isKnown() {
		pc.Type = TypeUnknown
	}

	return pc
}

func (t CommitType) isKnown() bool {
	switch t {
	case TypeFeat, TypeFix, TypePerf, TypeRefactor, TypeChore,
		TypeDocs, TypeStyle, TypeTest, TypeBuild, TypeCI, TypeRevert:
		return true
	}
	return false
}
