package tag

import (
	"github.com/spf13/cobra"
)

func init() {
	TagCmd.AddCommand(listCmd)
	TagCmd.AddCommand(getCmd)
}

var (
	TagCmd = &cobra.Command{
		Use:   "tag",
		Short: "Perform operations on tags",
	}
)
