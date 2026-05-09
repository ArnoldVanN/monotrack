package conventional

import "testing"

func TestDeriveBumpKind(t *testing.T) {
	tests := []struct {
		name string
		in   []ParsedCommit
		want Kind
	}{
		{"empty", nil, KindNone},
		{"breaking wins", []ParsedCommit{
			{Type: TypeFeat},
			{Type: TypeFix, Breaking: true},
		}, KindMajor},
		{"feat over fix", []ParsedCommit{
			{Type: TypeFix},
			{Type: TypeFeat},
		}, KindMinor},
		{"only fix", []ParsedCommit{{Type: TypeFix}}, KindPatch},
		{"only chore", []ParsedCommit{{Type: TypeChore}}, KindPatch},
		{"unknown only", []ParsedCommit{{Type: TypeUnknown}}, KindPatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveBumpKind(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
