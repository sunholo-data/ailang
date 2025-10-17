package types

import (
	"fmt"
)

// TypeEnv represents a type environment mapping names to types or schemes
type TypeEnv struct {
	bindings map[string]interface{} // Can be Type or *Scheme
	parent   *TypeEnv
}

// NewTypeEnv creates a new empty type environment
func NewTypeEnv() *TypeEnv {
	return &TypeEnv{
		bindings: make(map[string]interface{}),
		parent:   nil,
	}
}

// builtinEnvFactory is set by internal/link package during init to avoid import cycles
var builtinEnvFactory func() *TypeEnv

// SetBuiltinEnvFactory allows the link package to provide the builtin env implementation
func SetBuiltinEnvFactory(factory func() *TypeEnv) {
	builtinEnvFactory = factory
}

// NewTypeEnvWithBuiltins creates a type environment with builtin functions.
//
// This delegates to internal/link/env_seed.go which seeds the environment
// from the linker's $builtin interface, ensuring all 49 spec-registered
// builtins (with correct effect rows) are visible to the typechecker.
func NewTypeEnvWithBuiltins() *TypeEnv {
	if builtinEnvFactory == nil {
		// Fallback if link package hasn't initialized yet (shouldn't happen in normal use)
		panic("NewTypeEnvWithBuiltins called before link package initialized")
	}
	return builtinEnvFactory()
}

// Extend creates a new environment with an additional binding
func (env *TypeEnv) Extend(name string, typ Type) *TypeEnv {
	newEnv := &TypeEnv{
		bindings: make(map[string]interface{}),
		parent:   env,
	}
	newEnv.bindings[name] = typ
	return newEnv
}

// ExtendScheme creates a new environment with a scheme binding
func (env *TypeEnv) ExtendScheme(name string, scheme *Scheme) *TypeEnv {
	newEnv := &TypeEnv{
		bindings: make(map[string]interface{}),
		parent:   env,
	}
	newEnv.bindings[name] = scheme
	return newEnv
}

// Lookup finds a type or scheme in the environment
func (env *TypeEnv) Lookup(name string) (interface{}, error) {
	if binding, ok := env.bindings[name]; ok {
		return binding, nil
	}
	if env.parent != nil {
		return env.parent.Lookup(name)
	}
	return nil, fmt.Errorf("unbound variable: %s", name)
}

// BindScheme adds a scheme binding to the environment (for REPL persistence)
// This mutates the environment in-place, unlike Extend which creates a child.
// Use this only when you need top-level bindings to persist (e.g., REPL).
func (env *TypeEnv) BindScheme(name string, scheme *Scheme) {
	env.bindings[name] = scheme
}

// BindType adds a type binding to the environment (for REPL persistence)
// This mutates the environment in-place, unlike Extend which creates a child.
// Use this only when you need top-level bindings to persist (e.g., REPL).
func (env *TypeEnv) BindType(name string, typ Type) {
	env.bindings[name] = typ
}

// FreeTypeVars returns all free type variables in the environment
func (env *TypeEnv) FreeTypeVars() map[string]bool {
	free := make(map[string]bool)
	env.collectFreeTypeVars(free)
	return free
}

func (env *TypeEnv) collectFreeTypeVars(free map[string]bool) {
	for _, binding := range env.bindings {
		switch b := binding.(type) {
		case Type:
			collectFreeTypeVars(b, free)
		case *Scheme:
			// Scheme's quantified variables are not free
			schemeVars := make(map[string]bool)
			for _, v := range b.TypeVars {
				schemeVars[v] = true
			}
			typeFree := freeTypeVars(b.Type)
			for v := range typeFree {
				if !schemeVars[v] {
					free[v] = true
				}
			}
		}
	}
	if env.parent != nil {
		env.parent.collectFreeTypeVars(free)
	}
}

// FreeRowVars returns all free row variables in the environment
func (env *TypeEnv) FreeRowVars() map[string]bool {
	free := make(map[string]bool)
	env.collectFreeRowVars(free)
	return free
}

func (env *TypeEnv) collectFreeRowVars(free map[string]bool) {
	for _, binding := range env.bindings {
		switch b := binding.(type) {
		case Type:
			// Check for row variables in types
			if fn, ok := b.(*TFunc2); ok && fn.EffectRow != nil {
				collectFreeRowVars(fn.EffectRow, free)
			}
			if rec, ok := b.(*TRecord2); ok && rec.Row != nil {
				collectFreeRowVars(rec.Row, free)
			}
		case *Scheme:
			// Scheme's quantified row variables are not free
			schemeVars := make(map[string]bool)
			for _, v := range b.RowVars {
				schemeVars[v] = true
			}
			// Check the type for row variables
			if fn, ok := b.Type.(*TFunc2); ok && fn.EffectRow != nil {
				rowFree := freeRowVars(fn.EffectRow)
				for v := range rowFree {
					if !schemeVars[v] {
						free[v] = true
					}
				}
			}
		}
	}
	if env.parent != nil {
		env.parent.collectFreeRowVars(free)
	}
}
