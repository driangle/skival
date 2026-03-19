package cmd

import "testing"

func TestRootCmd_VerboseFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("verbose")
	if f == nil {
		t.Fatal("expected --verbose persistent flag on root command")
	}
	if f.Shorthand != "v" {
		t.Errorf("expected shorthand -v, got %q", f.Shorthand)
	}
	if f.DefValue != "false" {
		t.Errorf("expected default false, got %q", f.DefValue)
	}
}
