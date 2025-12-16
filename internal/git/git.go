package git

import (
	"fmt"
	"os/exec"
	"strings"
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

func GetTags() (string, error) {
	cmd := exec.Command("git", "tag", "--list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git command failed: %w: %s", err, out)
	}

	return string(out), nil
}
