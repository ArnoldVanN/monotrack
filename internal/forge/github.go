package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// githubCLI shells out to `gh`, using whatever auth gh already has
// (GITHUB_TOKEN in CI, or `gh auth login` locally).
type githubCLI struct {
	owner, repo string
}

func (g *githubCLI) Name() string { return "github" }

func (g *githubCLI) EnsurePR(ctx context.Context, req PRRequest) (PRResult, error) {
	existing, err := g.findOpenPR(ctx, req.Head)
	if err != nil {
		return PRResult{}, err
	}
	if existing != nil {
		_, err := runOut(ctx, "gh", "pr", "edit", fmt.Sprintf("%d", existing.Number),
			"--title", req.Title,
			"--body", req.Body,
		)
		if err != nil {
			return PRResult{}, fmt.Errorf("gh pr edit: %w", err)
		}
		return PRResult{URL: existing.URL, Number: existing.Number, Created: false}, nil
	}

	out, err := runOut(ctx, "gh", "pr", "create",
		"--base", req.Base,
		"--head", req.Head,
		"--title", req.Title,
		"--body", req.Body,
	)
	if err != nil {
		return PRResult{}, fmt.Errorf("gh pr create: %w", err)
	}
	url := strings.TrimSpace(out)
	return PRResult{URL: url, Number: parseGitHubPRNumber(url), Created: true}, nil
}

type ghPR struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

func (g *githubCLI) findOpenPR(ctx context.Context, head string) (*ghPR, error) {
	out, err := runOut(ctx, "gh", "pr", "list",
		"--head", head,
		"--state", "open",
		"--json", "number,url",
		"--limit", "1",
	)
	if err != nil {
		return nil, fmt.Errorf("gh pr list: %w", err)
	}
	var prs []ghPR
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, fmt.Errorf("parsing gh pr list output: %w", err)
	}
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

func parseGitHubPRNumber(url string) int {
	idx := strings.LastIndex(url, "/pull/")
	if idx < 0 {
		return 0
	}
	tail := url[idx+len("/pull/"):]
	var n int
	for _, r := range tail {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
