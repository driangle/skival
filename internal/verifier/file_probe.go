package verifier

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/driangle/skival/internal/suite"
)

// FileProbeVerifier checks that a file exists and/or contains expected content.
type FileProbeVerifier struct {
	Probe suite.FileProbe
	Dir   string // working directory for resolving relative paths
}

func (v *FileProbeVerifier) Verify(_ context.Context, _ VerifyInput) VerifyResult {
	a := v.Probe.Assert

	path := v.Probe.Path
	if !filepath.IsAbs(path) && v.Dir != "" {
		path = filepath.Join(v.Dir, path)
	}

	_, err := os.Stat(path)
	exists := err == nil

	if failure := checkFileExists(a, path, exists); failure != nil {
		return *failure
	}
	if failure := checkFileContains(a, path, exists); failure != nil {
		return *failure
	}

	return VerifyResult{Pass: true, Reason: fmt.Sprintf("file probe passed for %q", path)}
}

// checkFileExists validates the Exists assertion, returning nil when it holds.
func checkFileExists(a suite.FileProbeAssert, path string, exists bool) *VerifyResult {
	if a.Exists == nil {
		return nil
	}
	if *a.Exists && !exists {
		return &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("file %q does not exist", path),
		}
	}
	if !*a.Exists && exists {
		return &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("file %q exists but expected it not to", path),
		}
	}
	return nil
}

// checkFileContains validates the Contains assertion, returning nil when it holds.
func checkFileContains(a suite.FileProbeAssert, path string, exists bool) *VerifyResult {
	if a.Contains == "" {
		return nil
	}
	if !exists {
		return &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("cannot check contains: file %q does not exist", path),
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("failed to read file %q: %v", path, err),
		}
	}
	if !strings.Contains(string(data), a.Contains) {
		return &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("expected %q not found in file %q", a.Contains, path),
		}
	}
	return nil
}
