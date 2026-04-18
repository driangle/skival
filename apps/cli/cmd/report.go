package cmd

import (
	"fmt"
	"os"

	"github.com/driangle/skival/internal/persist"
	"github.com/driangle/skival/internal/report"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report <results-dir>",
	Short: "Generate reports from saved results",
	Long:  "Generate markdown or JSON reports from previously collected eval results.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sr, err := persist.Load(args[0])
		if err != nil {
			return fmt.Errorf("loading results: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		return report.Write(os.Stdout, sr, format, report.DefaultWeights())
	},
}

func init() {
	reportCmd.Flags().String("format", "markdown", "Output format: markdown, json")

	rootCmd.AddCommand(reportCmd)
}
