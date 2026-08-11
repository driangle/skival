package report

import (
	_ "embed"
	"html/template"
)

// The HTML report is assembled from three embedded assets. Keeping the markup,
// styles, and script in their own files (rather than one Go string constant)
// makes them editable with proper syntax highlighting while still compiling
// into the binary, so the rendered report stays a single self-contained file.

//go:embed templates/report.html.tmpl
var htmlTemplate string

//go:embed templates/report.css
var htmlCSS string

//go:embed templates/report.js
var htmlJS string

// reportCSS and reportJS mark the embedded assets as trusted so html/template
// inlines them verbatim into their <style>/<script> contexts instead of escaping.
var (
	reportCSS = template.CSS(htmlCSS)
	reportJS  = template.JS(htmlJS)
)

// reportTmpl is parsed once at init; the embedded markup never changes at
// runtime, so there is no reason to recompile it on every report.
var reportTmpl = template.Must(template.New("report").Parse(htmlTemplate))
