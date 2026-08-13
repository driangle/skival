package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/agentrunner/go/claudecode"
	"github.com/driangle/agentrunner/go/ollama"
	"github.com/driangle/skival/internal/executor"
	"github.com/driangle/skival/internal/persist"
	"github.com/driangle/skival/internal/registry"
	"github.com/driangle/skival/internal/report"
	"github.com/driangle/skival/internal/result"
	execrunner "github.com/driangle/skival/internal/runners/exec"
	"github.com/driangle/skival/internal/suite"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <suite.yaml>",
	Short: "Run an eval suite",
	Long:  "Execute an eval suite definition against configured variants and collect results.",
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
		variants, _ := cmd.Flags().GetStringSlice("variants")
		samples, _ := cmd.Flags().GetInt("samples")
		parallel, _ := cmd.Flags().GetInt("parallel")
		parallelVariants, _ := cmd.Flags().GetInt("parallel-variants")
		timeout, _ := cmd.Flags().GetInt("timeout")
		slog.Debug("Filters", "evals", evalIDs, "variants", variants, "samples", samples, "parallel", parallel, "parallel-variants", parallelVariants, "timeout", timeout)

		if timeout < 0 {
			return fmt.Errorf("--timeout must be a positive number of seconds")
		}
		if timeout == 0 && cmd.Flags().Changed("timeout") {
			return fmt.Errorf("--timeout must be a positive number of seconds")
		}

		compareOverride, err := compareOverride(cmd)
		if err != nil {
			return err
		}

		execOpts := &executor.Options{
			EvalIDs:          evalIDs,
			Variants:         variants,
			Progress:         os.Stderr,
			Samples:          samples,
			Parallel:         parallel,
			ParallelVariants: parallelVariants,
			Timeout:          timeout,
			Compare:          compareOverride,
		}

		sr, err := executor.Execute(cmd.Context(), s, reg, execOpts)
		if err != nil {
			return fmt.Errorf("executing suite: %w", err)
		}

		weights := rankingWeights(s, compareOverride)

		format, _ := cmd.Flags().GetString("format")
		linkSessions, _ := cmd.Flags().GetBool("link-sessions")

		resultsDir, _ := cmd.Flags().GetString("results-dir")
		if resultsDir != "" {
			outDir, err := persist.Save(resultsDir, sr, weights, persist.SaveOptions{LinkSessions: linkSessions})
			if err != nil {
				return fmt.Errorf("saving results: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Results saved to %s\n", outDir)

			// HTML is a file, not terminal output: write report.html into the
			// results dir (so its relative session links resolve) and point the
			// user at it rather than dumping the whole document to stdout.
			if format == "html" {
				reportPath := filepath.Join(outDir, "report.html")
				if err := writeReportFile(reportPath, sr, weights); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "HTML report written to %s\n", reportPath)
				return nil
			}
		}

		return report.Write(os.Stdout, sr, format, weights)
	},
}

// writeReportFile writes an HTML report to path via atomic temp+rename.
func writeReportFile(path string, sr *result.SuiteResult, weights report.Weights) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".report-*.html")
	if err != nil {
		return fmt.Errorf("creating temp file for report: %w", err)
	}
	if err := report.Write(f, sr, "html", weights); err != nil {
		f.Close()
		os.Remove(f.Name())
		return fmt.Errorf("writing report.html: %w", err)
	}
	f.Close()
	if err := os.Rename(f.Name(), path); err != nil {
		os.Remove(f.Name())
		return fmt.Errorf("writing report.html: %w", err)
	}
	return nil
}

// compareOverride resolves the --compare/--no-compare flags into a tri-state
// pointer: nil (defer to suite config), true (force on), or false (force off).
func compareOverride(cmd *cobra.Command) (*bool, error) {
	on := cmd.Flags().Changed("compare")
	off := cmd.Flags().Changed("no-compare")
	if on && off {
		return nil, fmt.Errorf("--compare and --no-compare are mutually exclusive")
	}
	if on {
		v := true
		return &v, nil
	}
	if off {
		v := false
		return &v, nil
	}
	return nil, nil
}

// rankingWeights maps suite ranking config to report weights. When the suite
// sets explicit weights they are used verbatim. Otherwise, if comparative
// judging will run, a quality weight is carved out of the defaults (the base
// correctness/cost/duration weights are renormalized to make room); if not, the
// unchanged defaults preserve today's ranking behavior.
func rankingWeights(s *suite.Suite, override *bool) report.Weights {
	if s.Ranking != nil {
		return report.Weights{
			Correctness: s.Ranking.Weights.Correctness,
			Cost:        s.Ranking.Weights.Cost,
			Duration:    s.Ranking.Weights.Duration,
			Quality:     s.Ranking.Weights.Quality,
			Tokens:      s.Ranking.Weights.Tokens,
		}
	}
	w := report.DefaultWeights()
	if s.CompareActive(override) {
		return withQualityWeight(w, s.CompareWeight())
	}
	return w
}

// withQualityWeight allocates q to the quality weight and renormalizes the base
// correctness/cost/duration weights to sum to 1-q, keeping the total at 1.0.
func withQualityWeight(base report.Weights, q float64) report.Weights {
	scale := 1 - q
	return report.Weights{
		Correctness: base.Correctness * scale,
		Cost:        base.Cost * scale,
		Duration:    base.Duration * scale,
		Tokens:      base.Tokens * scale,
		Quality:     q,
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
	reg.Register("exec", func(config map[string]any) (agentrunner.Runner, error) {
		return execrunner.NewRunner(execrunner.WithLogger(slog.Default())), nil
	})
	return reg
}

func init() {
	runCmd.Flags().Int("samples", 0, "Number of runs per variant (overrides suite/eval-level samples)")
	runCmd.Flags().IntP("parallel", "p", 0, "Max concurrent samples (default: sequential)")
	runCmd.Flags().Int("parallel-variants", 0, "Max concurrent variants per eval (default: sequential, skips reset hook)")
	runCmd.Flags().String("results-dir", "", "Directory for results output")
	runCmd.Flags().StringSlice("variants", nil, "Filter to specific variants")
	runCmd.Flags().StringSlice("evals", nil, "Filter to specific eval IDs")
	runCmd.Flags().String("format", "markdown", "Output format: markdown, json, html")
	runCmd.Flags().Int("timeout", 0, "Timeout in seconds for all evals (overrides suite/eval-level timeouts)")
	runCmd.Flags().Bool("compare", false, "Force comparative judging on where criteria are configured")
	runCmd.Flags().Bool("no-compare", false, "Disable comparative judging even if configured in the suite")
	runCmd.Flags().Bool("link-sessions", false, "Render a static session page per run (via the embedded vibeview renderer) and link it from the HTML report (requires --results-dir)")

	rootCmd.AddCommand(runCmd)
}
