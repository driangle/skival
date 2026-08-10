package report

// htmlData is the template data for the HTML report.
type htmlData struct {
	Description  string
	StartedAt    string
	FinishedAt   string
	MultiRunner  bool
	MultiModel   bool
	Results      []htmlResultRow
	Errors       []htmlError
	Skipped      []htmlSkippedGroup
	Comparisons  []htmlComparison
	Rankings     []htmlRanking
	ShowRankings bool
	ShowQuality  bool
}

type htmlComparison struct {
	Name    string
	ID      string
	Model   string
	Skipped string
	Scores  []htmlComparativeScore
}

type htmlComparativeScore struct {
	Variant string
	Rating  string
	Score   string
	Reason  string
}

type htmlResultRow struct {
	Eval     string
	Variant  string
	Sample   string
	Status   string
	Cost     string
	Duration string
	IsAgg    bool
	CVInfo   string
}

type htmlError struct {
	Name    string
	ID      string
	Message string
}

type htmlSkippedGroup struct {
	Name    string
	ID      string
	Entries []htmlSkippedEntry
}

type htmlSkippedEntry struct {
	Name   string
	Reason string
}

type htmlRanking struct {
	Rank           int
	Name           string
	Runner         string
	Model          string
	CompositeScore string
	PassRate       string
	QualityScore   string
	MedianCost     string
	MedianDuration string
}
