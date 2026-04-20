package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, versionString()) {
		t.Errorf("--version output = %q, want it to contain %q", got, versionString())
	}
}

func TestVersionSubcommandRemoved(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == "version" {
			t.Fatalf("expected no `version` subcommand, found one")
		}
	}
}

func TestVersionString_IncludesCommitAndSuffix(t *testing.T) {
	origCommit, origSuffix, origVersion := commit, suffix, Version
	t.Cleanup(func() {
		commit = origCommit
		suffix = origSuffix
		Version = origVersion
	})

	Version = "1.2.3"
	commit = "abcdef0123456789"
	suffix = "release"

	got := versionString()
	want := fmt.Sprintf("1.2.3 (commit %s, release)", "abcdef0")
	if got != want {
		t.Errorf("versionString() = %q, want %q", got, want)
	}
}

func TestVersionString_ShortCommitUnchanged(t *testing.T) {
	origCommit, origSuffix, origVersion := commit, suffix, Version
	t.Cleanup(func() {
		commit = origCommit
		suffix = origSuffix
		Version = origVersion
	})

	Version = "0.1.0"
	commit = "abc1234"
	suffix = "dev"

	got := versionString()
	want := "0.1.0 (commit abc1234, dev)"
	if got != want {
		t.Errorf("versionString() = %q, want %q", got, want)
	}
}
