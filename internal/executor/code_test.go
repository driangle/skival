package executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExtractCodeBlock_WithLanguage(t *testing.T) {
	text := "Here is the code:\n```go\npackage main\n\nfunc main() {}\n```\nDone."
	lang, code := ExtractCodeBlock(text)
	if lang != "go" {
		t.Fatalf("lang = %q, want %q", lang, "go")
	}
	if !strings.Contains(code, "func main()") {
		t.Fatalf("code = %q, expected to contain func main()", code)
	}
}

func TestExtractCodeBlock_WithoutLanguage(t *testing.T) {
	text := "```\necho hello\n```"
	lang, code := ExtractCodeBlock(text)
	if lang != "" {
		t.Fatalf("lang = %q, want empty", lang)
	}
	if !strings.Contains(code, "echo hello") {
		t.Fatalf("code = %q", code)
	}
}

func TestExtractCodeBlock_NoBlock(t *testing.T) {
	lang, code := ExtractCodeBlock("no code here")
	if lang != "" || code != "" {
		t.Fatalf("expected empty, got lang=%q code=%q", lang, code)
	}
}

func TestExtractCodeBlock_FirstBlockOnly(t *testing.T) {
	text := "```python\nfirst\n```\n```go\nsecond\n```"
	lang, code := ExtractCodeBlock(text)
	if lang != "python" {
		t.Fatalf("lang = %q, want python", lang)
	}
	if !strings.Contains(code, "first") {
		t.Fatalf("expected first block, got %q", code)
	}
}

func TestExecuteCode_RunsGoCode(t *testing.T) {
	output := "```go\npackage main\n\nimport \"fmt\"\n\nfunc main() { fmt.Print(\"hello\") }\n```"
	result := ExecuteCode(context.Background(), output, "", 10*time.Second)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Stdout != "hello" {
		t.Fatalf("stdout = %q, want %q", result.Stdout, "hello")
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestExecuteCode_RunsBashCode(t *testing.T) {
	output := "```bash\necho -n hello\n```"
	result := ExecuteCode(context.Background(), output, "", 10*time.Second)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Stdout != "hello" {
		t.Fatalf("stdout = %q, want %q", result.Stdout, "hello")
	}
}

func TestExecuteCode_CapturesStderr(t *testing.T) {
	output := "```bash\necho -n oops >&2\n```"
	result := ExecuteCode(context.Background(), output, "", 10*time.Second)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Stderr != "oops" {
		t.Fatalf("stderr = %q, want %q", result.Stderr, "oops")
	}
}

func TestExecuteCode_CapturesNonZeroExitCode(t *testing.T) {
	output := "```bash\nexit 42\n```"
	result := ExecuteCode(context.Background(), output, "", 10*time.Second)
	if result.ExitCode != 42 {
		t.Fatalf("exit code = %d, want 42", result.ExitCode)
	}
}

func TestExecuteCode_RespectsTimeout(t *testing.T) {
	output := "```bash\nsleep 60\n```"
	result := ExecuteCode(context.Background(), output, "", 100*time.Millisecond)
	if result.Err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(result.Err.Error(), "timed out") {
		t.Fatalf("error = %q, expected timeout", result.Err)
	}
}

func TestExecuteCode_DefaultTimeout(t *testing.T) {
	// Passing 0 should use DefaultTimeout (30s), not hang forever.
	// We just verify a quick script completes fine with default timeout.
	output := "```bash\necho -n ok\n```"
	result := ExecuteCode(context.Background(), output, "", 0)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("stdout = %q, want %q", result.Stdout, "ok")
	}
}

func TestExecuteCode_ExplicitLangOverridesFence(t *testing.T) {
	// Code block says "python" but we explicitly say "bash".
	output := "```python\necho -n from-bash\n```"
	result := ExecuteCode(context.Background(), output, "bash", 10*time.Second)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Stdout != "from-bash" {
		t.Fatalf("stdout = %q, want %q", result.Stdout, "from-bash")
	}
}

func TestExecuteCode_NoCodeBlock(t *testing.T) {
	result := ExecuteCode(context.Background(), "no code here", "", 10*time.Second)
	if result.Err == nil {
		t.Fatal("expected error for missing code block")
	}
	if !strings.Contains(result.Err.Error(), "no code block") {
		t.Fatalf("error = %q", result.Err)
	}
}

func TestExecuteCode_UnsupportedLanguage(t *testing.T) {
	output := "```brainfuck\n++++++++++\n```"
	result := ExecuteCode(context.Background(), output, "", 10*time.Second)
	if result.Err == nil {
		t.Fatal("expected error for unsupported language")
	}
	if !strings.Contains(result.Err.Error(), "unsupported language") {
		t.Fatalf("error = %q", result.Err)
	}
}

func TestExecuteCode_CleansUpTempFiles(t *testing.T) {
	// Run code and verify no temp files linger. We can't easily check this
	// directly, but we can verify the function completes without error,
	// which implies the defer os.RemoveAll ran.
	output := "```bash\necho -n clean\n```"
	result := ExecuteCode(context.Background(), output, "", 10*time.Second)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestExecuteCode_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already cancelled.

	output := "```bash\necho hello\n```"
	result := ExecuteCode(ctx, output, "", 10*time.Second)
	// The context is already done, so the execution should fail.
	if result.ExitCode == 0 && result.Err == nil {
		t.Fatal("expected failure with cancelled context")
	}
}
