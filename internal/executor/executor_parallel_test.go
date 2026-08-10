package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/registry"
	"github.com/driangle/skival/internal/suite"
)

// concurrentRunner is a thread-safe fake runner for parallel tests.
type concurrentRunner struct {
	mu       sync.Mutex
	calls    []fakeCall
	result   *agentrunner.Result // same result for every call
	delay    time.Duration       // artificial delay per call
	inflight atomic.Int32        // tracks concurrent in-flight calls
	maxSeen  atomic.Int32        // peak concurrency observed
}

func (r *concurrentRunner) Start(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Session, error) {
	var o agentrunner.Options
	for _, opt := range opts {
		opt(&o)
	}
	r.mu.Lock()
	r.calls = append(r.calls, fakeCall{Prompt: prompt, Opts: o})
	r.mu.Unlock()

	cur := r.inflight.Add(1)
	for {
		old := r.maxSeen.Load()
		if cur <= old || r.maxSeen.CompareAndSwap(old, cur) {
			break
		}
	}

	res := r.result

	sctx, cancel := context.WithCancel(ctx)
	return agentrunner.NewSession(sctx, cancel, func(_ context.Context, ch chan<- agentrunner.Message) (*agentrunner.Result, error) {
		if r.delay > 0 {
			time.Sleep(r.delay)
		}
		r.inflight.Add(-1)
		return res, nil
	}), nil
}

func (r *concurrentRunner) Run(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	return r.result, nil
}

func fakeRegistryWith(name string, runner agentrunner.Runner) *registry.Registry {
	reg := registry.New()
	reg.Register(name, func(config map[string]any) (agentrunner.Runner, error) {
		return runner, nil
	})
	return reg
}

func TestParallelSamplesRunConcurrently(t *testing.T) {
	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok", CostUSD: 0.01, Duration: time.Second},
		delay:  50 * time.Millisecond,
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(4)
	s.Evals[0].Parallel = intPtr(4)

	start := time.Now()
	sr, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), nil)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Variants[0].Runs
	if len(runs) != 4 {
		t.Fatalf("expected 4 runs, got %d", len(runs))
	}

	// With 4 parallel workers and 50ms delay each, should finish much faster than 4*50ms=200ms.
	if elapsed >= 180*time.Millisecond {
		t.Errorf("expected parallel execution to be faster, took %v", elapsed)
	}

	// Verify actual concurrency was observed.
	if runner.maxSeen.Load() < 2 {
		t.Errorf("expected at least 2 concurrent samples, peak was %d", runner.maxSeen.Load())
	}
}

func TestParallelResultsOrderPreserved(t *testing.T) {
	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok"},
		delay:  10 * time.Millisecond,
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(8)
	s.Evals[0].Parallel = intPtr(4)

	sr, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Variants[0].Runs
	if len(runs) != 8 {
		t.Fatalf("expected 8 runs, got %d", len(runs))
	}
	for i, run := range runs {
		if run.Sample != i+1 {
			t.Errorf("run %d: expected sample %d, got %d", i, i+1, run.Sample)
		}
	}
}

func TestParallelDefaultSequential(t *testing.T) {
	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok"},
		delay:  10 * time.Millisecond,
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(3)
	// No Parallel set — should default to sequential.

	sr, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Variants[0].Runs
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}

	// Sequential: peak concurrency should be 1.
	if runner.maxSeen.Load() != 1 {
		t.Errorf("expected peak concurrency 1 (sequential), got %d", runner.maxSeen.Load())
	}
}

func TestParallelIsolationDirsIndependent(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "data.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok"},
		delay:  20 * time.Millisecond,
	}

	s := &suite.Suite{
		Description: "test",
		Version:     1,
		Evals: []suite.Eval{
			{
				ID:       "eval-1",
				Name:     "Isolated Parallel",
				Prompt:   "do something",
				Dir:      srcDir,
				Isolate:  boolPtr(true),
				Samples:  intPtr(4),
				Parallel: intPtr(4),
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code"},
				},
			},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Variants[0].Runs
	if len(runs) != 4 {
		t.Fatalf("expected 4 runs, got %d", len(runs))
	}

	runner.mu.Lock()
	calls := runner.calls
	runner.mu.Unlock()

	dirs := make(map[string]bool)
	for i, call := range calls {
		dir := call.Opts.WorkingDir
		if dir == srcDir {
			t.Errorf("sample %d: should not use original dir", i+1)
		}
		if dirs[dir] {
			t.Errorf("sample %d: duplicate working dir %q", i+1, dir)
		}
		dirs[dir] = true
	}
}

func TestParallelProgressNoRace(t *testing.T) {
	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok", CostUSD: 0.01, Duration: time.Second},
		delay:  10 * time.Millisecond,
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(8)
	s.Evals[0].Parallel = intPtr(4)

	var buf bytes.Buffer
	opts := &Options{
		Progress: &buf,
	}

	sr, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Variants[0].Runs
	if len(runs) != 8 {
		t.Fatalf("expected 8 runs, got %d", len(runs))
	}

	// Verify progress output was written (no panics, no races — tested with -race flag).
	if buf.Len() == 0 {
		t.Error("expected progress output, got none")
	}
}

