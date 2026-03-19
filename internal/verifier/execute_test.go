package verifier

import (
	"context"
	"testing"
)

func TestExecuteVerifier_ExitZero(t *testing.T) {
	v := &ExecuteVerifier{}
	r := v.Verify(context.Background(), VerifyInput{ExitCode: 0})
	if !r.Pass {
		t.Error("expected pass for exit code 0")
	}
}

func TestExecuteVerifier_NonZeroExit(t *testing.T) {
	v := &ExecuteVerifier{}
	r := v.Verify(context.Background(), VerifyInput{ExitCode: 1})
	if r.Pass {
		t.Error("expected fail for exit code 1")
	}
	if r.Reason == "" {
		t.Error("expected a reason")
	}
}
