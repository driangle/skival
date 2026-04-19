package cmd

import (
	"fmt"

	"github.com/driangle/skival/internal/compare"
	"github.com/driangle/skival/internal/persist"
	"github.com/spf13/cobra"
)

var compareCmd = &cobra.Command{
	Use:   "compare <baseline-dir> <candidate-dir>",
	Short: "Compare results between two runs",
	Long:  "Load two result directories and produce a diff report showing how variants changed between runs.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		baseline, err := persist.Load(args[0])
		if err != nil {
			return fmt.Errorf("loading baseline: %w", err)
		}

		candidate, err := persist.Load(args[1])
		if err != nil {
			return fmt.Errorf("loading candidate: %w", err)
		}

		comparison := compare.Compare(baseline, candidate)

		format, _ := cmd.Flags().GetString("format")
		return compare.Write(cmd.OutOrStdout(), comparison, format)
	},
}

func init() {
	compareCmd.Flags().String("format", "markdown", "Output format: markdown, json")

	rootCmd.AddCommand(compareCmd)
}
