package verifier

import (
	"context"
	"testing"
	"time"

	"github.com/driangle/skival/internal/suite"
)

func TestCommandProbeVerifier_ExitCodeMatch(t *testing.T) {
	v := &CommandProbeVerifier{
		Probe: suite.CommandProbe{
			Run:    "exit 0",
			Assert: suite.CommandProbeAssert{Exits: intPtr(0)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass: %s", result.Reason)
	}
}

func TestCommandProbeVerifier_ExitCodeMismatch(t *testing.T) {
	v := &CommandProbeVerifier{
		Probe: suite.CommandProbe{
			Run:    "exit 1",
			Assert: suite.CommandProbeAssert{Exits: intPtr(0)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on wrong exit code")
	}
}

func TestCommandProbeVerifier_StdoutContainsMatch(t *testing.T) {
	v := &CommandProbeVerifier{
		Probe: suite.CommandProbe{
			Run:    "echo 'hello world'",
			Assert: suite.CommandProbeAssert{StdoutContains: "hello"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass: %s", result.Reason)
	}
}

func TestCommandProbeVerifier_StdoutContainsMismatch(t *testing.T) {
	v := &CommandProbeVerifier{
		Probe: suite.CommandProbe{
			Run:    "echo 'goodbye'",
			Assert: suite.CommandProbeAssert{StdoutContains: "hello"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail when stdout_contains not found")
	}
}

func TestCommandProbeVerifier_BothAssertions(t *testing.T) {
	v := &CommandProbeVerifier{
		Probe: suite.CommandProbe{
			Run: "echo 'accepting connections'",
			Assert: suite.CommandProbeAssert{
				Exits:          intPtr(0),
				StdoutContains: "accepting connections",
			},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass: %s", result.Reason)
	}
}

func TestCommandProbeVerifier_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	v := &CommandProbeVerifier{
		Probe: suite.CommandProbe{
			Run:    "sleep 10",
			Assert: suite.CommandProbeAssert{Exits: intPtr(0)},
		},
	}
	result := v.Verify(ctx, VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on timeout")
	}
}

func TestCommandProbeVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &CommandProbeVerifier{}
}
