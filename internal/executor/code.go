package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ExecutionResult holds the outcome of running generated code.
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// runtime maps a language identifier to the command used to run it and the file extension.
type runtime struct {
	Cmd []string // e.g. ["go", "run"] or ["python3"]
	Ext string   // e.g. ".go" or ".py"
}

var runtimes = map[string]runtime{
	"go":         {Cmd: []string{"go", "run"}, Ext: ".go"},
	"python":     {Cmd: []string{"python3"}, Ext: ".py"},
	"python3":    {Cmd: []string{"python3"}, Ext: ".py"},
	"javascript": {Cmd: []string{"node"}, Ext: ".js"},
	"js":         {Cmd: []string{"node"}, Ext: ".js"},
	"node":       {Cmd: []string{"node"}, Ext: ".js"},
	"typescript": {Cmd: []string{"npx", "tsx"}, Ext: ".ts"},
	"ts":         {Cmd: []string{"npx", "tsx"}, Ext: ".ts"},
	"ruby":       {Cmd: []string{"ruby"}, Ext: ".rb"},
	"bash":       {Cmd: []string{"bash"}, Ext: ".sh"},
	"sh":         {Cmd: []string{"sh"}, Ext: ".sh"},
}

// codeBlockRe matches fenced code blocks with an optional language tag.
var codeBlockRe = regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")

// DefaultTimeout is applied when no timeout is configured.
const DefaultTimeout = 30 * time.Second

// ExtractCodeBlock returns the first fenced code block from text along with its language tag.
// Returns empty strings if no code block is found.
func ExtractCodeBlock(text string) (lang, code string) {
	m := codeBlockRe.FindStringSubmatch(text)
	if m == nil {
		return "", ""
	}
	return strings.TrimSpace(m[1]), m[2]
}

// ExecuteCode extracts a code block from output, writes it to a temp file,
// and runs it with the appropriate runtime. The language can be provided
// explicitly; if empty it is inferred from the code fence.
func ExecuteCode(ctx context.Context, output string, lang string, timeout time.Duration) ExecutionResult {
	fenceLang, code := ExtractCodeBlock(output)
	if code == "" {
		return ExecutionResult{Err: fmt.Errorf("no code block found in output")}
	}

	if lang == "" {
		lang = fenceLang
	}
	lang = strings.ToLower(lang)

	rt, ok := runtimes[lang]
	if !ok {
		return ExecutionResult{Err: fmt.Errorf("unsupported language: %q", lang)}
	}

	if timeout == 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "skival-exec-*")
	if err != nil {
		return ExecutionResult{Err: fmt.Errorf("creating temp dir: %w", err)}
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "main"+rt.Ext)
	if err := os.WriteFile(tmpFile, []byte(code), 0o600); err != nil {
		return ExecutionResult{Err: fmt.Errorf("writing temp file: %w", err)}
	}

	args := append(rt.Cmd[1:], tmpFile)
	cmd := exec.CommandContext(ctx, rt.Cmd[0], args...)
	cmd.Dir = tmpDir
	cmd.WaitDelay = 5 * time.Second

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	result := ExecutionResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if runErr != nil {
		if ctx.Err() != nil {
			result.Err = fmt.Errorf("execution timed out after %s", timeout)
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if result.Err == nil {
			result.Err = runErr
		}
	}

	return result
}
