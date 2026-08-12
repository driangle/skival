package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

// sessionSuite has two runs: one with a rendered session page (link) and one
// with only a session id (fallback hint).
func sessionSuite() *result.SuiteResult {
	return &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "e1",
			Variants: []result.VariantResult{{
				Name: "control",
				Runs: []result.RunResult{
					{Sample: 1, Pass: boolPtr(true), SessionID: "withpage1", SessionPage: "evals/e1/control/run-1.session.html"},
					{Sample: 2, Pass: boolPtr(true), SessionID: "hintonly2"},
				},
			}},
		}},
	}
}

func TestWriteHTML_SessionColumn(t *testing.T) {
	out := renderHTML(t, sessionSuite())
	wantAll(t, out,
		"<th>Session</th>",
		`href="evals/e1/control/run-1.session.html"`, // page link
		`title="vibeview show hintonly2"`,             // fallback hint tooltip
	)
}

func TestWriteHTML_NoSessionColumnWithoutSessions(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "e1",
			Variants: []result.VariantResult{{
				Name: "control",
				Runs: []result.RunResult{{Sample: 1, Pass: boolPtr(true)}},
			}},
		}},
	}
	if strings.Contains(renderHTML(t, sr), "<th>Session</th>") {
		t.Error("Session column should be absent when no run has a session id")
	}
}

func TestWriteMarkdown_SessionsSection(t *testing.T) {
	var buf bytes.Buffer
	WriteMarkdown(&buf, sessionSuite(), DefaultWeights())
	out := buf.String()
	wantAll(t, out,
		"## Sessions",
		"[view session](evals/e1/control/run-1.session.html)",
		"`vibeview show hintonly2`",
	)
}

func TestWriteJSON_SessionFields(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sessionSuite(), DefaultWeights()); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	wantAll(t, buf.String(),
		`"session_id": "withpage1"`,
		`"session_page": "evals/e1/control/run-1.session.html"`,
		`"session_id": "hintonly2"`,
	)
}
