package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "skival",
	Short: "AI coding skill evaluator",
	Long:  "A CLI for evaluating AI coding skill performance. Measures time to completion, token usage, dollar cost, and correctness across configurable eval suites.",
}

func Execute() error {
	return rootCmd.Execute()
}
