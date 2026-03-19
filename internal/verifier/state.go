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
		method := a.Method
		if method == "" {
			method = http.MethodGet
		}

		req, err := http.NewRequestWithContext(ctx, method, a.URL, nil)
		if err != nil {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("failed to create request for %s %s: %v", method, a.URL, err),
			}
		}

		resp, err := v.client().Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return VerifyResult{
					Pass:   false,
					Reason: fmt.Sprintf("request timed out for %s %s: %v", method, a.URL, ctx.Err()),
				}
			}
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("request failed for %s %s: %v", method, a.URL, err),
			}
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("failed to read response body from %s %s: %v", method, a.URL, err),
			}
		}

		if !strings.Contains(string(body), a.Expect) {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("expected %q not found in response from %s %s", a.Expect, method, a.URL),
			}
		}
	}
	return VerifyResult{Pass: true, Reason: "all state assertions passed"}
}
