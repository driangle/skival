package verifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStateVerifier_PassWhenAllAssertionsMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","count":42}`))
	}))
	defer srv.Close()

	v := &StateVerifier{
		Assertions: []StateAssertion{
			{URL: srv.URL, Method: "GET", Expect: `"status":"ok"`},
			{URL: srv.URL, Method: "GET", Expect: "42"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass, got fail: %s", result.Reason)
	}
}

func TestStateVerifier_FailWhenSubstringMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"error"}`))
	}))
	defer srv.Close()

	v := &StateVerifier{
		Assertions: []StateAssertion{
			{URL: srv.URL, Expect: "ok"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail when expected string not in response")
	}
	if result.Reason == "" {
		t.Fatal("expected a failure reason")
	}
}

func TestStateVerifier_DefaultsToGET(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := &StateVerifier{
		Assertions: []StateAssertion{
			{URL: srv.URL, Expect: "ok"},
		},
	}
	v.Verify(context.Background(), VerifyInput{})
	if gotMethod != "GET" {
		t.Fatalf("expected GET, got %s", gotMethod)
	}
}

func TestStateVerifier_SupportsPostMethod(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Write([]byte("created"))
	}))
	defer srv.Close()

	v := &StateVerifier{
		Assertions: []StateAssertion{
			{URL: srv.URL, Method: "POST", Expect: "created"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass: %s", result.Reason)
	}
	if gotMethod != "POST" {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
}

func TestStateVerifier_FailOnConnectionError(t *testing.T) {
	v := &StateVerifier{
		Assertions: []StateAssertion{
			{URL: "http://127.0.0.1:1", Expect: "anything"},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on connection error")
	}
}

func TestStateVerifier_RespectsContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte("late"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	v := &StateVerifier{
		Assertions: []StateAssertion{
			{URL: srv.URL, Expect: "late"},
		},
	}
	result := v.Verify(ctx, VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on timeout")
	}
}

func TestStateVerifier_EmptyAssertionsPass(t *testing.T) {
	v := &StateVerifier{}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass for empty assertions: %s", result.Reason)
	}
}

func TestStateVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &StateVerifier{}
}
