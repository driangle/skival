package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/driangle/skival/internal/suite"
)

func boolPtrVal(b bool) *bool { return &b }

func TestFileProbeVerifier_ExistsTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(path, []byte("hello"), 0644)

	v := &FileProbeVerifier{
		Probe: suite.FileProbe{
			Path:   path,
			Assert: suite.FileProbeAssert{Exists: boolPtrVal(true)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass: %s", result.Reason)
	}
}

func TestFileProbeVerifier_ExistsTrueButMissing(t *testing.T) {
	v := &FileProbeVerifier{
		Probe: suite.FileProbe{
			Path:   "/nonexistent/file.txt",
			Assert: suite.FileProbeAssert{Exists: boolPtrVal(true)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail when file does not exist")
	}
}

func TestFileProbeVerifier_ExistsFalse(t *testing.T) {
	v := &FileProbeVerifier{
		Probe: suite.FileProbe{
			Path:   "/nonexistent/file.txt",
			Assert: suite.FileProbeAssert{Exists: boolPtrVal(false)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass: %s", result.Reason)
	}
}

func TestFileProbeVerifier_ExistsFalseButPresent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(path, []byte("hello"), 0644)

	v := &FileProbeVerifier{
		Probe: suite.FileProbe{
			Path:   path,
			Assert: suite.FileProbeAssert{Exists: boolPtrVal(false)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail when file exists but expected not to")
	}
}

func TestFileProbeVerifier_ContainsMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(path, []byte("hello world"), 0644)

	v := &FileProbeVerifier{
		Probe: suite.FileProbe{
			Path:   path,
			Assert: suite.FileProbeAssert{Contains: "hello world"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass: %s", result.Reason)
	}
}

func TestFileProbeVerifier_ContainsMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(path, []byte("goodbye"), 0644)

	v := &FileProbeVerifier{
		Probe: suite.FileProbe{
			Path:   path,
			Assert: suite.FileProbeAssert{Contains: "hello"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail when contains not found")
	}
}

func TestFileProbeVerifier_ContainsOnMissingFile(t *testing.T) {
	v := &FileProbeVerifier{
		Probe: suite.FileProbe{
			Path:   "/nonexistent/file.txt",
			Assert: suite.FileProbeAssert{Contains: "hello"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail when file does not exist for contains check")
	}
}

func TestFileProbeVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &FileProbeVerifier{}
}
