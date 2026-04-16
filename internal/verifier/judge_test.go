package verifier

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/suite"
)

type fakeRunner struct {
	text string
	err  error
}

func (f *fakeRunner) Run(_ context.Context, _ string, _ ...agentrunner.Option) (*agentrunner.Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &agentrunner.Result{Text: f.text}, nil
}

func (f *fakeRunner) Start(_ context.Context, _ string, _ ...agentrunner.Option) (*agentrunner.Session, error) {
	if f.err != nil {
		return nil, f.err
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := &agentrunner.Result{Text: f.text}
	return agentrunner.NewSession(ctx, cancel, func(_ context.Context, messages chan<- agentrunner.Message) (*agentrunner.Result, error) {
		messages <- agentrunner.Message{Raw: json.RawMessage(`{"type":"assistant","text":"` + f.text + `"}`)}
		return result, nil
	}), nil
}

func TestJudgeVerifier_Pass(t *testing.T) {
	v := &JudgeVerifier{
		Runner:   &fakeRunner{text: "PASS: looks good"},
		Criteria: []string{"output is correct"},
		Prompt:   "do something",
	}
	r := v.Verify(context.Background(), VerifyInput{RunOutput: "hello"})
	if !r.Pass {
		t.Errorf("expected pass, got fail: %s", r.Reason)
	}
	if r.Reason != "looks good" {
		t.Errorf("reason = %q, want 'looks good'", r.Reason)
	}
}

func TestJudgeVerifier_Fail(t *testing.T) {
	v := &JudgeVerifier{
		Runner:   &fakeRunner{text: "FAIL: missing edge case handling"},
		Criteria: []string{"handles edge cases"},
		Prompt:   "do something",
	}
	r := v.Verify(context.Background(), VerifyInput{RunOutput: "hello"})
	if r.Pass {
		t.Error("expected fail")
	}
	if r.Reason != "missing edge case handling" {
		t.Errorf("reason = %q", r.Reason)
	}
}

func TestJudgeVerifier_RunnerError(t *testing.T) {
	v := &JudgeVerifier{
		Runner:   &fakeRunner{err: errors.New("api error")},
		Criteria: []string{"something"},
		Prompt:   "do something",
	}
	r := v.Verify(context.Background(), VerifyInput{})
	if r.Pass {
		t.Error("expected fail on runner error")
	}
}

func TestJudgeVerifier_UnparseableResponse(t *testing.T) {
	v := &JudgeVerifier{
		Runner:   &fakeRunner{text: "I think it looks okay maybe"},
		Criteria: []string{"something"},
		Prompt:   "do something",
	}
	r := v.Verify(context.Background(), VerifyInput{})
	if r.Pass {
		t.Error("expected fail on unparseable response")
	}
}

func TestJudgeVerifier_ConversationPopulated(t *testing.T) {
	v := &JudgeVerifier{
		Runner:   &fakeRunner{text: "PASS: looks good"},
		Criteria: []string{"output is correct"},
		Prompt:   "do something",
	}
	r := v.Verify(context.Background(), VerifyInput{RunOutput: "hello"})
	if !r.Pass {
		t.Fatalf("expected pass, got fail: %s", r.Reason)
	}
	if len(r.Conversation) == 0 {
		t.Error("expected non-empty conversation")
	}
}

func TestJudgeVerifier_ConversationNilOnError(t *testing.T) {
	v := &JudgeVerifier{
		Runner:   &fakeRunner{err: errors.New("api error")},
		Criteria: []string{"something"},
		Prompt:   "do something",
	}
	r := v.Verify(context.Background(), VerifyInput{})
	if r.Pass {
		t.Error("expected fail on runner error")
	}
	if r.Conversation != nil {
		t.Error("expected nil conversation on error")
	}
}

func TestParseJudgeResponse_CaseInsensitive(t *testing.T) {
	r := parseJudgeResponse("pass: all good")
	if !r.Pass {
		t.Error("expected pass")
	}

	r = parseJudgeResponse("Fail: not good")
	if r.Pass {
		t.Error("expected fail")
	}
}

func TestParseJudgeResponse_MultiLine(t *testing.T) {
	r := parseJudgeResponse("Some preamble\nPASS: criteria met\nMore text")
	if !r.Pass {
		t.Error("expected pass from multiline")
	}
}

func TestBuildPipeline_WithJudge(t *testing.T) {
	runner := &fakeRunner{text: "PASS: ok"}
	p := BuildPipeline(suite.Correctness{
		Judge: []string{"output is correct"},
	}, "", WithJudge(runner, "do something"))

	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(p.steps))
	}
	if p.steps[0].name != "judge" {
		t.Errorf("expected judge step, got %q", p.steps[0].name)
	}
}

func TestBuildPipeline_JudgeWithoutRunner(t *testing.T) {
	p := BuildPipeline(suite.Correctness{
		Judge: []string{"output is correct"},
	}, "")

	if p != nil {
		t.Error("expected nil pipeline when judge configured but no runner")
	}
}

func TestBuildPipeline_JudgeIsLastStep(t *testing.T) {
	runner := &fakeRunner{text: "PASS: ok"}
	p := BuildPipeline(suite.Correctness{
		ExpectedOutput: []string{"hello"},
		Judge:          []string{"is good"},
	}, "", WithJudge(runner, "prompt"))

	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(p.steps))
	}
	if p.steps[0].name != "output" {
		t.Errorf("step 0 should be output, got %q", p.steps[0].name)
	}
	if p.steps[1].name != "judge" {
		t.Errorf("step 1 should be judge, got %q", p.steps[1].name)
	}
}
