package executor

import "io"

// Options configures execution behavior.
type Options struct {
	// EvalIDs filters to only these eval IDs. Empty means run all.
	EvalIDs []string
	// Variants filters to only these variant names. Empty means run all.
	Variants []string
	// Progress receives live progress updates. Nil disables progress.
	Progress io.Writer
	// Samples overrides the per-eval sample count when set (> 0).
	Samples int
	// Parallel sets the max number of concurrent samples. 0 or 1 = sequential.
	Parallel int
	// ParallelVariants sets the max number of concurrent variants per eval.
	// 0 or 1 = sequential. When > 1, the reset hook is skipped.
	ParallelVariants int
	// Timeout overrides the per-eval timeout (in seconds) when set (> 0).
	Timeout int
}
