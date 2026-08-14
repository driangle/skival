package compare

// Attribution marks every comparison report as skival-generated: a markdown link
// in the human-readable output, and a structured object in the JSON output.
const (
	attributionTool     = "skival"
	attributionURL      = "https://github.com/driangle/skival"
	attributionMarkdown = "Made with [skival](https://github.com/driangle/skival)"
)

// jsonAttribution stamps the JSON comparison report with the tool that produced it.
type jsonAttribution struct {
	Tool string `json:"tool"`
	URL  string `json:"url"`
}

func attribution() jsonAttribution {
	return jsonAttribution{Tool: attributionTool, URL: attributionURL}
}
