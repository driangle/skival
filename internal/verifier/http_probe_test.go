package verifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/driangle/skival/internal/suite"
)

func intPtr(n int) *int { return &n }

func TestHTTPProbeVerifier_PassOnStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	v := &HTTPProbeVerifier{
		Probe: suite.HTTPProbe{
			URL: srv.URL,
			Assert: suite.HTTPProbeAssert{
				Status:       intPtr(200),
				BodyContains: "ok",
			},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass, got fail: %s", result.Reason)
	}
}

func TestHTTPProbeVerifier_FailOnWrongStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	v := &HTTPProbeVerifier{
		Probe: suite.HTTPProbe{
			URL: srv.URL,
			Assert: suite.HTTPProbeAssert{
				Status: intPtr(200),
			},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on wrong status code")
	}
}

func TestHTTPProbeVerifier_FailOnMissingBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("error"))
	}))
	defer srv.Close()

	v := &HTTPProbeVerifier{
		Probe: suite.HTTPProbe{
			URL: srv.URL,
			Assert: suite.HTTPProbeAssert{
				BodyContains: "ok",
			},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail when body_contains not found")
	}
}

func TestHTTPProbeVerifier_DefaultsToGET(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := &HTTPProbeVerifier{
		Probe: suite.HTTPProbe{
			URL: srv.URL,
			Assert: suite.HTTPProbeAssert{
				BodyContains: "ok",
			},
		},
	}
	v.Verify(context.Background(), VerifyInput{})
	if gotMethod != "GET" {
		t.Fatalf("expected GET, got %s", gotMethod)
	}
}

func TestHTTPProbeVerifier_FailOnConnectionError(t *testing.T) {
	v := &HTTPProbeVerifier{
		Probe: suite.HTTPProbe{
			URL: "http://127.0.0.1:1",
			Assert: suite.HTTPProbeAssert{
				Status: intPtr(200),
			},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on connection error")
	}
}

func TestHTTPProbeVerifier_RespectsContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	v := &HTTPProbeVerifier{
		Probe: suite.HTTPProbe{
			URL: srv.URL,
			Assert: suite.HTTPProbeAssert{
				Status: intPtr(200),
			},
		},
	}
	result := v.Verify(ctx, VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on timeout")
	}
}

func TestHTTPProbeVerifier_PassWithNoAssertions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("error"))
	}))
	defer srv.Close()

	v := &HTTPProbeVerifier{
		Probe: suite.HTTPProbe{
			URL:    srv.URL,
			Assert: suite.HTTPProbeAssert{},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass with no assertions, got: %s", result.Reason)
	}
}

func TestHTTPProbeVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &HTTPProbeVerifier{}
}
