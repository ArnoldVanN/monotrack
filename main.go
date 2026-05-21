package main

import (
	"os"

	"github.com/arnoldvann/monotrack/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := cmd.Execute(version, commit, date); err != nil {
		os.Exit(1)
	}
}
