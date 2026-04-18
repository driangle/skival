package report

import (
	"fmt"
	"html/template"
	"io"

	"github.com/driangle/skival/internal/result"
)

// htmlData is the template data for the HTML report.
type htmlData struct {
	Description   string
	StartedAt     string
	FinishedAt    string
	MultiRunner   bool
	MultiModel    bool
	Results       []htmlResultRow
	Errors        []htmlError
	Skipped       []htmlSkippedGroup
	Rankings      []htmlRanking
	ShowRankings  bool
}

type htmlResultRow struct {
	Eval      string
	Treatment string
	Sample    string
	Status    string
	Cost      string
	Duration  string
	IsAgg     bool
	CVInfo    string
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
	MedianCost     string
	MedianDuration string
}

// WriteHTML writes a self-contained HTML report to w.
func WriteHTML(w io.Writer, sr *result.SuiteResult, weights Weights) error {
	data := buildHTMLData(sr, weights)

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing HTML template: %w", err)
	}
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing HTML template: %w", err)
	}
	return nil
}

func buildHTMLData(sr *result.SuiteResult, weights Weights) htmlData {
	multi := hasMultipleRunners(sr)
	multiModel := hasMultipleModels(sr)

	d := htmlData{
		Description: sr.Description,
		StartedAt:   sr.StartedAt.Format("2006-01-02 15:04:05"),
		FinishedAt:  sr.FinishedAt.Format("2006-01-02 15:04:05"),
		MultiRunner: multi,
		MultiModel:  multiModel,
	}

	// Results rows
	for _, eval := range sr.Evals {
		if eval.Err != nil {
			d.Results = append(d.Results, htmlResultRow{
				Eval:   eval.EvalName,
				Status: "ERROR",
			})
			continue
		}
		for _, treat := range eval.Treatments {
			label := treatmentLabel(treat, multi, multiModel)
			for _, run := range treat.Runs {
				d.Results = append(d.Results, htmlResultRow{
					Eval:      eval.EvalName,
					Treatment: label,
					Sample:    fmt.Sprintf("%d", run.Sample),
					Status:    runStatus(run),
					Cost:      fmt.Sprintf("$%.4f", run.CostUSD),
					Duration:  formatDuration(run.DurationMs),
				})
			}
			if agg := treat.Aggregate; agg != nil && len(treat.Runs) >= 2 {
				d.Results = append(d.Results, buildHTMLAggRow(eval.EvalName, label, agg))
			}
		}
	}

	// Errors
	for _, eval := range sr.Evals {
		if eval.Err != nil {
			d.Errors = append(d.Errors, htmlError{
				Name:    eval.EvalName,
				ID:      eval.EvalID,
				Message: eval.Err.Error(),
			})
		}
	}

	// Skipped
	for _, eval := range sr.Evals {
		if len(eval.Skipped) == 0 {
			continue
		}
		group := htmlSkippedGroup{Name: eval.EvalName, ID: eval.EvalID}
		for _, s := range eval.Skipped {
			group.Entries = append(group.Entries, htmlSkippedEntry{Name: s.Name, Reason: s.Reason})
		}
		d.Skipped = append(d.Skipped, group)
	}

	// Rankings
	ranks := RankTreatments(sr, weights)
	if len(ranks) >= 2 {
		d.ShowRankings = true
		for _, r := range ranks {
			d.Rankings = append(d.Rankings, htmlRanking{
				Rank:           r.Rank,
				Name:           r.Name,
				Runner:         r.Runner,
				Model:          r.Model,
				CompositeScore: fmt.Sprintf("%.3f", r.CompositeScore),
				PassRate:       fmt.Sprintf("%.0f%%", r.PassRate*100),
				MedianCost:     fmt.Sprintf("$%.4f", r.MedianCostUSD),
				MedianDuration: formatDuration(r.MedianDuration),
			})
		}
	}

	return d
}

