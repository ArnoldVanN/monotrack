package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Monotrack",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%v\n%v\n", version, date)
	},
}
