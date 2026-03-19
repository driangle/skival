package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <suite.yaml>",
	Short: "Run an eval suite",
	Long:  "Execute an eval suite definition against configured treatments and collect results.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Running suite: %s\n", args[0])
		return nil
	},
}

func init() {
	runCmd.Flags().Int("samples", 1, "Number of runs per treatment")
	runCmd.Flags().String("results-dir", "", "Directory for results output")
	runCmd.Flags().StringSlice("treatments", nil, "Filter to specific treatments")
	runCmd.Flags().StringSlice("evals", nil, "Filter to specific eval IDs")
	runCmd.Flags().String("model", "", "Override model for all treatments")
	runCmd.Flags().String("format", "markdown", "Output format: markdown, json")

	rootCmd.AddCommand(runCmd)
}
