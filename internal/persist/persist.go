package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/driangle/skival/internal/report"
	"github.com/driangle/skival/internal/result"
)

// SaveOptions configures optional Save behavior.
type SaveOptions struct {
	// LinkSessions, when true, renders a static vibeview session page per run
	// (next to its transcript sidecar) and records its path on the run result.
	LinkSessions bool
}

// Save writes all result data to a timestamped directory under baseDir.
// Returns the created directory path.
func Save(baseDir string, sr *result.SuiteResult, weights report.Weights, opts SaveOptions) (string, error) {
	timestamp := sr.StartedAt.Format("20060102-150405")
	dir := filepath.Join(baseDir, timestamp)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating results dir: %w", err)
	}

	// Record where artifacts live so the summary/report can point readers here
	// instead of listing every per-sample path.
	sr.ResultsDir = dir

	// Sidecars first: vibeview reads them, so they must exist on disk before
	// linkSessions runs. Run metadata (run-N.json) is written afterward so it
	// captures any SessionPage that linking sets on the in-memory result.
	if err := writeEvals(dir, sr); err != nil {
		return "", err
	}

	if opts.LinkSessions {
		linkSessions(dir, sr)
	}

	if err := writeRunMetas(dir, sr); err != nil {
		return "", err
	}

	if err := writeSummary(dir, sr, weights); err != nil {
		return "", err
	}

	return dir, nil
}

func writeEvals(dir string, sr *result.SuiteResult) error {
	evalsDir := filepath.Join(dir, "evals")

	for _, eval := range sr.Evals {
		if eval.Comparison != nil {
			evalDir := filepath.Join(evalsDir, eval.EvalID)
			if err := os.MkdirAll(evalDir, 0o755); err != nil {
				return fmt.Errorf("creating eval dir: %w", err)
			}
			if err := writeAtomicJSON(filepath.Join(evalDir, "comparison.json"), eval.Comparison); err != nil {
				return fmt.Errorf("writing comparison.json: %w", err)
			}
			if len(eval.Comparison.Conversation) > 0 {
				convPath := filepath.Join(evalDir, "comparison.judge.jsonl")
				if err := writeConversationJSONL(convPath, eval.Comparison.Conversation); err != nil {
					return fmt.Errorf("writing comparison judge conversation: %w", err)
				}
			}
		}
		for _, variant := range eval.Variants {
			variantDir := filepath.Join(evalsDir, eval.EvalID, variant.Name)
			if err := os.MkdirAll(variantDir, 0o755); err != nil {
				return fmt.Errorf("creating variant dir: %w", err)
			}

			for _, run := range variant.Runs {
				if err := writeRunSidecars(variantDir, run); err != nil {
					return err
				}
			}

			if variant.Aggregate != nil {
				if err := writeAtomicJSON(filepath.Join(variantDir, "aggregate.json"), variant.Aggregate); err != nil {
					return fmt.Errorf("writing aggregate.json: %w", err)
				}
			}
		}
	}

	return nil
}

// writeRunMetas writes each run's run-N.json. It runs after writeEvals (which
// writes transcript sidecars) and after optional session linking, so each
// run-N.json captures the SessionPage set during linking.
func writeRunMetas(dir string, sr *result.SuiteResult) error {
	evalsDir := filepath.Join(dir, "evals")
	for _, eval := range sr.Evals {
		for _, variant := range eval.Variants {
			variantDir := filepath.Join(evalsDir, eval.EvalID, variant.Name)
			for _, run := range variant.Runs {
				if err := writeRunMeta(variantDir, run); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeSummary(dir string, sr *result.SuiteResult, weights report.Weights) error {
	// summary.json
	if err := writeAtomicJSON(filepath.Join(dir, "summary.json"), buildSummaryJSON(sr, weights)); err != nil {
		return fmt.Errorf("writing summary.json: %w", err)
	}

	// summary.md
	summaryPath := filepath.Join(dir, "summary.md")
	f, err := os.CreateTemp(filepath.Dir(summaryPath), ".summary-*.md")
	if err != nil {
		return fmt.Errorf("creating temp file for summary.md: %w", err)
	}
	report.WriteMarkdown(f, sr, weights)
	f.Close()
	if err := os.Rename(f.Name(), summaryPath); err != nil {
		os.Remove(f.Name())
		return fmt.Errorf("writing summary.md: %w", err)
	}

	return nil
}

type summaryJSON struct {
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description"`
	StartedAt   string               `json:"started_at"`
	FinishedAt  string               `json:"finished_at"`
	Rankings    []report.VariantRank `json:"rankings,omitempty"`
}

func buildSummaryJSON(sr *result.SuiteResult, weights report.Weights) summaryJSON {
	return summaryJSON{
		Title:       sr.Title,
		Description: sr.Description,
		StartedAt:   sr.StartedAt.Format(time.RFC3339),
		FinishedAt:  sr.FinishedAt.Format(time.RFC3339),
		Rankings:    report.RankVariants(sr, weights),
	}
}

// writeAtomicJSON writes data as JSON to path via a temp file + rename.
func writeAtomicJSON(path string, data any) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return fmt.Errorf("encoding JSON: %w", err)
	}
	f.Close()

	if err := os.Rename(f.Name(), path); err != nil {
		os.Remove(f.Name())
		return err
	}
	return nil
}
