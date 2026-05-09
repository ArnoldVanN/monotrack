package conventional

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantType CommitType
		wantScope string
		wantBreak bool
		wantDesc  string
	}{
		{"feat with scope", "feat(api): add endpoint", TypeFeat, "api", false, "add endpoint"},
		{"fix no scope", "fix: handle nil", TypeFix, "", false, "handle nil"},
		{"breaking bang", "feat(api)!: drop v1", TypeFeat, "api", true, "drop v1"},
		{"breaking footer", "feat: thing\n\nBREAKING CHANGE: removed flag", TypeFeat, "", true, "thing"},
		{"breaking-change hyphen", "fix: x\n\nBREAKING-CHANGE: y", TypeFix, "", true, "x"},
		{"unknown type kept lower", "wip: stuff", TypeUnknown, "", false, ""},
		{"non-conventional", "WIP fix stuff", TypeUnknown, "", false, ""},
		{"perf", "perf(db): cache query", TypePerf, "db", false, "cache query"},
		{"empty", "", TypeUnknown, "", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := Parse("abc1234567", tt.raw)
			if pc.Type != tt.wantType {
				t.Errorf("type = %q, want %q", pc.Type, tt.wantType)
			}
			if pc.Scope != tt.wantScope {
				t.Errorf("scope = %q, want %q", pc.Scope, tt.wantScope)
			}
			if pc.Breaking != tt.wantBreak {
				t.Errorf("breaking = %v, want %v", pc.Breaking, tt.wantBreak)
			}
			if tt.wantDesc != "" && pc.Description != tt.wantDesc {
				t.Errorf("desc = %q, want %q", pc.Description, tt.wantDesc)
			}
		})
	}
}

func TestShortHash(t *testing.T) {
	pc := ParsedCommit{Hash: "abcdef1234567890"}
	if pc.ShortHash() != "abcdef1" {
		t.Errorf("got %q, want abcdef1", pc.ShortHash())
	}
	pc.Hash = "abc"
	if pc.ShortHash() != "abc" {
		t.Errorf("short hash for short input should be unchanged")
	}
}
