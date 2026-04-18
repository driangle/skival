package executor

import (
	"math"
	"math/rand"
	"time"

	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

// retryConfig holds resolved retry settings with defaults applied.
type retryConfig struct {
	maxAttempts int
	backoff     string
	delay       time.Duration
	on          string
}

func resolveRetryConfig(r *suite.Retry) retryConfig {
	cfg := retryConfig{
		maxAttempts: 1,
		backoff:     "fixed",
		delay:       2 * time.Second,
		on:          "transient",
	}
	if r == nil {
		return cfg
	}
	if r.MaxAttempts != nil {
		cfg.maxAttempts = *r.MaxAttempts
	}
	if r.Backoff != "" {
		cfg.backoff = r.Backoff
	}
	if r.Delay != "" {
		if d, err := time.ParseDuration(r.Delay); err == nil {
			cfg.delay = d
		}
	}
	if r.On != "" {
		cfg.on = r.On
	}
	return cfg
}

// shouldRetry returns true if the run outcome warrants another attempt.
func shouldRetry(run *result.RunResult, cfg retryConfig) bool {
	if run.Err != nil {
		// Execution error (runner crash, timeout, network) — always transient.
		return true
	}
	if cfg.on == "all" {
		// Retry on any non-pass: correctness failure or unverified.
		return run.Pass != nil && !*run.Pass
	}
	// on: transient — don't retry correctness failures.
	return false
}

// isBetterResult returns true if candidate is a better result than current.
// Preference: pass > fail > error. On tie, prefer lower cost.
func isBetterResult(candidate, current result.RunResult) bool {
	cs := resultScore(candidate)
	os := resultScore(current)
	if cs != os {
		return cs > os
	}
	return candidate.CostUSD < current.CostUSD
}

func resultScore(r result.RunResult) int {
	if r.Err != nil {
		return 0 // error
	}
	if r.Pass == nil {
		return 1 // unverified
	}
	if !*r.Pass {
		return 1 // fail (same tier as unverified)
	}
	return 2 // pass
}

// backoffDelay computes the delay before the next retry attempt.
// attempt is 1-indexed (delay before attempt 2 uses attempt=1).
func backoffDelay(attempt int, cfg retryConfig) time.Duration {
	base := cfg.delay
	if cfg.backoff == "exponential" {
		base = time.Duration(float64(cfg.delay) * math.Pow(2, float64(attempt-1)))
	}
	// Add jitter: +-25% of the base delay.
	jitter := float64(base) * 0.25 * (rand.Float64()*2 - 1)
	return base + time.Duration(jitter)
}
