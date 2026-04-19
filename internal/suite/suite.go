package suite

// Suite is the top-level configuration for a benchmark suite.
type Suite struct {
	Version     int       `yaml:"version"`
	Description string    `yaml:"description"`
	Defaults    Defaults  `yaml:"defaults"`
	Ranking     *Ranking  `yaml:"ranking,omitempty"`
	Evals       []Eval    `yaml:"evals"`
}

// Ranking configures how treatments are scored and ranked.
type Ranking struct {
	Weights RankingWeights `yaml:"weights"`
}

// RankingWeights defines the relative importance of each metric in the composite score.
// All weights must be >= 0 and sum to 1.0.
type RankingWeights struct {
	Correctness float64 `yaml:"correctness"`
	Cost        float64 `yaml:"cost"`
	Duration    float64 `yaml:"duration"`
}
// Retry configures retry behavior for failed sample runs.
type Retry struct {
	MaxAttempts *int   `yaml:"max_attempts"` // total attempts including the first (default: 1)
	Backoff     string `yaml:"backoff"`      // "fixed" or "exponential" (default: "fixed")
	Delay       string `yaml:"delay"`        // base delay between retries (default: "2s")
	On          string `yaml:"on"`           // "transient" or "all" (default: "transient")
}

// Defaults defines suite-level defaults applied to evals that don't override them.
type Defaults struct {
	Samples      *int           `yaml:"samples"`
	Timeout      *int           `yaml:"timeout"`
	Parallel     *int           `yaml:"parallel"`
	Model        string         `yaml:"model"`
	JudgeModel   string         `yaml:"judge_model"`
	Runner       string         `yaml:"runner"`
	RunnerConfig map[string]any `yaml:"runner_config"`
	Retry        *Retry         `yaml:"retry"`
}

// Eval defines a single evaluation within a suite.
type Eval struct {
	File         string         `yaml:"file,omitempty"`
	ID           string         `yaml:"id"`
	Name         string         `yaml:"name"`
	Prompt       string         `yaml:"prompt"`
	Dir          string         `yaml:"dir"`
	Isolate      bool           `yaml:"isolate"`
	Complexity   string         `yaml:"complexity"`
	Samples      *int           `yaml:"samples"`
	Timeout      *int           `yaml:"timeout"`
	Parallel     *int           `yaml:"parallel"`
	Model        string         `yaml:"model"`
	JudgeModel   string         `yaml:"judge_model,omitempty"`
	Runner       string         `yaml:"runner"`
	RunnerConfig map[string]any `yaml:"runner_config"`
	Setup        Setup          `yaml:"setup"`
	Verify       []VerifyStep   `yaml:"verify,omitempty"`
	Correctness  Correctness    `yaml:"correctness"` // Deprecated: use Verify
	Retry        *Retry         `yaml:"retry"`
	Matrix       *Matrix        `yaml:"matrix,omitempty"`
	Treatments   Treatments     `yaml:"treatments"`
}

// VerifyStep defines a single verification step.
// Type determines which fields are relevant.
type VerifyStep struct {
	Type string `yaml:"type"`
	Name string `yaml:"name,omitempty"`

	// check, check_output, command
	Run string `yaml:"run,omitempty"`

	// command
	Exits          *int   `yaml:"exits,omitempty"`
	StdoutContains string `yaml:"stdout_contains,omitempty"`

	// output_contains
	Values []string `yaml:"values,omitempty"`

	// file_contains
	Path     string `yaml:"path,omitempty"`
	Contains string `yaml:"contains,omitempty"`
	Exists   *bool  `yaml:"exists,omitempty"`

	// http_check
	URL          string `yaml:"url,omitempty"`
	Method       string `yaml:"method,omitempty"`
	Status       *int   `yaml:"status,omitempty"`
	BodyContains string `yaml:"body_contains,omitempty"`

	// tcp_check
	Host string `yaml:"host,omitempty"`
	Port int    `yaml:"port,omitempty"`

	// judge
	Criteria []string `yaml:"criteria,omitempty"`
}

// Setup defines lifecycle hooks for an eval.
type Setup struct {
	Before string `yaml:"before"`
	After  string `yaml:"after"`
	Reset  string `yaml:"reset"`
}

// Correctness defines how to verify an eval's output.
type Correctness struct {
	Check        string           `yaml:"check"`
	AgentExitsOK *bool            `yaml:"agent_exits_ok"`
	Output       Output           `yaml:"output"`
	CheckOutput  string           `yaml:"check_output"`
	State        []StateAssertion `yaml:"state"`
	Probes       []Probe          `yaml:"probes"`
	Judge        []string         `yaml:"judge"`
	JudgeModel   string           `yaml:"judge_model"`
}

