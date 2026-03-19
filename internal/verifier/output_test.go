package verifier

import (
	"context"
	"testing"
)

func TestOutputVerifier_AllSubstringsPresent(t *testing.T) {
	v := &OutputVerifier{
		ExpectedSubstrings: []string{"hello", "world"},
	}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "hello beautiful world"})
	if !result.Pass {
		t.Fatalf("expected pass, got fail: %s", result.Reason)
	}
}

func TestOutputVerifier_PartialMatch(t *testing.T) {
	v := &OutputVerifier{
		ExpectedSubstrings: []string{"hello", "missing"},
	}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "hello world"})
	if result.Pass {
		t.Fatal("expected fail for partial match")
	}
	want := `expected substring not found in output: "missing"`
	if result.Reason != want {
		t.Fatalf("reason = %q, want %q", result.Reason, want)
	}
}

func TestOutputVerifier_NoMatch(t *testing.T) {
	v := &OutputVerifier{
		ExpectedSubstrings: []string{"foo", "bar"},
	}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "completely different output"})
	if result.Pass {
		t.Fatal("expected fail when no substrings match")
	}
	want := `expected substring not found in output: "foo"`
	if result.Reason != want {
		t.Fatalf("reason = %q, want %q", result.Reason, want)
	}
}

func TestOutputVerifier_EmptyExpectedOutput(t *testing.T) {
	v := &OutputVerifier{
		ExpectedSubstrings: []string{},
	}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "anything here"})
	if !result.Pass {
		t.Fatalf("expected pass for empty expected_output, got fail: %s", result.Reason)
	}
}

func TestOutputVerifier_NilExpectedOutput(t *testing.T) {
	v := &OutputVerifier{}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: "anything here"})
	if !result.Pass {
		t.Fatalf("expected pass for nil expected_output, got fail: %s", result.Reason)
	}
}

func TestOutputVerifier_EmptyRunOutput(t *testing.T) {
	v := &OutputVerifier{
		ExpectedSubstrings: []string{"something"},
	}
	result := v.Verify(context.Background(), VerifyInput{RunOutput: ""})
	if result.Pass {
		t.Fatal("expected fail when output is empty but substrings expected")
	}
}

func TestOutputVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &OutputVerifier{}
}
