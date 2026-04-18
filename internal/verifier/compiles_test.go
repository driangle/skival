package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompilesVerifier_GoSuccess(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func main() {}
`)
	v := &CompilesVerifier{Dir: dir}
	r := v.Verify(context.Background(), VerifyInput{})
	if !r.Pass {
		t.Fatalf("expected pass, got fail: %s", r.Reason)
	}
}

func TestCompilesVerifier_GoFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func main() {
	undefined()
}
`)
	v := &CompilesVerifier{Dir: dir}
	r := v.Verify(context.Background(), VerifyInput{})
	if r.Pass {
		t.Fatal("expected fail for code with undefined function")
	}
	if r.Reason == "" {
		t.Fatal("expected a reason")
	}
}

func TestCompilesVerifier_NoLanguageDetected(t *testing.T) {
	dir := t.TempDir()
	v := &CompilesVerifier{Dir: dir}
	r := v.Verify(context.Background(), VerifyInput{})
	if r.Pass {
		t.Fatal("expected fail when no language files found")
	}
	if r.Reason != "compiles: no recognized language files found in directory" {
		t.Fatalf("unexpected reason: %s", r.Reason)
	}
}

func TestCompilesVerifier_RespectsContextCancellation(t *testing.T) {
	dir := t.TempDir()
	// Write valid Go that would compile, but cancel immediately.
	writeFile(t, filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func main() {}
`)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	// Let the timeout fire.
	time.Sleep(5 * time.Millisecond)

	v := &CompilesVerifier{Dir: dir}
	r := v.Verify(ctx, VerifyInput{})
	if r.Pass {
		t.Fatal("expected fail on cancelled context")
	}
}

func TestCompilesVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &CompilesVerifier{}
}

func TestDetectBuildCommand_GoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test\n")

	lang, cmd, args := detectBuildCommand(dir)
	if lang != "go" || cmd != "go" {
		t.Fatalf("expected go, got lang=%q cmd=%q", lang, cmd)
	}
	if len(args) != 2 || args[0] != "build" || args[1] != "./..." {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestDetectBuildCommand_GoFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n")

	lang, cmd, _ := detectBuildCommand(dir)
	if lang != "go" || cmd != "go" {
		t.Fatalf("expected go, got lang=%q cmd=%q", lang, cmd)
	}
}

func TestDetectBuildCommand_Cargo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Cargo.toml"), "[package]\n")

	lang, cmd, args := detectBuildCommand(dir)
	if lang != "rust" || cmd != "cargo" {
		t.Fatalf("expected rust/cargo, got lang=%q cmd=%q", lang, cmd)
	}
	if len(args) != 1 || args[0] != "check" {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestDetectBuildCommand_TypeScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tsconfig.json"), "{}\n")

	lang, cmd, args := detectBuildCommand(dir)
	if lang != "typescript" || cmd != "npx" {
		t.Fatalf("expected typescript/npx, got lang=%q cmd=%q", lang, cmd)
	}
	if len(args) != 2 || args[0] != "tsc" || args[1] != "--noEmit" {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestDetectBuildCommand_Empty(t *testing.T) {
	dir := t.TempDir()
	lang, cmd, _ := detectBuildCommand(dir)
	if lang != "" || cmd != "" {
		t.Fatalf("expected empty, got lang=%q cmd=%q", lang, cmd)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
