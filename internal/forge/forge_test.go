package forge

import "testing"

func TestParseRemote(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantHost  string
		wantOwner string
		wantRepo  string
	}{
		{"github https", "https://github.com/foo/bar.git", "github.com", "foo", "bar"},
		{"github https no .git", "https://github.com/foo/bar", "github.com", "foo", "bar"},
		{"github ssh shorthand", "git@github.com:foo/bar.git", "github.com", "foo", "bar"},
		{"github ssh url form", "ssh://git@github.com/foo/bar.git", "github.com", "foo", "bar"},
		{"gitlab self-hosted", "https://gitlab.example.com/group/project.git", "gitlab.example.com", "group", "project"},
		{"trailing slash", "https://github.com/foo/bar/", "github.com", "foo", "bar"},
		{"unrecognized", "file:///tmp/repo", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, owner, repo := parseRemote(tt.url)
			if host != tt.wantHost || owner != tt.wantOwner || repo != tt.wantRepo {
				t.Errorf("got (%q, %q, %q), want (%q, %q, %q)",
					host, owner, repo, tt.wantHost, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

func TestGenericCompareURL(t *testing.T) {
	tests := []struct {
		name string
		g    *generic
		want string
	}{
		{
			"github",
			&generic{host: "github.com", owner: "o", repo: "r", remoteURL: "ignored"},
			"https://github.com/o/r/compare/main...release?expand=1",
		},
		{
			"gitlab",
			&generic{host: "gitlab.com", owner: "o", repo: "r", remoteURL: "ignored"},
			"https://gitlab.com/o/r/-/merge_requests/new?merge_request[source_branch]=release&merge_request[target_branch]=main",
		},
		{
			"unknown host returns remote URL",
			&generic{host: "code.example.com", owner: "o", repo: "r", remoteURL: "https://code.example.com/o/r.git"},
			"https://code.example.com/o/r.git",
		},
		{
			"unparseable returns remote URL",
			&generic{remoteURL: "weird://thing"},
			"weird://thing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.g.compareURL("main", "release")
			if got != tt.want {
				t.Errorf("got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestParseGitHubPRNumber(t *testing.T) {
	tests := []struct {
		url  string
		want int
	}{
		{"https://github.com/o/r/pull/42", 42},
		{"https://github.com/o/r/pull/1234?foo=bar", 1234},
		{"https://github.com/o/r", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := parseGitHubPRNumber(tt.url)
		if got != tt.want {
			t.Errorf("parseGitHubPRNumber(%q) = %d, want %d", tt.url, got, tt.want)
		}
	}
}
