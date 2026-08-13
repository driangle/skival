package exec

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
)

// runWith starts the runner with the given config and prompt in workDir and
// returns the collected conversation and the final result.
func runWith(t *testing.T, cfg Config, prompt, workDir string, extraOpts ...agentrunner.Option) ([]json.RawMessage, *agentrunner.Result, error) {
	t.Helper()
	opts := append([]agentrunner.Option{WithConfig(cfg)}, extraOpts...)
	if workDir != "" {
		opts = append(opts, agentrunner.WithWorkingDir(workDir))
	}
	session, err := NewRunner().Start(context.Background(), prompt, opts...)
	if err != nil {
		return nil, nil, err
	}
	var conv []json.RawMessage
	for msg := range session.Messages {
		if msg.Raw != nil {
			conv = append(conv, msg.Raw)
		}
	}
	res, err := session.Result()
	return conv, res, err
}

func TestStart_StdinDelivery(t *testing.T) {
	// cat echoes stdin to stdout, so the prompt round-trips as the output text.
	_, res, err := runWith(t, Config{Command: []string{"cat"}}, "hello via stdin", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(res.Text) != "hello via stdin" {
		t.Errorf("Text = %q, want %q", res.Text, "hello via stdin")
	}
	if res.ExitCode != 0 || res.IsError {
		t.Errorf("ExitCode=%d IsError=%v, want 0/false", res.ExitCode, res.IsError)
	}
}

func TestStart_EnvDelivery(t *testing.T) {
	cfg := Config{
		Command:   []string{"sh", "-c", `printf '%s' "$SKIVAL_PROMPT"`},
		PromptVia: PromptViaEnv,
	}
	_, res, err := runWith(t, cfg, "from env", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Text != "from env" {
		t.Errorf("Text = %q, want %q", res.Text, "from env")
	}
}

func TestStart_CustomPromptEnv(t *testing.T) {
	cfg := Config{
		Command:   []string{"sh", "-c", `printf '%s' "$MY_PROMPT"`},
		PromptVia: PromptViaEnv,
		PromptEnv: "MY_PROMPT",
	}
	_, res, err := runWith(t, cfg, "custom env var", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Text != "custom env var" {
		t.Errorf("Text = %q, want %q", res.Text, "custom env var")
	}
}

func TestStart_ArgFileDelivery(t *testing.T) {
	// cat {prompt_file} prints the prompt written to the temp file.
	cfg := Config{
		Command:   []string{"cat", PromptFilePlaceholder},
		PromptVia: PromptViaArgFile,
	}
	_, res, err := runWith(t, cfg, "prompt in a file", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Text != "prompt in a file" {
		t.Errorf("Text = %q, want %q", res.Text, "prompt in a file")
	}
}

func TestStart_RunDirInjected(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Command: []string{"sh", "-c", `printf '%s' "$SKIVAL_RUN_DIR"`}}
	_, res, err := runWith(t, cfg, "x", dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Text != dir {
		t.Errorf("SKIVAL_RUN_DIR = %q, want %q", res.Text, dir)
	}
}

func TestStart_NonZeroExit(t *testing.T) {
	cfg := Config{Command: []string{"sh", "-c", "echo oops; exit 3"}}
	_, res, err := runWith(t, cfg, "x", "")
	if err != nil {
		t.Fatalf("Run: unexpected error %v (non-zero exit must not be a runner error)", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}
	if !res.IsError {
		t.Errorf("IsError = false, want true for non-zero exit")
	}
	if strings.TrimSpace(res.Text) != "oops" {
		t.Errorf("Text = %q, want %q", res.Text, "oops")
	}
}

func TestStart_DurationMeasured(t *testing.T) {
	// The exec runner must report the subprocess wall-clock time; a sleeping
	// command should yield a duration at least as long as it slept.
	cfg := Config{Command: []string{"sleep", "0.1"}}
	_, res, err := runWith(t, cfg, "x", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Duration < 100*time.Millisecond {
		t.Errorf("Duration = %v, want >= 100ms for a 0.1s sleep", res.Duration)
	}
}

func TestStart_Timeout(t *testing.T) {
	cfg := Config{Command: []string{"sleep", "5"}}
	session, err := NewRunner().Start(context.Background(), "x",
		WithConfig(cfg), agentrunner.WithTimeout(100*time.Millisecond))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	for range session.Messages {
	}
	_, err = session.Result()
	if !errors.Is(err, agentrunner.ErrTimeout) {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestStart_MissingCommand(t *testing.T) {
	_, err := NewRunner().Start(context.Background(), "x", WithConfig(Config{}))
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestStart_BinaryNotFound(t *testing.T) {
	cfg := Config{Command: []string{"skival-nonexistent-binary-xyz"}}
	_, err := NewRunner().Start(context.Background(), "x", WithConfig(cfg))
	if !errors.Is(err, agentrunner.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestStart_EventsIngestedAndFinalParsed(t *testing.T) {
	dir := t.TempDir()
	// Program writes two events then a final event to $SKIVAL_EVENTS_PATH.
	script := `printf '%s\n' \
'{"type":"tool_use","name":"read_file","input":{"path":"README.md"}}' \
'{"type":"tool_result","tool_use_id":"1","content":"contents"}' \
'{"type":"final","text":"done","usage":{"input_tokens":10,"output_tokens":5},"cost_usd":0.02}' \
> "$SKIVAL_EVENTS_PATH"; printf 'stdout answer'`
	cfg := Config{
		Command:    []string{"sh", "-c", script},
		EventsPath: "events.jsonl",
	}
	conv, res, err := runWith(t, cfg, "x", dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(conv) != 2 {
		t.Fatalf("forwarded %d events, want 2 (final is consumed, not forwarded)", len(conv))
	}
	if res.Text != "stdout answer" {
		t.Errorf("Text = %q, want stdout to win over final.text", res.Text)
	}
	if res.CostUSD != 0.02 {
		t.Errorf("CostUSD = %v, want 0.02", res.CostUSD)
	}
	if res.Usage.InputTokens != 10 || res.Usage.OutputTokens != 5 {
		t.Errorf("Usage = %+v, want input=10 output=5", res.Usage)
	}
	// The forwarded events must be valid JSON carrying the tool name.
	if !strings.Contains(string(conv[0]), "read_file") {
		t.Errorf("first event = %s, want it to contain read_file", conv[0])
	}
}

func TestStart_FinalTextFallbackWhenNoStdout(t *testing.T) {
	dir := t.TempDir()
	script := `printf '%s\n' '{"type":"final","text":"only from final"}' > "$SKIVAL_EVENTS_PATH"`
	cfg := Config{Command: []string{"sh", "-c", script}, EventsPath: "events.jsonl"}
	_, res, err := runWith(t, cfg, "x", dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Text != "only from final" {
		t.Errorf("Text = %q, want final.text fallback", res.Text)
	}
}

func TestStart_MissingEventsFileTolerated(t *testing.T) {
	dir := t.TempDir()
	// Program never creates the events file.
	cfg := Config{Command: []string{"sh", "-c", "printf ok"}, EventsPath: "events.jsonl"}
	conv, res, err := runWith(t, cfg, "x", dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(conv) != 0 {
		t.Errorf("conv = %v, want empty", conv)
	}
	if res.Text != "ok" {
		t.Errorf("Text = %q, want ok", res.Text)
	}
}

func TestStart_EventsPathRunDirExpansion(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Command:    []string{"sh", "-c", `printf '%s\n' '{"type":"tool_use","name":"t"}' > "$SKIVAL_EVENTS_PATH"`},
		EventsPath: "${SKIVAL_RUN_DIR}/sub/events.jsonl",
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	conv, _, err := runWith(t, cfg, "x", dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(conv) != 1 {
		t.Errorf("forwarded %d events, want 1 (expansion of ${SKIVAL_RUN_DIR})", len(conv))
	}
}
