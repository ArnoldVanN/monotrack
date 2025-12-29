package tag

import (
	"github.com/spf13/cobra"
)

func init() {
	TagCmd.PersistentFlags().BoolVar(&includeProj, "proj-pref", true, "whether to include the project prefix")

	TagCmd.AddCommand(listCmd)
	TagCmd.AddCommand(getCmd)
	TagCmd.AddCommand(bumpCmd)
}

var (
	includeProj bool

	TagCmd = &cobra.Command{
		Use:   "tag",
		Short: "Perform operations on tags",
	}
)