// Output defines structured output matching criteria.
type Output struct {
	Contains []string `yaml:"contains"`
}

// StateAssertion defines an HTTP assertion to check after execution.
// Deprecated: use Probe with HTTPProbe instead.
type StateAssertion struct {
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
	Expect string `yaml:"expect"`
}

// Probe defines a single typed probe. Exactly one type key must be set.
type Probe struct {
	HTTP    *HTTPProbe    `yaml:"http,omitempty"`
	File    *FileProbe    `yaml:"file,omitempty"`
	Command *CommandProbe `yaml:"command,omitempty"`
	TCP     *TCPProbe     `yaml:"tcp,omitempty"`
}

// HTTPProbe checks an HTTP endpoint.
type HTTPProbe struct {
	URL    string          `yaml:"url"`
	Method string          `yaml:"method"`
	Assert HTTPProbeAssert `yaml:"assert"`
}

// HTTPProbeAssert defines assertions for an HTTP probe.
type HTTPProbeAssert struct {
	Status       *int   `yaml:"status,omitempty"`
	BodyContains string `yaml:"body_contains,omitempty"`
}

// FileProbe checks a file on disk.
type FileProbe struct {
	Path   string          `yaml:"path"`
	Assert FileProbeAssert `yaml:"assert"`
}

// FileProbeAssert defines assertions for a file probe.
type FileProbeAssert struct {
	Exists   *bool  `yaml:"exists,omitempty"`
	Contains string `yaml:"contains,omitempty"`
}

// CommandProbe runs a shell command and checks its output.
type CommandProbe struct {
	Run    string             `yaml:"run"`
	Assert CommandProbeAssert `yaml:"assert"`
}

// CommandProbeAssert defines assertions for a command probe.
type CommandProbeAssert struct {
	Exits          *int   `yaml:"exits,omitempty"`
	StdoutContains string `yaml:"stdout_contains,omitempty"`
}

// TCPProbe checks TCP connectivity.
type TCPProbe struct {
	Host   string         `yaml:"host"`
	Port   int            `yaml:"port"`
	Assert TCPProbeAssert `yaml:"assert"`
}

// TCPProbeAssert defines assertions for a TCP probe.
type TCPProbeAssert struct {
	Open *bool `yaml:"open,omitempty"`
}

// Matrix defines dimensions for generating treatments from a cartesian product.
// Each dimension has a name and a list of values. The cartesian product of all
// dimensions produces the full set of treatments.
type Matrix struct {
	Dimensions []MatrixDimension `yaml:"dimensions"`
}

// MatrixDimension defines a single axis of variation in a matrix.
type MatrixDimension struct {
	Name   string              `yaml:"name"`
	Values []MatrixDimensionValue `yaml:"values"`
}

// MatrixDimensionValue defines one value within a matrix dimension.
// Label is required for naming. The remaining fields override the
// corresponding treatment fields.
type MatrixDimensionValue struct {
	Label        string            `yaml:"label"`
	Prompt       string            `yaml:"prompt,omitempty"`
	ConfigDir    string            `yaml:"config_dir,omitempty"`
	Model        string            `yaml:"model,omitempty"`
	Runner       string            `yaml:"runner,omitempty"`
	RunnerConfig map[string]any    `yaml:"runner_config,omitempty"`
	Skill        string            `yaml:"skill,omitempty"`
	Skills       []string          `yaml:"skills,omitempty"`
	Env          map[string]string `yaml:"env,omitempty"`
}

// Treatments defines the control and variation treatments for an eval.
type Treatments struct {
	Control    Treatment   `yaml:"control"`
	Variations []Treatment `yaml:"variations"`
}

// Treatment defines a single treatment configuration.
type Treatment struct {
	Name            string            `yaml:"name"`
	Prompt          string            `yaml:"prompt"`
	Dir             string            `yaml:"dir"`
	ConfigDir       string            `yaml:"config_dir"`
	Model           string            `yaml:"model"`
	Runner          string            `yaml:"runner"`
	RunnerConfig    map[string]any    `yaml:"runner_config"`
	Skill           string            `yaml:"skill"`
	Skills          []string          `yaml:"skills"`
	AllowedTools    []string          `yaml:"allowed_tools,omitempty"` // Deprecated: use runner_config.allowed_tools
	Env             map[string]string `yaml:"env"`
	Retry           *Retry            `yaml:"retry"`
	DimensionValues map[string]string `yaml:"-"` // populated by matrix expansion, not parsed from YAML
}
