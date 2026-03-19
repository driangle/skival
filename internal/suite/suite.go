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
	Samples *int   `yaml:"samples"`
	Timeout *int   `yaml:"timeout"`
	Model   string `yaml:"model"`
}

// Eval defines a single evaluation within a suite.
type Eval struct {
	File        string      `yaml:"file,omitempty"`
	ID          string      `yaml:"id"`
	Name        string      `yaml:"name"`
	Prompt      string      `yaml:"prompt"`
	Dir         string      `yaml:"dir"`
	Complexity  string      `yaml:"complexity"`
	Samples     *int        `yaml:"samples"`
	Timeout     *int        `yaml:"timeout"`
	Model       string      `yaml:"model"`
	Setup       Setup       `yaml:"setup"`
	Correctness Correctness `yaml:"correctness"`
	Treatments  Treatments  `yaml:"treatments"`
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
}

// StateAssertion defines an HTTP assertion to check after execution.
type StateAssertion struct {
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
	Expect string `yaml:"expect"`
}

// Treatments defines the control and variation treatments for an eval.
type Treatments struct {
	Control    Treatment   `yaml:"control"`
	Variations []Treatment `yaml:"variations"`
}

// Treatment defines a single treatment configuration.
type Treatment struct {
	Name         string            `yaml:"name"`
	Dir          string            `yaml:"dir"`
	Model        string            `yaml:"model"`
	Skill        string            `yaml:"skill"`
	AllowedTools []string          `yaml:"allowed_tools"`
	Env          map[string]string `yaml:"env"`
}
