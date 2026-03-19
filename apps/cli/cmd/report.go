package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report <results-dir>",
	Short: "Generate reports from saved results",
	Long:  "Generate markdown or JSON reports from previously collected eval results.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Generating report from: %s\n", args[0])
		return nil
	},
}

func init() {
	reportCmd.Flags().String("format", "markdown", "Output format: markdown, json")

	rootCmd.AddCommand(reportCmd)
}
