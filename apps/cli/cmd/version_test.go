package cmd

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if got != "dev\n" {
		t.Errorf("version output = %q, want %q", got, "dev\n")
	}
}
