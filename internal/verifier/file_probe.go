package verifier

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/driangle/skival/internal/suite"
)

// FileProbeVerifier checks that a file exists and/or contains expected content.
type FileProbeVerifier struct {
	Probe suite.FileProbe
}

func (v *FileProbeVerifier) Verify(_ context.Context, _ VerifyInput) VerifyResult {
	a := v.Probe.Assert

	_, err := os.Stat(v.Probe.Path)
	exists := err == nil

	if a.Exists != nil {
		if *a.Exists && !exists {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("file %q does not exist", v.Probe.Path),
			}
		}
		if !*a.Exists && exists {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("file %q exists but expected it not to", v.Probe.Path),
			}
		}
	}

	if a.Contains != "" {
		if !exists {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("cannot check contains: file %q does not exist", v.Probe.Path),
			}
		}
		data, err := os.ReadFile(v.Probe.Path)
		if err != nil {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("failed to read file %q: %v", v.Probe.Path, err),
			}
		}
		if !strings.Contains(string(data), a.Contains) {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("expected %q not found in file %q", a.Contains, v.Probe.Path),
			}
		}
	}

	return VerifyResult{Pass: true, Reason: fmt.Sprintf("file probe passed for %q", v.Probe.Path)}
}
