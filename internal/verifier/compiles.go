package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CompilesVerifier checks that the code in a directory compiles successfully.
// It auto-detects the language from project files and runs the appropriate build command.
type CompilesVerifier struct {
	Dir string
}

func (v *CompilesVerifier) Verify(ctx context.Context, _ VerifyInput) VerifyResult {
	lang, cmd, args := detectBuildCommand(v.Dir)
	if cmd == "" {
		return VerifyResult{
			Pass:   false,
			Reason: "compiles: no recognized language files found in directory",
		}
	}

	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = v.Dir

	var stderr bytes.Buffer
	c.Stderr = &stderr

	err := c.Run()
	if err == nil {
		return VerifyResult{Pass: true, Reason: fmt.Sprintf("compiles: %s build succeeded", lang)}
	}

	if ctx.Err() != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("compiles: build timed out: %v", ctx.Err()),
		}
	}

	reason := strings.TrimSpace(stderr.String())
	if reason == "" {
		reason = err.Error()
	}
	return VerifyResult{
		Pass:   false,
		Reason: fmt.Sprintf("compiles: %s build failed: %s", lang, reason),
	}
}

// detectBuildCommand returns the language name, command, and args for building code in dir.
func detectBuildCommand(dir string) (lang string, cmd string, args []string) {
	if fileExists(filepath.Join(dir, "go.mod")) {
		return "go", "go", []string{"build", "./..."}
	}
	if hasFilesWithExt(dir, ".go") {
		return "go", "go", []string{"build", "."}
	}
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return "rust", "cargo", []string{"check"}
	}
	if fileExists(filepath.Join(dir, "tsconfig.json")) {
		return "typescript", "npx", []string{"tsc", "--noEmit"}
	}
	return "", "", nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasFilesWithExt(dir string, ext string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ext {
			return true
		}
	}
	return false
}
