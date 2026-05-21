package projects

import "testing"

func TestResolveReleaseBranch(t *testing.T) {
	tests := []struct {
		name string
		cfg  ReleaseConfig
		base string
		want string
	}{
		{"default empty", ReleaseConfig{}, "main", "monotrack/release-main"},
		{"default different base", ReleaseConfig{}, "develop", "monotrack/release-develop"},
		{"explicit literal", ReleaseConfig{Branch: "release-please-pr"}, "main", "release-please-pr"},
		{"custom template", ReleaseConfig{Branch: "auto-release/{base}"}, "trunk", "auto-release/trunk"},
		{"template without placeholder", ReleaseConfig{Branch: "next-release"}, "main", "next-release"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ResolveReleaseBranch(tt.base)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
