package eval

import "sync"

// Environment represents a variable environment.
// Thread-safe: concurrent reads are protected by RWMutex to support
// serve-api's per-request evaluator forking, where module-level
// environments are shared (read) across goroutines while per-request
// environments are written to by individual goroutines.
type Environment struct {
	mu     sync.RWMutex
	values map[string]Value
	parent *Environment
}

// NewEnvironment creates a new environment
func NewEnvironment() *Environment {
	return &Environment{
		values: make(map[string]Value),
		parent: nil,
	}
}

// NewChildEnvironment creates a child environment
func (e *Environment) NewChildEnvironment() *Environment {
	return &Environment{
		values: make(map[string]Value),
		parent: e,
	}
}

// Set sets a value in the environment
func (e *Environment) Set(name string, value Value) {
	e.mu.Lock()
	e.values[name] = value
	e.mu.Unlock()
}

// Get gets a value from the environment
func (e *Environment) Get(name string) (Value, bool) {
	e.mu.RLock()
	value, ok := e.values[name]
	e.mu.RUnlock()
	if ok {
		return value, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, false
}

// Clone creates a shallow copy of the environment (copies values map, shares parent)
func (e *Environment) Clone() *Environment {
	e.mu.RLock()
	newEnv := &Environment{
		values: make(map[string]Value, len(e.values)),
		parent: e.parent,
	}
	for k, v := range e.values {
		newEnv.values[k] = v
	}
	e.mu.RUnlock()
	return newEnv
}

// Extend creates a new child environment with a binding
func (e *Environment) Extend(name string, value Value) *Environment {
	child := e.NewChildEnvironment()
	child.Set(name, value)
	return child
}

// GetAllBindings returns all bindings in this environment and its parents
func (e *Environment) GetAllBindings() map[string]Value {
	result := make(map[string]Value)

	// Collect from parent first (so current env can override)
	if e.parent != nil {
		for k, v := range e.parent.GetAllBindings() {
			result[k] = v
		}
	}

	// Add current env bindings (overriding parent if needed)
	e.mu.RLock()
	for k, v := range e.values {
		result[k] = v
	}
	e.mu.RUnlock()

	return result
}

// New creates a new evaluator with built-in functions (for compatibility)
func New() *SimpleEvaluator {
	return NewSimple()
}
