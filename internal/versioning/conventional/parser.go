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
