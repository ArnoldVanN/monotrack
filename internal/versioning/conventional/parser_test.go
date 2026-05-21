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

func TestParseAll_NoOverride(t *testing.T) {
	got := ParseAll("abc1234", "feat: thing")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Type != TypeFeat {
		t.Errorf("type = %q, want feat", got[0].Type)
	}
}

func TestParseAll_OverrideBlock(t *testing.T) {
	raw := `chore: original boring subject

Some body text.

BEGIN_COMMIT_OVERRIDE
feat(api): expose new endpoint

fix(db): handle null gracefully
END_COMMIT_OVERRIDE

trailer ignored
`
	got := ParseAll("abc1234", raw)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Type != TypeFeat || got[0].Scope != "api" {
		t.Errorf("entry 0 = %+v, want feat(api)", got[0])
	}
	if got[1].Type != TypeFix || got[1].Scope != "db" {
		t.Errorf("entry 1 = %+v, want fix(db)", got[1])
	}
	for i, p := range got {
		if p.Hash != "abc1234" {
			t.Errorf("entry %d hash = %q, want abc1234", i, p.Hash)
		}
	}
}

func TestParseAll_MalformedOverrideFallsBack(t *testing.T) {
	raw := "feat: thing\n\nBEGIN_COMMIT_OVERRIDE\nfix: ignored"
	got := ParseAll("abc1234", raw)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (fallback)", len(got))
	}
	if got[0].Type != TypeFeat {
		t.Errorf("type = %q, want feat", got[0].Type)
	}
}

func TestParseAll_EmptyOverrideFallsBack(t *testing.T) {
	raw := "feat: thing\n\nBEGIN_COMMIT_OVERRIDE\nEND_COMMIT_OVERRIDE"
	got := ParseAll("abc1234", raw)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Type != TypeFeat {
		t.Errorf("type = %q, want feat", got[0].Type)
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
