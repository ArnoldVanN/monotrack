package printer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// Format resolves -o against the formats a subcommand actually implements,
// so an unsupported value errors instead of silently falling back to plain.
func Format(cmd *cobra.Command, supported ...string) (string, error) {
	format := "plain"
	if f := cmd.InheritedFlags().Lookup("out"); f != nil && f.Value.String() != "" {
		format = f.Value.String()
	}
	if slices.Contains(supported, format) {
		return format, nil
	}
	return "", fmt.Errorf("unsupported output format %q, want one of: %s", format, strings.Join(supported, ", "))
}
