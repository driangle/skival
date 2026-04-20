package cmd

import "testing"

// TestSamplesFlagDefaultIsUnsetSentinel guards against regression where the
// --samples flag default was 1, which silently overrode suite.yaml's
// `defaults.samples` because the executor treats any positive value as a CLI
// override. The default must be 0 so the YAML value is honored.
func TestSamplesFlagDefaultIsUnsetSentinel(t *testing.T) {
	f := runCmd.Flags().Lookup("samples")
	if f == nil {
		t.Fatal("expected --samples flag to be registered")
	}
	if f.DefValue != "0" {
		t.Errorf("expected --samples default to be 0 (unset sentinel), got %q", f.DefValue)
	}
}

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
