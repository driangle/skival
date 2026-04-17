// Package registry maps runner names to factory functions, enabling
// dynamic runner construction from suite configuration.
package registry

import (
	"fmt"

	agentrunner "github.com/driangle/agentrunner/go"
)

// Factory creates a Runner from runner-specific configuration.
type Factory func(config map[string]any) (agentrunner.Runner, error)

// Registry holds named runner factories.
type Registry struct {
	factories map[string]Factory
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register adds a factory under the given name.
func (r *Registry) Register(name string, f Factory) {
	r.factories[name] = f
}

// Create looks up the factory for name and calls it with config.
// It returns an error if no factory is registered for name.
func (r *Registry) Create(name string, config map[string]any) (agentrunner.Runner, error) {
	f, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("unknown runner %q", name)
	}
	return f(config)
}
