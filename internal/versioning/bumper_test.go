package versioning

import "testing"

func TestIsReleaseCommit(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"chore(release): bump 2 project(s)", true},
		{"chore(release): bump\n\n- a v0.1 -> v0.2", true},
		{"  chore(release): leading space", true},
		{"feat: add thing", false},
		{"chore: cleanup", false},
		{"chore(deps): bump foo", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isReleaseCommit(tt.msg); got != tt.want {
			t.Errorf("isReleaseCommit(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}
