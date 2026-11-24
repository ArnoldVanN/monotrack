package changelog

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/arnoldvann/monotrack/internal/forge"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/arnoldvann/monotrack/internal/projects"
	"github.com/arnoldvann/monotrack/internal/versioning"
)

// Links builds web URLs into a repository on GitHub, GitLab or Forgejo. The
// zero value builds none, so changelogs render without links.
type Links struct {
	base string
	kind string
}

// knownHosts maps public forge hosts to their type.
var knownHosts = map[string]string{
	"github.com":   "github",
	"gitlab.com":   "gitlab",
	"codeberg.org": "forgejo",
}

// NewLinks returns the Links for a repo, from the forge config and origin's
// remote URL. The config's type and URL win over what the remote implies.
// It returns the zero value when the forge type can't be determined.
func NewLinks(cfg projects.ForgeConfig, remoteURL string) Links {
	base := strings.TrimSuffix(cfg.URL, "/")
	var host string
	if base != "" {
		if u, err := url.Parse(base); err == nil {
			host = u.Hostname()
		}
	} else {
		h, owner, repo := forge.ParseRemote(remoteURL)
		if owner == "" || repo == "" {
			return Links{}
		}
		host = h
		base = "https://" + host + "/" + owner + "/" + repo
	}
	kind := cfg.Type
	if kind == "" {
		kind = knownHosts[host]
	}
	if kind == "" {
		return Links{}
	}
	return Links{base: base, kind: kind}
}

// RepoLinks returns the Links for this repo, from the forge config and the
// origin remote.
func RepoLinks(cfg projects.ForgeConfig) Links {
	remote, _ := git.RemoteURL("origin")
	return NewLinks(cfg, remote)
}

// PrevTag returns the tag of the release r follows, or "" when there's no
// release to diff against: a first release, or one re-anchored after its
// previous tag left the history.
func PrevTag(cfg *projects.Config, r versioning.BumpResult) string {
	if r.OldVersion == "" || r.Reason == versioning.ReasonInitial || r.Reason == versioning.ReasonOrphaned {
		return ""
	}
	return cfg.TagFor(r.Project.Name(), r.OldVersion)
}

func (l Links) path(p string) string {
	if l.kind == "gitlab" {
		return l.base + "/-/" + p
	}
	return l.base + "/" + p
}

func (l Links) compare(from, to string) string {
	if l.base == "" || from == "" || to == "" {
		return ""
	}
	return l.path("compare/" + from + "..." + to)
}

// commit renders a commit's short hash, linked to the commit.
func (l Links) commit(c string, short string) string {
	if l.base == "" || c == "" {
		return short
	}
	return "[" + short + "](" + l.path("commit/"+c) + ")"
}

var prRefRe = regexp.MustCompile(`\(#(\d+)\)`)

// prRefs links "(#123)" references, as squash merges append to the subject,
// to the pull request. On GitLab "#123" is an issue, so it's left alone.
func (l Links) prRefs(s string) string {
	var prefix string
	switch l.kind {
	case "github":
		prefix = l.base + "/pull/"
	case "forgejo":
		prefix = l.base + "/pulls/"
	default:
		return s
	}
	return prRefRe.ReplaceAllString(s, "([#$1]("+prefix+"$1))")
}
