package cmd

import (
	"fmt"
	"os"

	"github.com/driangle/agent-runner/go/claudecode"
	"github.com/driangle/skival/internal/executor"
	"github.com/driangle/skival/internal/persist"
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

		evalIDs, _ := cmd.Flags().GetStringSlice("evals")
		treatments, _ := cmd.Flags().GetStringSlice("treatments")

		execOpts := &executor.Options{
			EvalIDs:    evalIDs,
			Treatments: treatments,
			Progress:   os.Stderr,
		}

		sr, err := executor.Execute(cmd.Context(), s, runner, execOpts)
		if err != nil {
			return fmt.Errorf("executing suite: %w", err)
		}

		resultsDir, _ := cmd.Flags().GetString("results-dir")
		if resultsDir != "" {
			outDir, err := persist.Save(resultsDir, sr)
			if err != nil {
				return fmt.Errorf("saving results: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Results saved to %s\n", outDir)
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