func buildHTMLAggRow(evalName, treatName string, agg *result.Aggregate) htmlResultRow {
	costRange := fmt.Sprintf("$%.4f [$%.4f\u2013$%.4f]", agg.MedianCostUSD, agg.MinCostUSD, agg.MaxCostUSD)
	durationRange := fmt.Sprintf("%s [%s\u2013%s]", formatDuration(agg.MedianDurationMs), formatDuration(agg.MinDurationMs), formatDuration(agg.MaxDurationMs))

	status := "\u2014"
	if agg.Pass != nil {
		if *agg.Pass {
			status = "PASS"
		} else {
			status = "FAIL"
		}
	}

	var cvInfo string
	if agg.CostCV != nil {
		cvInfo += fmt.Sprintf("cost_cv=%.1f%%", *agg.CostCV*100)
	}
	if agg.DurationCV != nil {
		if cvInfo != "" {
			cvInfo += " "
		}
		cvInfo += fmt.Sprintf("dur_cv=%.1f%%", *agg.DurationCV*100)
	}

	return htmlResultRow{
		Eval:      evalName,
		Treatment: treatName,
		Sample:    "agg",
		Status:    status,
		Cost:      costRange,
		Duration:  durationRange,
		IsAgg:     true,
		CVInfo:    cvInfo,
	}
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Eval Report</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; color: #1a1a1a; max-width: 1200px; margin: 0 auto; padding: 2rem; background: #fafafa; }
  h1 { font-size: 1.5rem; margin-bottom: 0.25rem; }
  h2 { font-size: 1.2rem; margin: 2rem 0 0.75rem; border-bottom: 2px solid #e5e7eb; padding-bottom: 0.25rem; }
  .meta { color: #6b7280; font-size: 0.875rem; margin-bottom: 1.5rem; }
  .description { margin-bottom: 1rem; color: #374151; }
  table { width: 100%; border-collapse: collapse; font-size: 0.875rem; background: #fff; border: 1px solid #e5e7eb; border-radius: 4px; }
  th { background: #f9fafb; text-align: left; padding: 0.5rem 0.75rem; border-bottom: 2px solid #e5e7eb; cursor: pointer; user-select: none; white-space: nowrap; }
  th:hover { background: #f3f4f6; }
  th::after { content: " \2195"; color: #9ca3af; font-size: 0.75rem; }
  td { padding: 0.5rem 0.75rem; border-bottom: 1px solid #f3f4f6; }
  tr:hover { background: #f9fafb; }
  tr.agg { font-weight: 600; background: #f9fafb; }
  .status-pass { color: #16a34a; font-weight: 600; }
  .status-fail, .status-failed { color: #dc2626; font-weight: 600; }
  .status-error { color: #dc2626; font-weight: 600; }
  .status-ok { color: #6b7280; }
  .cv-info { color: #9ca3af; font-size: 0.8em; margin-left: 0.5rem; }
  .errors-list { list-style: none; }
  .errors-list li { padding: 0.5rem 0; border-bottom: 1px solid #f3f4f6; }
  .errors-list .eval-name { font-weight: 600; }
  .errors-list .eval-id { color: #9ca3af; font-family: monospace; font-size: 0.85em; }
  .skipped-group { margin-bottom: 1rem; }
  .skipped-group h3 { font-size: 0.95rem; font-weight: 600; }
  .skipped-group .eval-id { color: #9ca3af; font-family: monospace; font-size: 0.85em; }
  .skipped-group ul { list-style: disc; margin-left: 1.5rem; margin-top: 0.25rem; }
</style>
</head>
<body>

<h1>Eval Report</h1>
{{if .Description}}<p class="description">{{.Description}}</p>{{end}}
<p class="meta"><strong>Started:</strong> {{.StartedAt}} &nbsp; <strong>Finished:</strong> {{.FinishedAt}}</p>

<h2>Results</h2>
<table>
<thead>
<tr>
  <th onclick="sortTable(this, 0)">Eval</th>
  <th onclick="sortTable(this, 1)">Treatment</th>
  <th onclick="sortTable(this, 2)">Sample</th>
  <th onclick="sortTable(this, 3)">Status</th>
  <th onclick="sortTable(this, 4)">Cost</th>
  <th onclick="sortTable(this, 5)">Duration</th>
</tr>
</thead>
<tbody>
{{range .Results}}<tr{{if .IsAgg}} class="agg"{{end}}>
  <td>{{.Eval}}</td>
  <td>{{.Treatment}}</td>
  <td>{{.Sample}}</td>
  <td><span class="status-{{.Status}}">{{.Status}}</span></td>
  <td>{{.Cost}}</td>
  <td>{{.Duration}}{{if .CVInfo}}<span class="cv-info">{{.CVInfo}}</span>{{end}}</td>
</tr>
{{end}}</tbody>
</table>

{{if .Errors}}
<h2>Errors</h2>
<ul class="errors-list">
{{range .Errors}}<li><span class="eval-name">{{.Name}}</span> <span class="eval-id">({{.ID}})</span>: {{.Message}}</li>
{{end}}</ul>
{{end}}

{{if .Skipped}}
<h2>Skipped Treatments</h2>
{{range .Skipped}}<div class="skipped-group">
<h3>{{.Name}} <span class="eval-id">({{.ID}})</span></h3>
<ul>
{{range .Entries}}<li>{{.Name}} — {{.Reason}}</li>
{{end}}</ul>
</div>
{{end}}
{{end}}

{{if .ShowRankings}}
<h2>Rankings</h2>
<table>
<thead>
<tr>
  <th onclick="sortTable(this, 0)">Rank</th>
  <th onclick="sortTable(this, 1)">Treatment</th>
  {{if .MultiRunner}}<th onclick="sortTable(this, 2)">Runner</th>{{end}}
  {{if .MultiModel}}<th onclick="sortTable(this, {{if .MultiRunner}}3{{else}}2{{end}})">Model</th>{{end}}
  <th onclick="sortTable(this, {{if .MultiRunner}}{{if .MultiModel}}4{{else}}3{{end}}{{else}}{{if .MultiModel}}3{{else}}2{{end}}{{end}})">Score</th>
  <th onclick="sortTable(this, {{if .MultiRunner}}{{if .MultiModel}}5{{else}}4{{end}}{{else}}{{if .MultiModel}}4{{else}}3{{end}}{{end}})">Pass Rate</th>
  <th onclick="sortTable(this, {{if .MultiRunner}}{{if .MultiModel}}6{{else}}5{{end}}{{else}}{{if .MultiModel}}5{{else}}4{{end}}{{end}})">Median Cost</th>
  <th onclick="sortTable(this, {{if .MultiRunner}}{{if .MultiModel}}7{{else}}6{{end}}{{else}}{{if .MultiModel}}6{{else}}5{{end}}{{end}})">Median Duration</th>
</tr>
</thead>
<tbody>
{{range .Rankings}}<tr>
  <td>#{{.Rank}}</td>
  <td>{{.Name}}</td>
  {{if $.MultiRunner}}<td>{{.Runner}}</td>{{end}}
  {{if $.MultiModel}}<td>{{.Model}}</td>{{end}}
  <td>{{.CompositeScore}}</td>
  <td>{{.PassRate}}</td>
  <td>{{.MedianCost}}</td>
  <td>{{.MedianDuration}}</td>
</tr>
{{end}}</tbody>
</table>
{{end}}

<script>
function sortTable(th, colIdx) {
  var table = th.closest("table");
  var tbody = table.querySelector("tbody");
  var rows = Array.from(tbody.querySelectorAll("tr"));
  var asc = th.dataset.sortDir !== "asc";
  th.dataset.sortDir = asc ? "asc" : "desc";
  // Reset other headers in same table
  Array.from(table.querySelectorAll("th")).forEach(function(h) { if (h !== th) delete h.dataset.sortDir; });
  rows.sort(function(a, b) {
    var av = a.cells[colIdx].textContent.trim();
    var bv = b.cells[colIdx].textContent.trim();
    // Try numeric comparison (strip $, #, %, s, ms)
    var an = parseFloat(av.replace(/[$#%sms,]/g, ""));
    var bn = parseFloat(bv.replace(/[$#%sms,]/g, ""));
    if (!isNaN(an) && !isNaN(bn)) return asc ? an - bn : bn - an;
    return asc ? av.localeCompare(bv) : bv.localeCompare(av);
  });
  rows.forEach(function(r) { tbody.appendChild(r); });
}
</script>
</body>
</html>`
