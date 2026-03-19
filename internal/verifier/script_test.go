package verifier

import (
	"context"
	"testing"
	"time"
)

func TestScriptVerifier_PassOnExitZero(t *testing.T) {
	v := &ScriptVerifier{Script: "exit 0", Dir: t.TempDir()}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "some output"})
	if !result.Pass {
		t.Fatalf("expected pass, got fail: %s", result.Reason)
	}
}

func TestScriptVerifier_FailOnNonZeroExit(t *testing.T) {
	v := &ScriptVerifier{Script: "echo 'bad result' >&2; exit 1", Dir: t.TempDir()}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "some output"})
	if result.Pass {
		t.Fatal("expected fail on non-zero exit")
	}
	if result.Reason != "script failed: bad result" {
		t.Fatalf("unexpected reason: %s", result.Reason)
	}
}

func TestScriptVerifier_CapturesStderrAsReason(t *testing.T) {
	v := &ScriptVerifier{Script: "echo 'detail about failure' >&2; exit 2", Dir: t.TempDir()}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail")
	}
	if result.Reason != "script failed: detail about failure" {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestScriptVerifier_ReceivesRunOutputOnStdin(t *testing.T) {
	v := &ScriptVerifier{Script: "grep -q 'needle'", Dir: t.TempDir()}

	pass := v.Verify(context.Background(), VerifyInput{RunOutput: "hay needle hay"})
	if !pass.Pass {
		t.Fatalf("expected pass when stdin contains needle: %s", pass.Reason)
	}

	fail := v.Verify(context.Background(), VerifyInput{RunOutput: "just hay"})
	if fail.Pass {
		t.Fatal("expected fail when stdin does not contain needle")
	}
}

func TestScriptVerifier_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	v := &ScriptVerifier{Script: "sleep 10", Dir: t.TempDir()}
	result := v.Verify(ctx, VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on timeout")
	}
	if result.Reason == "" {
		t.Fatal("expected a reason on timeout")
	}
}

func TestScriptVerifier_FallsBackToErrorWhenNoStderr(t *testing.T) {
	v := &ScriptVerifier{Script: "exit 1", Dir: t.TempDir()}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail")
	}
	if result.Reason != "script failed: exit status 1" {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestScriptVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &ScriptVerifier{}
}