func TestTimeoutOverrideFromOptions(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Timeout = intPtr(30) // eval says 30s

	// CLI overrides to 120s.
	_, _ = Execute(context.Background(), s, fakeRegistry(runner), &Options{Timeout: 120})

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	if runner.calls[0].Opts.Timeout != 120*time.Second {
		t.Errorf("expected timeout 120s (CLI override), got %v", runner.calls[0].Opts.Timeout)
	}
}

func TestTimeoutOverrideWithoutEvalTimeout(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	// No eval-level timeout set.

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), &Options{Timeout: 60})

	if runner.calls[0].Opts.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s (CLI override), got %v", runner.calls[0].Opts.Timeout)
	}
}

func TestNoTimeoutOverrideUsesEval(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Timeout = intPtr(45)

	// No CLI timeout override (Timeout: 0).
	_, _ = Execute(context.Background(), s, fakeRegistry(runner), &Options{})

	if runner.calls[0].Opts.Timeout != 45*time.Second {
		t.Errorf("expected timeout 45s (eval-level), got %v", runner.calls[0].Opts.Timeout)
	}
}

func TestNoTimeoutAnywhere(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	// No timeout set at any level.

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), &Options{})

	if runner.calls[0].Opts.Timeout != 0 {
		t.Errorf("expected no timeout (0), got %v", runner.calls[0].Opts.Timeout)
	}
}

func TestParallelOptionOverridesSuiteDefault(t *testing.T) {
	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok"},
		delay:  50 * time.Millisecond,
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(4)
	// Eval says sequential.
	s.Evals[0].Parallel = intPtr(1)

	// CLI overrides to parallel.
	opts := &Options{Parallel: 4}

	start := time.Now()
	sr, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), opts)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Variants[0].Runs
	if len(runs) != 4 {
		t.Fatalf("expected 4 runs, got %d", len(runs))
	}

	// Should have run in parallel despite eval setting sequential.
	if elapsed >= 180*time.Millisecond {
		t.Errorf("expected parallel execution (CLI override), took %v", elapsed)
	}
}

func TestParallelVariantsRunConcurrently(t *testing.T) {
	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok", CostUSD: 0.01, Duration: time.Second},
		delay:  50 * time.Millisecond,
	}

	s := &suite.Suite{
		Version: 1,
		Evals: []suite.Eval{
			{
				ID:     "eval-1",
				Name:   "Test Eval",
				Prompt: "do something",
				Variants: []suite.Variant{
					{Name: "v1", Runner: "claude-code", Model: "claude-sonnet-4-6"},
					{Name: "v2", Runner: "claude-code", Model: "claude-opus-4-6"},
					{Name: "v3", Runner: "claude-code", Model: "claude-sonnet-4-6"},
					{Name: "v4", Runner: "claude-code", Model: "claude-opus-4-6"},
				},
			},
		},
	}

	start := time.Now()
	sr, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), &Options{ParallelVariants: 4})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sr.Evals[0].Variants) != 4 {
		t.Fatalf("expected 4 variants, got %d", len(sr.Evals[0].Variants))
	}

	// With 4 parallel variants and 50ms delay each, should finish much faster than 4*50ms=200ms.
	if elapsed >= 180*time.Millisecond {
		t.Errorf("expected parallel variant execution to be faster, took %v", elapsed)
	}

	// Verify actual concurrency was observed.
	if runner.maxSeen.Load() < 2 {
		t.Errorf("expected at least 2 concurrent variants, peak was %d", runner.maxSeen.Load())
	}
}

func TestParallelVariantsOrderPreserved(t *testing.T) {
	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok"},
		delay:  10 * time.Millisecond,
	}

	s := &suite.Suite{
		Version: 1,
		Evals: []suite.Eval{
			{
				ID:     "eval-1",
				Name:   "Test Eval",
				Prompt: "do something",
				Variants: []suite.Variant{
					{Name: "v1", Runner: "claude-code", Model: "claude-sonnet-4-6"},
					{Name: "v2", Runner: "claude-code", Model: "claude-sonnet-4-6"},
					{Name: "v3", Runner: "claude-code", Model: "claude-sonnet-4-6"},
				},
			},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), &Options{ParallelVariants: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	variants := sr.Evals[0].Variants
	if len(variants) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(variants))
	}
	for i, tr := range variants {
		expected := fmt.Sprintf("v%d", i+1)
		if tr.Name != expected {
			t.Errorf("variant[%d] name = %q, want %q", i, tr.Name, expected)
		}
	}
}

func TestParallelVariantsDefaultSequential(t *testing.T) {
	runner := &concurrentRunner{
		result: &agentrunner.Result{Text: "ok"},
		delay:  10 * time.Millisecond,
	}

	s := &suite.Suite{
		Version: 1,
		Evals: []suite.Eval{
			{
				ID:     "eval-1",
				Name:   "Test Eval",
				Prompt: "do something",
				Variants: []suite.Variant{
					{Name: "v1", Runner: "claude-code", Model: "claude-sonnet-4-6"},
					{Name: "v2", Runner: "claude-code", Model: "claude-sonnet-4-6"},
				},
			},
		},
	}

	_, err := Execute(context.Background(), s, fakeRegistryWith("claude-code", runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sequential: peak concurrency should be 1.
	if runner.maxSeen.Load() != 1 {
		t.Errorf("expected peak concurrency 1 (sequential), got %d", runner.maxSeen.Load())
	}
}
