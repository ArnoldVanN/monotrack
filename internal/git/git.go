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

func GetTagsForProjects(cfg *projects.Config, p map[string]projects.Project) (map[projects.Project][]string, error) {
	cmd := exec.Command("git", "tag", "--list")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error getting tags: %w: %s", err, out)
	}

	allTags := strings.Split(strings.TrimSpace(string(out)), "\n")
	filtered := filterTagsForProjects(cfg, allTags, p)

	return filtered, nil
}

func filterTagsForProjects(cfg *projects.Config, tags []string, proj map[string]projects.Project) map[projects.Project][]string {
	filteredTags := make(map[projects.Project][]string, 0)
	for _, p := range proj {
		for _, t := range tags {
			if _, ok := cfg.MatchTag(t, p.Name()); ok {
				filteredTags[p] = append(filteredTags[p], t)
			}
		}
	}
	return filteredTags
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

func AddFiles(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	repo, err := GetRepoRoot()
	if err != nil {
		return err
	}
	args := append([]string{"-C", repo, "add", "--"}, paths...)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %w: %s", err, out)
	}
	return nil
}

// CommitPaths commits only the given paths with the provided message. Returns
// the resulting HEAD SHA. Unrelated staged/unstaged changes are preserved.
func CommitPaths(message string, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("CommitPaths called with no paths")
	}
	repo, err := GetRepoRoot()
	if err != nil {
		return "", err
	}
	args := append([]string{"-C", repo, "commit", "-m", message, "--"}, paths...)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git commit failed: %w: %s", err, out)
	}
	return GetHead()
}

// CurrentBranch returns the checked-out branch name, or "" when on a detached
// HEAD.
func CurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --abbrev-ref HEAD failed: %w: %s", err, out)
	}
	name := strings.TrimSpace(string(out))
	if name == "HEAD" {
		return "", nil
	}
	return name, nil
}

// PushRefsAtomic pushes the given fully-qualified refs to origin atomically.
// Returns ErrNonFastForward when the push was rejected because the remote has
// diverged.
func PushRefsAtomic(refs []string) error {
	if len(refs) == 0 {
		return nil
	}
	args := append([]string{"push", "--atomic", "origin"}, refs...)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if isNonFastForward(out) {
			return fmt.Errorf("%w: %s", ErrNonFastForward, out)
		}
		return fmt.Errorf("git push --atomic failed: %s: %w", out, err)
	}
	return nil
}

// ErrNonFastForward indicates a push was rejected because the remote ref
// advanced concurrently. The caller should fetch + rebase + retry.
var ErrNonFastForward = fmt.Errorf("non-fast-forward")

func isNonFastForward(out []byte) bool {
	s := string(out)
	return strings.Contains(s, "non-fast-forward") ||
		strings.Contains(s, "fetch first") ||
		strings.Contains(s, "[rejected]")
}

// HasUncommittedChanges reports whether the working tree or index differs
// from HEAD.
func HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

// CheckoutBranchFrom creates or resets branch at startPoint and switches to
// it (`git checkout -B`).
func CheckoutBranchFrom(branch, startPoint string) error {
	cmd := exec.Command("git", "checkout", "-B", branch, startPoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -B %s %s failed: %s: %w", branch, startPoint, out, err)
	}
	return nil
}

func CheckoutBranch(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout %s failed: %s: %w", branch, out, err)
	}
	return nil
}

func ForcePushBranch(branch, localRef string) error {
	if branch == "" {
		return fmt.Errorf("ForcePushBranch: empty branch")
	}
	ref := localRef
	if ref == "" {
		ref = "HEAD"
	}
	cmd := exec.Command("git", "push", "--force", "origin", fmt.Sprintf("%s:refs/heads/%s", ref, branch))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push --force origin %s failed: %s: %w", branch, out, err)
	}
	return nil
}

// FetchBranch fetches the given branch from origin into FETCH_HEAD.
func FetchBranch(branch string) error {
	cmd := exec.Command("git", "fetch", "origin", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch origin %s failed: %s: %w", branch, out, err)
	}
	return nil
}

