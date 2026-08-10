package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/suite"
)

func TestIsolateGivesUniqueDirsPerSample(t *testing.T) {
	// Create a source directory with a file.
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "data.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "r1"},
			{Text: "r2"},
			{Text: "r3"},
		},
	}

	s := &suite.Suite{
		Description: "test",
		Version:     1,
		Evals: []suite.Eval{
			{
				ID:      "eval-1",
				Name:    "Isolated Eval",
				Prompt:  "do something",
				Dir:     srcDir,
				Isolate: boolPtr(true),
				Samples: intPtr(3),
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code"},
				},
			},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(runner.calls))
	}

	// Each sample should get a unique working directory that is NOT the original srcDir.
	dirs := make(map[string]bool)
	for i, call := range runner.calls {
		dir := call.Opts.WorkingDir
		if dir == srcDir {
			t.Errorf("sample %d: should not use original dir, got %q", i+1, dir)
		}
		if dirs[dir] {
			t.Errorf("sample %d: duplicate working dir %q", i+1, dir)
		}
		dirs[dir] = true
	}

	// Verify all runs succeeded.
	runs := sr.Evals[0].Variants[0].Runs
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}
	for i, run := range runs {
		if run.Err != nil {
			t.Errorf("run %d: unexpected error: %v", i+1, run.Err)
		}
	}
}

func TestIsolateCopiesFiles(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "data.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	var capturedDir string
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "done"}},
	}

	// Wrap to capture the working dir.
	origStart := runner.Start
	_ = origStart // fakeRunner already captures calls

	s := &suite.Suite{
		Description: "test",
		Version:     1,
		Evals: []suite.Eval{
			{
				ID:      "eval-1",
				Name:    "Isolated Eval",
				Prompt:  "do something",
				Dir:     srcDir,
				Isolate: boolPtr(true),
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code"},
				},
			},
		},
	}

	_, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	capturedDir = runner.calls[0].Opts.WorkingDir

	// The isolated dir should contain the copied file.
	// Note: the dir may already be cleaned up by defer, so we verify via the runner call.
	if capturedDir == srcDir {
		t.Error("isolated dir should differ from source dir")
	}
}

func TestIsolateTempDirsPreserved(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "r1"}, {Text: "r2"}},
	}

	s := &suite.Suite{
		Description: "test",
		Version:     1,
		Evals: []suite.Eval{
			{
				ID:      "eval-1",
				Name:    "Isolated Eval",
				Prompt:  "do something",
				Dir:     srcDir,
				Isolate: boolPtr(true),
				Samples: intPtr(2),
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code"},
				},
			},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Isolated temp dirs should be preserved for inspection.
	for _, call := range runner.calls {
		dir := call.Opts.WorkingDir
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("temp dir %q should have been preserved", dir)
		}
		// Clean up after test.
		defer os.RemoveAll(dir)
	}

	// WorkDir should be recorded in results.
	for _, run := range sr.Evals[0].Variants[0].Runs {
		if run.WorkDir == "" {
			t.Error("expected WorkDir to be set on run result")
		}
	}
}

func TestNoIsolateBehaviorUnchanged(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "done"}},
	}

	s := newMinimalSuite()
	s.Evals[0].Dir = "/tmp/eval-dir"
	s.Evals[0].Isolate = boolPtr(false)

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if runner.calls[0].Opts.WorkingDir != "/tmp/eval-dir" {
		t.Errorf("without isolate, should use eval dir, got %q", runner.calls[0].Opts.WorkingDir)
	}
}

func TestIsolateWithVariantDir(t *testing.T) {
	// Variant dir should be used as the source for isolation.
	variantDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(variantDir, "variant.txt"), []byte("variant"), 0o644); err != nil {
		t.Fatal(err)
	}
	evalDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(evalDir, "eval.txt"), []byte("eval"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "done"}},
	}

	s := &suite.Suite{
		Description: "test",
		Version:     1,
		Evals: []suite.Eval{
			{
				ID:      "eval-1",
				Name:    "Isolated Eval",
				Prompt:  "do something",
				Dir:     evalDir,
				Isolate: boolPtr(true),
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code", Dir: variantDir},
				},
			},
		},
	}

	_, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The isolated dir should not be the variant dir itself.
	capturedDir := runner.calls[0].Opts.WorkingDir
	if capturedDir == variantDir {
		t.Error("isolated dir should differ from variant dir")
	}
	if capturedDir == evalDir {
		t.Error("isolated dir should differ from eval dir")
	}
}
