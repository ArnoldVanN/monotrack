package forge

import (
	"context"
	"fmt"
)

// generic returns a compare/MR-new URL for the user to click; it assumes the
// caller has already pushed the head branch.
type generic struct {
	host, owner, repo string
	remoteURL         string
}

func (g *generic) Name() string { return "generic" }

func (g *generic) EnsurePR(ctx context.Context, req PRRequest) (PRResult, error) {
	return PRResult{URL: g.compareURL(req.Base, req.Head)}, nil
}

func (g *generic) compareURL(base, head string) string {
	if g.owner == "" || g.repo == "" {
		return g.remoteURL
	}
	switch g.host {
	case "github.com":
		return fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s?expand=1",
			g.owner, g.repo, base, head)
	case "gitlab.com":
		return fmt.Sprintf("https://gitlab.com/%s/%s/-/merge_requests/new?merge_request[source_branch]=%s&merge_request[target_branch]=%s",
			g.owner, g.repo, head, base)
	default:
		return g.remoteURL
	}
}
