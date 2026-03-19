package cmd

import (
	"fmt"
	"os"

	"github.com/driangle/agent-runner/go/claudecode"
	"github.com/driangle/skival/internal/executor"
	"github.com/driangle/skival/internal/report"
	"github.com/driangle/skival/internal/suite"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <suite.yaml>",
	Short: "Run an eval suite",
	Long:  "Execute an eval suite definition against configured treatments and collect results.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := suite.Load(args[0])
		if err != nil {
			return fmt.Errorf("loading suite: %w", err)
		}

		runner := claudecode.NewRunner()

		sr, err := executor.Execute(cmd.Context(), s, runner)
		if err != nil {
			return fmt.Errorf("executing suite: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		return report.Write(os.Stdout, sr, format)
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
