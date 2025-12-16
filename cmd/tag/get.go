package tag

import (
	"fmt"

	"github.com/spf13/cobra"
)

// func init() {
// }

var (
	getCmd = &cobra.Command{
		Use:   "get",
		Short: "List tag for a specific project",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Monotrack versioning tool v0.0.1 -- HEAD")
		},
	}
)
