package report

import "html/template"

// htmlData is the template data for the HTML report. Every value the template
// renders is precomputed here: the template does no arithmetic, so the numbers
// in the summary block can never drift from the tables below it.
type htmlData struct {
	CSS template.CSS
	JS  template.JS

	Description  string
	StartedAt    string
	FinishedAt   string
	WallDuration string
	Counts       string
	WeightsNote  string

	Verdict htmlVerdict
	Health  htmlHealth

	VariantNames []string
	Rankings     []htmlRanking
	Evals        []htmlEval
	Errors       []htmlError

	ShowQuality bool
	// HasSessions is true when at least one run carries session info, which
	// gates the "Session" column in the samples tables.
	HasSessions bool
}

// htmlVerdict is the headline answer: which variant won, and by how much
// against the runner-up. Winner is empty when fewer than two variants ran.
type htmlVerdict struct {
	Winner      string
	Summary     string
	ScoreMargin string
	CostDelta   string
	SpeedDelta  string
}

// htmlHealth is the correctness-at-a-glance panel: one cell per sample run.
type htmlHealth struct {
	PassSummary string
	TotalSpend  string
	Runners     string
	Judge       string
	Cells       []htmlHealthCell
}

type htmlHealthCell struct {
	Pass  bool
	Title string
}

// htmlEval is one collapsible eval card: its sample rows, the judge's verdicts,
// and any variants skipped for it.
type htmlEval struct {
	ID       string
	Name     string
	Judge    string
	Summary  string
	Note     string
	Open     bool
	Rows     []htmlResultRow
	Verdicts []htmlJudgeVerdict
	Skipped  []htmlSkippedEntry
}

// htmlResultRow is one sample run, or the per-variant aggregate row (IsAgg).
// SpanStyle positions the min–max duration bar within the eval's slowest run.
type htmlResultRow struct {
	Variant     string
	Sample      string
	Status      string
	StatusClass string
	Cost        string
	Duration    string
	CVInfo      string
	SpanStyle   template.CSS
	IsAgg       bool
	// Detail is the failure reason for an errored run, revealed by expanding the
	// row. Empty for runs that completed.
	Detail string
	// SessionPage is the relative path to a static vibeview session page for the
	// run, when one was produced. SessionID/SessionShort back the fallback hint
	// shown when no page exists.
	SessionPage  string
	SessionID    string
	SessionShort string
}

// htmlJudgeVerdict is one comparative-judge score. Pips renders the 1-5 rating
// as filled/empty marks; Teaser is the first sentence of Reason.
type htmlJudgeVerdict struct {
	Variant string
	Rating  string
	Score   string
	Teaser  string
	Reason  string
	Pips    []bool
}

// htmlRanking is one row of the rankings block. The *Width fields are inline
// bar widths, expressed relative to the worst value in the column.
type htmlRanking struct {
	Rank           int
	Name           string
	Attribution    string
	CompositeScore string
	PassRate       string
	QualityScore   string
	MedianCost     string
	MedianDuration string
	CompositeWidth template.CSS
	QualityWidth   template.CSS
	CostWidth      template.CSS
	DurationWidth  template.CSS
}

type htmlError struct {
	Name    string
	ID      string
	Message string
}

type htmlSkippedEntry struct {
	Name   string
	Reason string
}
