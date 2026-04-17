package cmd

import "testing"

func TestDefaultRegistry(t *testing.T) {
	reg := defaultRegistry()

	for _, name := range []string{"claude-code", "ollama"} {
		runner, err := reg.Create(name, nil)
		if err != nil {
			t.Errorf("expected %q to be registered, got error: %v", name, err)
		}
		if runner == nil {
			t.Errorf("expected non-nil runner for %q", name)
		}
	}

	_, err := reg.Create("nonexistent", nil)
	if err == nil {
		t.Error("expected error for unregistered runner")
	}
}
