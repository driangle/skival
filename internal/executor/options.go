package executor

import "io"

// Options configures execution behavior.
type Options struct {
	// EvalIDs filters to only these eval IDs. Empty means run all.
	EvalIDs []string
	// Treatments filters to only these treatment names. Empty means run all.
	Treatments []string
	// Progress receives live progress updates. Nil disables progress.
	Progress io.Writer
}
