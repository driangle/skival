package suite

// Suite is the top-level configuration for a benchmark suite.
type Suite struct {
	Version     int       `yaml:"version"`
	Description string    `yaml:"description"`
	Defaults    Defaults  `yaml:"defaults"`
	Evals       []Eval    `yaml:"evals"`
}

// Defaults defines suite-level defaults applied to evals that don't override them.
type Defaults struct {
	Samples      *int           `yaml:"samples"`
	Timeout      *int           `yaml:"timeout"`
	Parallel     *int           `yaml:"parallel"`
	Model        string         `yaml:"model"`
	Runner       string         `yaml:"runner"`
	RunnerConfig map[string]any `yaml:"runner_config"`
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
	Runner       string         `yaml:"runner"`
	RunnerConfig map[string]any `yaml:"runner_config"`
	Setup        Setup          `yaml:"setup"`
	Correctness  Correctness    `yaml:"correctness"`
	Matrix       *Matrix        `yaml:"matrix,omitempty"`
	Treatments   Treatments     `yaml:"treatments"`
}

// Setup defines lifecycle hooks for an eval.
type Setup struct {
	Before string `yaml:"before"`
	After  string `yaml:"after"`
	Reset  string `yaml:"reset"`
}

// Correctness defines how to verify an eval's output.
type Correctness struct {
	Compiles       *bool            `yaml:"compiles"`
	Execute        *bool            `yaml:"execute"`
	ExpectedOutput []string         `yaml:"expected_output"`
	Script         string           `yaml:"script"`
	State          []StateAssertion `yaml:"state"`
	Judge          []string         `yaml:"judge"`
}

// StateAssertion defines an HTTP assertion to check after execution.
type StateAssertion struct {
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
	Expect string `yaml:"expect"`
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
	DimensionValues map[string]string `yaml:"-"` // populated by matrix expansion, not parsed from YAML
}
