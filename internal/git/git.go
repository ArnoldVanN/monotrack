package git

import (
	"fmt"
	"os/exec"
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

func GetTagsForProjects(p map[string]projects.Project) (map[projects.Project][]string, error) {
	cmd := exec.Command("git", "tag", "--list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git command failed: %w: %s", err, out)
	}

	allTags := strings.Split(strings.TrimSpace(string(out)), "\n")

	filtered := filterTagsForProjects(allTags, p)

	for p, tags := range filtered {
		if len(tags) == 0 {
			filtered[p] = append(filtered[p], p.Name()+"/v0.0.0")
		}
	}

	return filtered, nil
}

func filterTagsForProjects(tags []string, proj map[string]projects.Project) map[projects.Project][]string {
	filteredTags := make(map[projects.Project][]string, 0)
	for _, p := range proj {
		for _, t := range tags {
			if strings.Contains(t, p.Name()) {
				filteredTags[p] = append(filteredTags[p], t)
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
