package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/arnoldvann/monotrack/internal/projects"
)

func GitDiff(base string, head string) ([]string, error) {
	path, err := GetRepoRoot()
	if err != nil {
		return nil, err
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

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting repo root: %s", out)
	}
	return strings.TrimSpace(string(out)), nil
}

func IsShallowRepo() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--is-shallow-repository")

	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("error checking for shallow clone: %w", err)
	}

	b, err := strconv.ParseBool(strings.TrimSpace(string(out)))
	if err != nil {
		return false, fmt.Errorf("unexpected git output: %q", out)
	}

	return b, nil
}

func GetTagsForProjects(p map[string]projects.Project) (map[projects.Project][]string, error) {
	cmd := exec.Command("git", "tag", "--list")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error getting tags: %w: %s", err, out)
	}

	allTags := strings.Split(strings.TrimSpace(string(out)), "\n")
	filtered := filterTagsForProjects(allTags, p)

	return filtered, nil
}

func filterTagsForProjects(tags []string, proj map[string]projects.Project) map[projects.Project][]string {
	filteredTags := make(map[projects.Project][]string, 0)
	for _, p := range proj {
		for _, t := range tags {
			if strings.HasPrefix(t, p.Name()+"/") {
				filteredTags[p] = append(filteredTags[p], t)
			}
		}
	}

	return filteredTags
}

func PushTags(tags []string) error {
	args := append([]string{"push", "origin"}, makeRefs(tags)...)
	cmd := exec.Command("git", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error pushing tags: %s, %w", string(out), err)
	}

	return nil
}

func makeRefs(tags []string) []string {
	refs := make([]string, len(tags))
	for i, tag := range tags {
		refs[i] = fmt.Sprintf("refs/tags/%s", tag)
	}
	return refs
}

func CreateTag(tag, message, commit string) error {
	cmd := exec.Command("git", "tag", "-a", tag, "-m", message, commit)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating tag %q: %s", tag, out)
	}

	return nil
}

func TagExistsOnRemote(tag string) (bool, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "origin", fmt.Sprintf("refs/tags/%s", tag))

	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git ls-remote failed: %w", err)
	}

	return len(bytes.TrimSpace(out)) > 0, nil
}

func GetHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting head commit: %w: %s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}

func GetBase(tag string) (string, error) {
	cmd := exec.Command("git", "rev-list", "-n", "1", tag)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting base commit: %w: %s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}
