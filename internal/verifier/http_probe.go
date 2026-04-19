package verifier

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/driangle/skival/internal/suite"
)

// HTTPProbeVerifier makes an HTTP request and checks status code and body content.
type HTTPProbeVerifier struct {
	Probe  suite.HTTPProbe
	Client *http.Client
}

func (v *HTTPProbeVerifier) client() *http.Client {
	if v.Client != nil {
		return v.Client
	}
	return http.DefaultClient
}

func (v *HTTPProbeVerifier) Verify(ctx context.Context, _ VerifyInput) VerifyResult {
	method := v.Probe.Method
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, v.Probe.URL, nil)
	if err != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("failed to create request for %s %s: %v", method, v.Probe.URL, err),
		}
	}

	resp, err := v.client().Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("request timed out for %s %s: %v", method, v.Probe.URL, ctx.Err()),
			}
		}
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("request failed for %s %s: %v", method, v.Probe.URL, err),
		}
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("failed to read response body from %s %s: %v", method, v.Probe.URL, err),
		}
	}

	a := v.Probe.Assert

	if a.Status != nil && resp.StatusCode != *a.Status {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("expected status %d, got %d from %s %s", *a.Status, resp.StatusCode, method, v.Probe.URL),
		}
	}

	if a.BodyContains != "" && !strings.Contains(string(body), a.BodyContains) {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("expected %q not found in response from %s %s", a.BodyContains, method, v.Probe.URL),
		}
	}

	return VerifyResult{Pass: true, Reason: fmt.Sprintf("http probe passed for %s %s", method, v.Probe.URL)}
}
