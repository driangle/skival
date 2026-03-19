package main

import (
	"os"

	"github.com/driangle/skival/apps/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
