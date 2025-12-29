package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/arnoldvann/monotrack/internal/projects"
)

func GitDiff(base string, head string) (string, error) {
	path, err := GetRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get repo root: %w", err)
	}

	cmd := exec.Command("git", "-C", path, "diff", base, head, "--name-only")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git diff failed: %w: %s", err, out)
	}

	return string(out), nil
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
		return string(out), fmt.Errorf("git command failed: %w: %s", err, out)
	}

	return string(out), nil
}

func GetBase() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git command failed: %w: %s", err, out)
	}

	fmt.Printf(string(out))

	return string(out), nil
}