// RebaseOnto rebases the current branch onto the given commit-ish and returns
// the new HEAD.
func RebaseOnto(onto string) (string, error) {
	cmd := exec.Command("git", "rebase", onto)
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command("git", "rebase", "--abort").Run()
		return "", fmt.Errorf("git rebase %s failed: %s: %w", onto, out, err)
	}
	return GetHead()
}

// DeleteLocalTag removes a local tag. Used when recreating tags after a rebase.
func DeleteLocalTag(tag string) error {
	cmd := exec.Command("git", "tag", "-d", tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git tag -d %s failed: %s: %w", tag, out, err)
	}
	return nil
}

func RemoteURL(remote string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remote)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git remote get-url %s failed: %s: %w", remote, out, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func GetHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting head commit: %w: %s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}

type RawCommit struct {
	Hash    string
	Message string
}

// LogBetween returns commits in base..head that touched any of the given paths.
// If paths is empty, returns all commits in the range.
func LogBetween(base, head string, paths []string) ([]RawCommit, error) {
	repo, err := GetRepoRoot()
	if err != nil {
		return nil, err
	}

	args := []string{"-C", repo, "log", fmt.Sprintf("%s..%s", base, head), "--format=%H%x1f%B%x1e"}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w: %s", err, out)
	}

	trimmed := strings.TrimSpace(strings.Trim(string(out), "\x1e\n"))
	if trimmed == "" {
		return nil, nil
	}

	var commits []RawCommit
	for rec := range strings.SplitSeq(trimmed, "\x1e") {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, "\x1f", 2)
		if len(parts) != 2 {
			continue
		}
		commits = append(commits, RawCommit{
			Hash:    strings.TrimSpace(parts[0]),
			Message: strings.TrimSpace(parts[1]),
		})
	}

	return commits, nil
}

func GetBase(tag string) (string, error) {
	cmd := exec.Command("git", "rev-list", "-n", "1", tag)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting base commit: %w: %s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}

func GetCommitMessage(ref string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%B", ref)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git log --format=%%B %s failed: %s: %w", ref, out, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func DeleteRemoteTag(tag string) error {
	cmd := exec.Command("git", "push", "origin", ":refs/tags/"+tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push origin :refs/tags/%s failed: %s: %w", tag, out, err)
	}
	return nil
}

// tagDeleteChunk bounds how many tags are passed to a single git invocation so
// we don't blow the OS argument limit when pruning hundreds of tags.
const tagDeleteChunk = 100

// DeleteLocalTags removes the given local tags in chunked `git tag -d` calls.
func DeleteLocalTags(tags []string) error {
	for chunk := range chunked(tags, tagDeleteChunk) {
		args := append([]string{"tag", "-d"}, chunk...)
		cmd := exec.Command("git", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git tag -d failed: %s: %w", out, err)
		}
	}
	return nil
}

// DeleteRemoteTags deletes the given tags on origin in chunked
// `git push origin --delete` calls.
func DeleteRemoteTags(tags []string) error {
	for chunk := range chunked(tags, tagDeleteChunk) {
		refs := make([]string, 0, len(chunk))
		for _, t := range chunk {
			refs = append(refs, "refs/tags/"+t)
		}
		args := append([]string{"push", "origin", "--delete"}, refs...)
		cmd := exec.Command("git", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git push origin --delete failed: %s: %w", out, err)
		}
	}
	return nil
}

func chunked(items []string, size int) func(func([]string) bool) {
	return func(yield func([]string) bool) {
		for i := 0; i < len(items); i += size {
			end := min(i+size, len(items))
			if !yield(items[i:end]) {
				return
			}
		}
	}
}

func ResetHard(ref string) error {
	cmd := exec.Command("git", "reset", "--hard", ref)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git reset --hard %s failed: %s: %w", ref, out, err)
	}
	return nil
}

// IsAncestor reports whether commit is an ancestor of head.
func IsAncestor(commit, head string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", commit, head)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("error checking ancestry of %q in %q: %w", commit, head, err)
}
