// Package forge abstracts the platform-specific PR operations needed by the
// propose phase of monotrack's release flow.
package forge

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/arnoldvann/monotrack/internal/git"
)

type PRRequest struct {
	Base  string
	Head  string
	Title string
	Body  string
}

// PRResult.URL is the PR URL for forges that opened/updated one, or the
// compare URL the user should click for the generic fallback.
type PRResult struct {
	URL     string
	Number  int
	Created bool
}

type Forge interface {
	Name() string
	EnsurePR(ctx context.Context, req PRRequest) (PRResult, error)
}

// Detect resolves origin's host and picks an implementation. Falls back to
// generic when the host isn't recognized or the host's CLI isn't installed.
func Detect() (Forge, error) {
	url, err := git.RemoteURL("origin")
	if err != nil {
		return nil, fmt.Errorf("detecting forge: %w", err)
	}
	host, owner, repo := parseRemote(url)
	if host == "github.com" && hasCLI("gh") {
		return &githubCLI{owner: owner, repo: repo}, nil
	}
	return &generic{host: host, owner: owner, repo: repo, remoteURL: url}, nil
}

// Matches https://host/o/r(.git), git@host:o/r(.git), ssh://git@host/o/r(.git).
var remoteRe = regexp.MustCompile(`(?:https?://|ssh://[^@]+@|[^@]+@)([^/:]+)[:/]([^/]+)/([^/]+?)(?:\.git)?/?$`)

func parseRemote(url string) (host, owner, repo string) {
	m := remoteRe.FindStringSubmatch(url)
	if m == nil {
		return "", "", ""
	}
	return m[1], m[2], m[3]
}

func hasCLI(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runOut(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %s: %w", name, strings.Join(args, " "), out, err)
	}
	return strings.TrimSpace(string(out)), nil
}
