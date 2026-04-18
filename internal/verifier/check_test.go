package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckVerifier_GoSuccess(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func main() {}
`)
	v := &CheckVerifier{Dir: dir, Command: "go build ./..."}
	r := v.Verify(context.Background(), VerifyInput{})
	if !r.Pass {
		t.Fatalf("expected pass, got fail: %s", r.Reason)
	}
}

func TestCheckVerifier_GoFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func main() {
	undefined()
}
`)
	v := &CheckVerifier{Dir: dir, Command: "go build ./..."}
	r := v.Verify(context.Background(), VerifyInput{})
	if r.Pass {
		t.Fatal("expected fail for code with undefined function")
	}
	if r.Reason == "" {
		t.Fatal("expected a reason")
	}
}

func TestCheckVerifier_CustomCommand(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "hello.txt"), "hello\n")

	v := &CheckVerifier{Dir: dir, Command: "test -f hello.txt"}
	r := v.Verify(context.Background(), VerifyInput{})
	if !r.Pass {
		t.Fatalf("expected pass, got fail: %s", r.Reason)
	}
}

func TestCheckVerifier_CustomCommandFailure(t *testing.T) {
	dir := t.TempDir()

	v := &CheckVerifier{Dir: dir, Command: "test -f nonexistent.txt"}
	r := v.Verify(context.Background(), VerifyInput{})
	if r.Pass {
		t.Fatal("expected fail for missing file")
	}
}

func TestCheckVerifier_RespectsContextCancellation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func main() {}
`)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)

	v := &CheckVerifier{Dir: dir, Command: "go build ./..."}
	r := v.Verify(ctx, VerifyInput{})
	if r.Pass {
		t.Fatal("expected fail on cancelled context")
	}
}

func TestCheckVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &CheckVerifier{}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
