package report

import (
	"bytes"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

func TestWrite_Markdown(t *testing.T) {
	sr := &result.SuiteResult{StartedAt: time.Now(), FinishedAt: time.Now()}
	var buf bytes.Buffer
	err := Write(&buf, sr, "markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestWrite_JSON(t *testing.T) {
	sr := &result.SuiteResult{StartedAt: time.Now(), FinishedAt: time.Now()}
	var buf bytes.Buffer
	err := Write(&buf, sr, "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestWrite_InvalidFormat(t *testing.T) {
	sr := &result.SuiteResult{StartedAt: time.Now(), FinishedAt: time.Now()}
	var buf bytes.Buffer
	err := Write(&buf, sr, "xml")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}
