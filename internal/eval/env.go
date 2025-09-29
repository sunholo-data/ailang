package eval

// Environment represents a variable environment
type Environment struct {
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
	e.values[name] = value
}

// Get gets a value from the environment
func (e *Environment) Get(name string) (Value, bool) {
	if value, ok := e.values[name]; ok {
		return value, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, false
}

// Clone creates a deep copy of the environment
func (e *Environment) Clone() *Environment {
	newEnv := &Environment{
		values: make(map[string]Value),
		parent: e.parent,
	}
	for k, v := range e.values {
		newEnv.values[k] = v
	}
	return newEnv
}

// Extend creates a new child environment with a binding
func (e *Environment) Extend(name string, value Value) *Environment {
	child := e.NewChildEnvironment()
	child.Set(name, value)
	return child
}

// New creates a new evaluator with built-in functions (for compatibility)
func New() *SimpleEvaluator {
	return NewSimple()
}
