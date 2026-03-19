package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "skival",
	Short: "AI coding skill evaluator",
	Long:  "A CLI for evaluating AI coding skill performance. Measures time to completion, token usage, dollar cost, and correctness across configurable eval suites.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})))
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug-level logging")
}

func Execute() error {
	return rootCmd.Execute()
}
