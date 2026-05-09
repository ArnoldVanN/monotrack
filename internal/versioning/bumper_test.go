package versioning

import "testing"

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		kind       BumpKind
		preRelease bool
		want       string
	}{
		// Stable → stable
		{"patch on stable", "v1.2.3", PatchBump, false, "v1.2.4"},
		{"minor on stable", "v1.2.3", MinorBump, false, "v1.3.0"},
		{"major on stable", "v1.2.3", MajorBump, false, "v2.0.0"},

		// Stable → start prerelease
		{"start prerelease patch", "v1.2.3", PatchBump, true, "v1.2.4-rc.1"},
		{"start prerelease minor", "v1.2.3", MinorBump, true, "v1.3.0-rc.1"},

		// Prerelease → bump rc counter
		{"bump rc counter", "v1.3.0-rc.1", MinorBump, true, "v1.3.0-rc.2"},
		{"bump rc counter patch ignored", "v1.3.0-rc.5", PatchBump, true, "v1.3.0-rc.6"},

		// Prerelease → promote to stable (kind ignored, suffix stripped)
		{"promote from rc", "v1.3.0-rc.3", PatchBump, false, "v1.3.0"},
		{"promote ignores kind", "v1.3.0-rc.3", MajorBump, false, "v1.3.0"},
		{"promote from rc.1", "v2.0.0-rc.1", MinorBump, false, "v2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bumpVersion(tt.in, tt.kind, tt.preRelease)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("bumpVersion(%q, %q, %v) = %q, want %q", tt.in, tt.kind, tt.preRelease, got, tt.want)
			}
		})
	}
}

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
