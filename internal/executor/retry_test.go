package executor

import (
	"errors"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

func TestResolveRetryConfig_Defaults(t *testing.T) {
	cfg := resolveRetryConfig(nil)
	if cfg.maxAttempts != 1 {
		t.Errorf("expected maxAttempts=1, got %d", cfg.maxAttempts)
	}
	if cfg.backoff != "fixed" {
		t.Errorf("expected backoff=fixed, got %q", cfg.backoff)
	}
	if cfg.delay != 2*time.Second {
		t.Errorf("expected delay=2s, got %v", cfg.delay)
	}
	if cfg.on != "transient" {
		t.Errorf("expected on=transient, got %q", cfg.on)
	}
}

func TestResolveRetryConfig_Override(t *testing.T) {
	attempts := 3
	r := &suite.Retry{
		MaxAttempts: &attempts,
		Backoff:     "exponential",
		Delay:       "500ms",
		On:          "all",
	}
	cfg := resolveRetryConfig(r)
	if cfg.maxAttempts != 3 {
		t.Errorf("expected maxAttempts=3, got %d", cfg.maxAttempts)
	}
	if cfg.backoff != "exponential" {
		t.Errorf("expected backoff=exponential, got %q", cfg.backoff)
	}
	if cfg.delay != 500*time.Millisecond {
		t.Errorf("expected delay=500ms, got %v", cfg.delay)
	}
	if cfg.on != "all" {
		t.Errorf("expected on=all, got %q", cfg.on)
	}
}

func TestShouldRetry_TransientMode(t *testing.T) {
	cfg := retryConfig{on: "transient"}

	// Execution error → retry.
	run := &result.RunResult{Err: errors.New("timeout")}
	if !shouldRetry(run, cfg) {
		t.Error("expected retry on execution error in transient mode")
	}

	// Correctness failure → no retry.
	f := false
	run = &result.RunResult{Pass: &f}
	if shouldRetry(run, cfg) {
		t.Error("expected no retry on correctness failure in transient mode")
	}

	// Pass → no retry.
	tr := true
	run = &result.RunResult{Pass: &tr}
	if shouldRetry(run, cfg) {
		t.Error("expected no retry on pass")
	}
}

func TestShouldRetry_AllMode(t *testing.T) {
	cfg := retryConfig{on: "all"}

	// Execution error → retry.
	run := &result.RunResult{Err: errors.New("crash")}
	if !shouldRetry(run, cfg) {
		t.Error("expected retry on execution error in all mode")
	}

	// Correctness failure → retry.
	f := false
	run = &result.RunResult{Pass: &f}
	if shouldRetry(run, cfg) != true {
		t.Error("expected retry on correctness failure in all mode")
	}

	// Pass → no retry.
	tr := true
	run = &result.RunResult{Pass: &tr}
	if shouldRetry(run, cfg) {
		t.Error("expected no retry on pass in all mode")
	}
}

func TestIsBetterResult(t *testing.T) {
	pass := true
	fail := false

	errRun := result.RunResult{Err: errors.New("err"), CostUSD: 0.01}
	failRun := result.RunResult{Pass: &fail, CostUSD: 0.05}
	passRun := result.RunResult{Pass: &pass, CostUSD: 0.10}
	cheapPass := result.RunResult{Pass: &pass, CostUSD: 0.03}

	if !isBetterResult(passRun, failRun) {
		t.Error("pass should be better than fail")
	}
	if !isBetterResult(failRun, errRun) {
		t.Error("fail should be better than error")
	}
	if isBetterResult(errRun, passRun) {
		t.Error("error should not be better than pass")
	}
	if !isBetterResult(cheapPass, passRun) {
		t.Error("cheaper pass should be better than expensive pass")
	}
}

func TestBackoffDelay_Fixed(t *testing.T) {
	cfg := retryConfig{backoff: "fixed", delay: 100 * time.Millisecond}

	for attempt := 1; attempt <= 3; attempt++ {
		d := backoffDelay(attempt, cfg)
		// Fixed: base +-25% jitter → between 75ms and 125ms.
		if d < 75*time.Millisecond || d > 125*time.Millisecond {
			t.Errorf("attempt %d: expected delay ~100ms, got %v", attempt, d)
		}
	}
}

func TestBackoffDelay_Exponential(t *testing.T) {
	cfg := retryConfig{backoff: "exponential", delay: 100 * time.Millisecond}

	// attempt 1: base=100ms, attempt 2: base=200ms, attempt 3: base=400ms
	for attempt, expectedBase := range map[int]time.Duration{1: 100, 2: 200, 3: 400} {
		base := expectedBase * time.Millisecond
		d := backoffDelay(attempt, cfg)
		low := time.Duration(float64(base) * 0.75)
		high := time.Duration(float64(base) * 1.25)
		if d < low || d > high {
			t.Errorf("attempt %d: expected delay ~%v (range %v-%v), got %v", attempt, base, low, high, d)
		}
	}
}
