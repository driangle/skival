package report

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
  .comparison-group { margin-bottom: 1.5rem; }
  .comparison-group h3 { font-size: 0.95rem; font-weight: 600; margin-bottom: 0.5rem; }
  .comparison-group .eval-id { color: #9ca3af; font-family: monospace; font-size: 0.85em; }
  .comparison-group .judge-model { color: #9ca3af; font-size: 0.8em; }
  .comparison-skipped { color: #6b7280; font-size: 0.875rem; font-style: italic; }
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
  <th onclick="sortTable(this, 1)">Variant</th>
  <th onclick="sortTable(this, 2)">Sample</th>
  <th onclick="sortTable(this, 3)">Status</th>
  <th onclick="sortTable(this, 4)">Cost</th>
  <th onclick="sortTable(this, 5)">Duration</th>
</tr>
</thead>
<tbody>
{{range .Results}}<tr{{if .IsAgg}} class="agg"{{end}}>
  <td>{{.Eval}}</td>
  <td>{{.Variant}}</td>
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
<h2>Skipped Variants</h2>
{{range .Skipped}}<div class="skipped-group">
<h3>{{.Name}} <span class="eval-id">({{.ID}})</span></h3>
<ul>
{{range .Entries}}<li>{{.Name}} — {{.Reason}}</li>
{{end}}</ul>
</div>
{{end}}
{{end}}

{{if .Comparisons}}
<h2>Comparative Quality</h2>
{{range .Comparisons}}<div class="comparison-group">
<h3>{{.Name}} <span class="eval-id">({{.ID}})</span>{{if .Model}} <span class="judge-model">judge: {{.Model}}</span>{{end}}</h3>
{{if .Scores}}<table>
<thead>
<tr>
  <th onclick="sortTable(this, this.cellIndex)">Variant</th>
  <th onclick="sortTable(this, this.cellIndex)">Rating</th>
  <th onclick="sortTable(this, this.cellIndex)">Score</th>
  <th onclick="sortTable(this, this.cellIndex)">Reason</th>
</tr>
</thead>
<tbody>
{{range .Scores}}<tr>
  <td>{{.Variant}}</td>
  <td>{{.Rating}}</td>
  <td>{{.Score}}</td>
  <td>{{.Reason}}</td>
</tr>
{{end}}</tbody>
</table>
{{else}}<p class="comparison-skipped">{{if .Skipped}}{{.Skipped}}{{else}}no scores produced{{end}}</p>
{{end}}</div>
{{end}}
{{end}}

{{if .ShowRankings}}
<h2>Rankings</h2>
<table>
<thead>
<tr>
  <th onclick="sortTable(this, 0)">Rank</th>
  <th onclick="sortTable(this, 1)">Variant</th>
  {{if .MultiRunner}}<th onclick="sortTable(this, 2)">Runner</th>{{end}}
  {{if .MultiModel}}<th onclick="sortTable(this, {{if .MultiRunner}}3{{else}}2{{end}})">Model</th>{{end}}
  <th onclick="sortTable(this, {{if .MultiRunner}}{{if .MultiModel}}4{{else}}3{{end}}{{else}}{{if .MultiModel}}3{{else}}2{{end}}{{end}})">Score</th>
  <th onclick="sortTable(this, {{if .MultiRunner}}{{if .MultiModel}}5{{else}}4{{end}}{{else}}{{if .MultiModel}}4{{else}}3{{end}}{{end}})">Pass Rate</th>
  <th onclick="sortTable(this, {{if .MultiRunner}}{{if .MultiModel}}6{{else}}5{{end}}{{else}}{{if .MultiModel}}5{{else}}4{{end}}{{end}})">Median Cost</th>
  <th onclick="sortTable(this, {{if .MultiRunner}}{{if .MultiModel}}7{{else}}6{{end}}{{else}}{{if .MultiModel}}6{{else}}5{{end}}{{end}})">Median Duration</th>
  {{if .ShowQuality}}<th onclick="sortTable(this, this.cellIndex)">Quality</th>{{end}}
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
  {{if $.ShowQuality}}<td>{{.QualityScore}}</td>{{end}}
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
