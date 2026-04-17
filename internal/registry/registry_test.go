package registry

import (
	"context"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
)

// stubRunner implements agentrunner.Runner for testing.
type stubRunner struct{ tag string }

func (s *stubRunner) Run(context.Context, string, ...agentrunner.Option) (*agentrunner.Result, error) {
	return nil, nil
}

func (s *stubRunner) Start(context.Context, string, ...agentrunner.Option) (*agentrunner.Session, error) {
	return nil, nil
}

func TestRegisterAndCreate(t *testing.T) {
	reg := New()
	reg.Register("stub", func(cfg map[string]any) (agentrunner.Runner, error) {
		return &stubRunner{tag: "ok"}, nil
	})

	runner, err := reg.Create("stub", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sr, ok := runner.(*stubRunner)
	if !ok {
		t.Fatal("expected *stubRunner")
	}
	if sr.tag != "ok" {
		t.Fatalf("got tag %q, want %q", sr.tag, "ok")
	}
}

func TestCreatePassesConfig(t *testing.T) {
	reg := New()
	var received map[string]any
	reg.Register("cfg", func(cfg map[string]any) (agentrunner.Runner, error) {
		received = cfg
		return &stubRunner{}, nil
	})

	config := map[string]any{"key": "value", "n": 42}
	_, err := reg.Create("cfg", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received["key"] != "value" {
		t.Fatalf("config not passed through: got %v", received)
	}
}

func TestCreateUnknownName(t *testing.T) {
	reg := New()
	_, err := reg.Create("nope", nil)
	if err == nil {
		t.Fatal("expected error for unknown runner")
	}
	want := `unknown runner "nope"`
	if err.Error() != want {
		t.Fatalf("got error %q, want %q", err.Error(), want)
	}
}
