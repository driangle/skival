package result

import (
	"encoding/json"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
)

// RunResult captures the outcome of a single sample run.
type RunResult struct {
	Sample     int
	Text       string
	IsError    bool
	ExitCode   int
	CostUSD    float64
	DurationMs int64
	Usage      agentrunner.Usage
	SessionID  string
	Err                error
	Pass               *bool
	Conversation       []json.RawMessage
	JudgeConversation  []json.RawMessage
}

// TreatmentResult groups runs for one treatment.
type TreatmentResult struct {
	Name      string `json:"name"`
	Runner    string `json:"runner,omitempty"`
	IsControl bool   `json:"is_control"`
	Runs      []RunResult
	Aggregate *Aggregate
}

// EvalResult groups treatments for one eval.
type EvalResult struct {
	EvalID     string
	EvalName   string
	Treatments []TreatmentResult
	Err        error
}

// SuiteResult is the top-level result for an entire suite execution.
type SuiteResult struct {
	Description string
	StartedAt   time.Time
	FinishedAt  time.Time
	Evals       []EvalResult
}
