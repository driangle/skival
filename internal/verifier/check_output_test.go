package verifier

import (
	"context"
	"testing"
	"time"
)

func TestCheckOutputVerifier_PassOnExitZero(t *testing.T) {
	v := &CheckOutputVerifier{Command: "exit 0", Dir: t.TempDir()}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "some output"})
	if !result.Pass {
		t.Fatalf("expected pass, got fail: %s", result.Reason)
	}
}

func TestCheckOutputVerifier_FailOnNonZeroExit(t *testing.T) {
	v := &CheckOutputVerifier{Command: "echo 'bad result' >&2; exit 1", Dir: t.TempDir()}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "some output"})
	if result.Pass {
		t.Fatal("expected fail on non-zero exit")
	}
	if result.Reason != "check_output failed: bad result" {
		t.Fatalf("unexpected reason: %s", result.Reason)
	}
}

func TestCheckOutputVerifier_CapturesStderrAsReason(t *testing.T) {
	v := &CheckOutputVerifier{Command: "echo 'detail about failure' >&2; exit 2", Dir: t.TempDir()}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail")
	}
	if result.Reason != "check_output failed: detail about failure" {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestCheckOutputVerifier_ReceivesRunOutputOnStdin(t *testing.T) {
	v := &CheckOutputVerifier{Command: "grep -q 'needle'", Dir: t.TempDir()}

	pass := v.Verify(context.Background(), VerifyInput{RunOutput: "hay needle hay"})
	if !pass.Pass {
		t.Fatalf("expected pass when stdin contains needle: %s", pass.Reason)
	}

	fail := v.Verify(context.Background(), VerifyInput{RunOutput: "just hay"})
	if fail.Pass {
		t.Fatal("expected fail when stdin does not contain needle")
	}
}

func TestCheckOutputVerifier_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	v := &CheckOutputVerifier{Command: "sleep 10", Dir: t.TempDir()}
	result := v.Verify(ctx, VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on timeout")
	}
	if result.Reason == "" {
		t.Fatal("expected a reason on timeout")
	}
}

func TestCheckOutputVerifier_FallsBackToErrorWhenNoStderr(t *testing.T) {
	v := &CheckOutputVerifier{Command: "exit 1", Dir: t.TempDir()}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail")
	}
	if result.Reason != "check_output failed: exit status 1" {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestCheckOutputVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &CheckOutputVerifier{}
}
