package tag

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/arnoldvann/monotrack/internal/app"
	"github.com/arnoldvann/monotrack/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	undoCmd.Flags().BoolVar(&undoDry, "dry", false, "preview what would be undone without making changes")
	undoCmd.Flags().BoolVarP(&undoYes, "yes", "y", false, "skip confirmation prompt")
}

var (
	undoDry bool
	undoYes bool

	undoCmd = &cobra.Command{
		Use:   "undo",
		Short: "Undo the most recent release (delete tags, reset changelog commit, force-push)",
		RunE:  runUndo,
	}
)

type releaseEntry struct {
	project    string
	oldVersion string
	newVersion string
	tag        string
}

func parseReleaseCommit(message string, tagFor func(string, string) string) ([]releaseEntry, error) {
	parts := strings.SplitN(message, "\n", 2)
	subject := strings.TrimSpace(parts[0])
	if !strings.HasPrefix(subject, "chore(release)") {
		return nil, fmt.Errorf("HEAD is not a release commit (expected \"chore(release): ...\" subject)")
	}
	if len(parts) < 2 {
		return nil, fmt.Errorf("release commit has no body listing projects")
	}

	var entries []releaseEntry
	for line := range strings.SplitSeq(parts[1], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// A line is expected to be "- <name> <old> -> <new>"
		fields := strings.Fields(line[2:])
		if len(fields) != 4 || fields[2] != "->" {
			return nil, fmt.Errorf("malformed project entry in release commit body: %q", line)
		}
		name, oldV, newV := fields[0], fields[1], fields[3]
		entries = append(entries, releaseEntry{
			project:    name,
			oldVersion: oldV,
			newVersion: newV,
			tag:        tagFor(name, newV),
		})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("could not parse any project entries from release commit body")
	}
	return entries, nil
}

func runUndo(cmd *cobra.Command, args []string) error {
	branch, err := git.CurrentBranch()
	if err != nil {
		return err
	}
	if branch == "" {
		return fmt.Errorf("undo requires a checked-out branch (detached HEAD)")
	}

	msg, err := git.GetCommitMessage("HEAD")
	if err != nil {
		return err
	}

	entries, err := parseReleaseCommit(msg, app.State.Config.TagFor)
	if err != nil {
		return err
	}

	remoteExists := make(map[string]bool, len(entries))
	anyRemote := false
	for _, e := range entries {
		exists, err := git.TagExistsOnRemote(e.tag)
		if err != nil {
			return fmt.Errorf("checking remote tag %q: %w", e.tag, err)
		}
		remoteExists[e.tag] = exists
		if exists {
			anyRemote = true
		}
	}

	fmt.Fprintln(os.Stderr, "will undo the following release:")
	fmt.Fprintln(os.Stderr)
	for _, e := range entries {
		remote := ""
		if remoteExists[e.tag] {
			remote = " (exists on remote)"
		}
		fmt.Fprintf(os.Stderr, "  delete tag %s%s\n", e.tag, remote)
	}
	fmt.Fprintf(os.Stderr, "  reset changelog commit on %s\n", branch)
	fmt.Fprintf(os.Stderr, "  force-push %s\n", branch)
	fmt.Fprintln(os.Stderr)

	if anyRemote {
		fmt.Fprintln(os.Stderr, "warning: some of these tags exist on the remote. If pushing them")
		fmt.Fprintln(os.Stderr, "already triggered CI, undo will NOT reverse published artifacts (images,")
		fmt.Fprintln(os.Stderr, "charts, packages), deployments, or GitHub releases created from the tag.")
		fmt.Fprintln(os.Stderr, "Roll forward with a new release instead if any of that has happened.")
		fmt.Fprintln(os.Stderr)
	}

	if undoDry {
		return nil
	}

	if !undoYes {
		fmt.Fprint(os.Stderr, "are you sure? this cannot be undone. [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		}
	}

	for _, e := range entries {
		if err := git.DeleteLocalTag(e.tag); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		}
	}

	var remoteErrs []string
	for _, e := range entries {
		if !remoteExists[e.tag] {
			continue
		}
		if err := git.DeleteRemoteTag(e.tag); err != nil {
			remoteErrs = append(remoteErrs, fmt.Sprintf("  %s: %v", e.tag, err))
			continue
		}
		fmt.Fprintf(os.Stderr, "deleted remote tag %s\n", e.tag)
	}

	if err := git.ResetHard("HEAD~1"); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "reset changelog commit")

	if err := git.ForcePushBranch(branch, "HEAD"); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "force-pushed %s\n", branch)

	if len(remoteErrs) > 0 {
		return fmt.Errorf("some remote tags could not be deleted:\n%s", strings.Join(remoteErrs, "\n"))
	}
	return nil
}
