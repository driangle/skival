package verifier

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// StateAssertion defines a single HTTP assertion to verify.
type StateAssertion struct {
	URL    string
	Method string
	Expect string
}

// StateVerifier makes HTTP requests and checks response bodies for expected strings.
type StateVerifier struct {
	Assertions []StateAssertion
	Client     *http.Client
}

func (v *StateVerifier) client() *http.Client {
	if v.Client != nil {
		return v.Client
	}
	return http.DefaultClient
}

func (v *StateVerifier) Verify(ctx context.Context, _ VerifyInput) VerifyResult {
	for _, a := range v.Assertions {
		if failure := v.checkAssertion(ctx, a); failure != nil {
			return *failure
		}
	}
	return VerifyResult{Pass: true, Reason: "all state assertions passed"}
}

// checkAssertion issues one HTTP request and verifies the expected body
// substring, returning nil when the assertion holds.
func (v *StateVerifier) checkAssertion(ctx context.Context, a StateAssertion) *VerifyResult {
	method := a.Method
	if method == "" {
		method = http.MethodGet
	}

	body, failure := v.fetchBody(ctx, method, a.URL)
	if failure != nil {
		return failure
	}

	if !strings.Contains(body, a.Expect) {
		return &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("expected %q not found in response from %s %s", a.Expect, method, a.URL),
		}
	}
	return nil
}

// fetchBody performs the request and returns the response body, or a failing
// VerifyResult describing the transport-level error.
func (v *StateVerifier) fetchBody(ctx context.Context, method, url string) (string, *VerifyResult) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return "", &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("failed to create request for %s %s: %v", method, url, err),
		}
	}

	resp, err := v.client().Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", &VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("request timed out for %s %s: %v", method, url, ctx.Err()),
			}
		}
		return "", &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("request failed for %s %s: %v", method, url, err),
		}
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("failed to read response body from %s %s: %v", method, url, err),
		}
	}
	return string(body), nil
}
