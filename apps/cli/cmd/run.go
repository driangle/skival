package cmd

import (
	"fmt"
	"log/slog"
	"os"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/agentrunner/go/claudecode"
	"github.com/driangle/agentrunner/go/ollama"
	"github.com/driangle/skival/internal/executor"
	"github.com/driangle/skival/internal/persist"
	"github.com/driangle/skival/internal/registry"
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
		slog.Debug("Loading suite", "path", args[0])
		s, err := suite.Load(args[0])
		if err != nil {
			return fmt.Errorf("loading suite: %w", err)
		}
		slog.Debug("Suite loaded", "description", s.Description, "evals", len(s.Evals))

		reg := defaultRegistry()

		evalIDs, _ := cmd.Flags().GetStringSlice("evals")
		treatments, _ := cmd.Flags().GetStringSlice("treatments")
		samples, _ := cmd.Flags().GetInt("samples")
		parallel, _ := cmd.Flags().GetInt("parallel")
		timeout, _ := cmd.Flags().GetInt("timeout")
		slog.Debug("Filters", "evals", evalIDs, "treatments", treatments, "samples", samples, "parallel", parallel, "timeout", timeout)

		if timeout < 0 {
			return fmt.Errorf("--timeout must be a positive number of seconds")
		}
		if timeout == 0 && cmd.Flags().Changed("timeout") {
			return fmt.Errorf("--timeout must be a positive number of seconds")
		}

		execOpts := &executor.Options{
			EvalIDs:    evalIDs,
			Treatments: treatments,
			Progress:   os.Stderr,
			Samples:    samples,
			Parallel:   parallel,
			Timeout:    timeout,
		}

		sr, err := executor.Execute(cmd.Context(), s, reg, execOpts)
		if err != nil {
			return fmt.Errorf("executing suite: %w", err)
		}

		weights := rankingWeights(s)

		resultsDir, _ := cmd.Flags().GetString("results-dir")
		if resultsDir != "" {
			outDir, err := persist.Save(resultsDir, sr, weights)
			if err != nil {
				return fmt.Errorf("saving results: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Results saved to %s\n", outDir)
		}

		format, _ := cmd.Flags().GetString("format")
		return report.Write(os.Stdout, sr, format, weights)
	},
}

func rankingWeights(s *suite.Suite) report.Weights {
	if s.Ranking == nil {
		return report.DefaultWeights()
	}
	return report.Weights{
		Correctness: s.Ranking.Weights.Correctness,
		Cost:        s.Ranking.Weights.Cost,
		Duration:    s.Ranking.Weights.Duration,
	}
}

func defaultRegistry() *registry.Registry {
	reg := registry.New()
	reg.Register("claude-code", func(config map[string]any) (agentrunner.Runner, error) {
		return claudecode.NewRunner(claudecode.WithLogger(slog.Default())), nil
	})
	reg.Register("ollama", func(config map[string]any) (agentrunner.Runner, error) {
		return ollama.NewRunner(), nil
	})
	return reg
}

func init() {
	runCmd.Flags().Int("samples", 1, "Number of runs per treatment")
	runCmd.Flags().IntP("parallel", "p", 0, "Max concurrent samples (default: sequential)")
	runCmd.Flags().String("results-dir", "", "Directory for results output")
	runCmd.Flags().StringSlice("treatments", nil, "Filter to specific treatments")
	runCmd.Flags().StringSlice("evals", nil, "Filter to specific eval IDs")
	runCmd.Flags().String("format", "markdown", "Output format: markdown, json")
	runCmd.Flags().Int("timeout", 0, "Timeout in seconds for all evals (overrides suite/eval-level timeouts)")

	rootCmd.AddCommand(runCmd)
}
