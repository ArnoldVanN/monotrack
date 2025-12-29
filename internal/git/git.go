package git

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/arnoldvann/monotrack/internal/projects"
)

func GitDiff(base string, head string) ([]string, error) {
	path, err := GetRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo root: %w", err)
	}

	cmd := exec.Command("git", "-C", path, "diff", base, head, "--name-only")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w: %s", err, out)
	}

	if len(out) == 0 {
		return nil, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	return lines, nil
}

func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func GetTagsForProjects(p map[string]projects.Project) ([]string, error) {
	cmd := exec.Command("git", "tag", "--list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git command failed: %w: %s", err, out)
	}

	allTags := strings.Split(strings.TrimSpace(string(out)), "\n")

	filtered := filterTagsForProjects(allTags, p)

	if len(filtered) < len(p) {
		for _, p := range p {
			if !slices.ContainsFunc(filtered, func(t string) bool { return strings.Contains(t, p.Name()) }) {
				fmt.Printf("WARN: no tags found for project %q\n", p.Name())
			}
		}
	}

	return filtered, nil
}

func filterTagsForProjects(tags []string, projects map[string]projects.Project) []string {
	filteredTags := make([]string, 0)
	for _, p := range projects {
		for _, t := range tags {
			if strings.Contains(t, p.Name()) {
				filteredTags = append(filteredTags, t)
			}
		}
	}

	return filteredTags
}

func GetHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w: %s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}

func GetBase(tag string) (string, error) {
	cmd := exec.Command("git", "rev-list", "-n", "1", tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-list failed: %w: %s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}
