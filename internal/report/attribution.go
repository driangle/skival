package report

// Attribution marks every report as skival-generated. Each format renders it in
// its native idiom: a markdown link, an HTML anchor (in the template footer), or
// a structured JSON object.
const (
	attributionTool     = "skival"
	attributionURL      = "https://github.com/driangle/skival"
	attributionMarkdown = "Made with [skival](https://github.com/driangle/skival)"
)

// jsonAttribution stamps machine-readable reports with the tool that produced them.
type jsonAttribution struct {
	Tool string `json:"tool"`
	URL  string `json:"url"`
}

func attribution() jsonAttribution {
	return jsonAttribution{Tool: attributionTool, URL: attributionURL}
}
