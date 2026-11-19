package types

import (
	"fmt"
	"slices"
	"sync/atomic"
)

// envScanBindings counts bindings visited by the environment free-variable
// walks (collectFreeTypeVars, collectFreeRowVars, collectFreeEffectRowVars and
// implicitRowsAbove). It is a deterministic measure of type-checker work used
// by the env-scan linearity test (M-TYPECHECK-ENV-SCAN-COST); wall clock and
// RSS are machine-dependent, this is not.
var envScanBindings atomic.Uint64

// EnvScanWork returns the bindings visited by environment free-variable walks
// since the last ResetEnvScanWork.
func EnvScanWork() uint64 { return envScanBindings.Load() }

// ResetEnvScanWork zeroes the counter read by EnvScanWork.
func ResetEnvScanWork() { envScanBindings.Store(0) }

// TypeEnv represents a type environment mapping names to types or schemes
type TypeEnv struct {
	bindings map[string]interface{} // Can be Type or *Scheme; written in place ONLY by BindScheme/BindType, which invalidate the caches below
	parent   *TypeEnv

	// Memoised free-variable sets (env_cache.go).
	own         *envVarSets // this frame's bindings only
	chain       *envVarSets // this frame and all ancestors
	chainParent uint64      // stamp of the parent chain this chain was built from
	chainEpoch  uint64      // envEpoch when chain was last validated
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
	env.invalidateVarCaches()
}

// BindType adds a type binding to the environment (for REPL persistence)
// This mutates the environment in-place, unlike Extend which creates a child.
// Use this only when you need top-level bindings to persist (e.g., REPL).
func (env *TypeEnv) BindType(name string, typ Type) {
	env.bindings[name] = typ
	env.invalidateVarCaches()
}

// FreeTypeVars returns all free type variables in the environment
// The result is the caller's to modify.
func (env *TypeEnv) FreeTypeVars() map[string]bool {
	return copyVarSet(env.ownedVars().types)
}

// FreeTypeVarsAbove returns the vars free in the bindings of the frames
// between env (inclusive) and base (exclusive): the bindings a declaration
// pushed on top of its base env. It returns an empty set when base is nil or
// not an ancestor of env, so callers only ever withhold MORE, never less.
func (env *TypeEnv) FreeTypeVarsAbove(base *TypeEnv) map[string]bool {
	free := make(map[string]bool)
	if base == nil {
		return free
	}
	frames, ok := env.framesAbove(base)
	if !ok {
		return free // base is not an ancestor
	}
	for _, f := range frames {
		for v := range f.ownSets().types {
			free[v] = true
		}
	}
	return free
}

func (env *TypeEnv) collectFreeTypeVars(free map[string]bool) {
	env.collectFreeVarSets(free, nil, nil)
}

// collectFreeVarSets adds the variables free in the bindings of env and its
// ancestors to the given sets (a nil set is skipped): type variables to types,
// row variables of every kind to rows, effect-kinded row variables to
// effectRows. A scheme's own quantifiers are not free: TypeVars for type
// variables, RowVars for row variables.
//
// It is built on varWalker, the traversal generalisation uses for the type
// being generalised (free_vars.go), so "free in the environment" and
// "generalisable in this type" cannot disagree about a type form. They did:
// this walk had no case for records, record rows or type applications, so a
// variable owned by a parameter of type {name: a} or Option[a] was generalised
// by an inner let (PR #2209 review). One walk per binding fills all three sets.
func (env *TypeEnv) collectFreeVarSets(types, rows, effectRows map[string]bool) {
	for _, binding := range env.bindings {
		envScanBindings.Add(1)
		var t Type
		var typeQuant, rowQuant []string
		switch b := binding.(type) {
		case Type:
			t = b
		case *Scheme:
			t, typeQuant, rowQuant = b.Type, b.TypeVars, b.RowVars
		default:
			continue
		}
		forEachVar(t, false, func(name string, s varSort, row bool) {
			switch {
			case name == "":
			case !row:
				if types != nil && !slices.Contains(typeQuant, name) {
					types[name] = true
				}
			case !slices.Contains(rowQuant, name):
				if rows != nil {
					rows[name] = true
				}
				if effectRows != nil && s == sortEffectRow {
					effectRows[name] = true
				}
			}
		})
	}
	if env.parent != nil {
		env.parent.collectFreeVarSets(types, rows, effectRows)
	}
}

// FreeRowVars returns all free row variables in the environment
// The result is the caller's to modify.
func (env *TypeEnv) FreeRowVars() map[string]bool {
	return copyVarSet(env.ownedVars().rows)
}

// collectFreeRowVars adds the row variables of every kind (record and effect)
// that occur free in the environment.
func (env *TypeEnv) collectFreeRowVars(free map[string]bool) {
	env.collectFreeVarSets(nil, free, nil)
}

// FreeEffectRowVars returns every free EffectRow-kinded row variable reachable
// through the COMPLETE type of each binding in the environment (descending into
// nested function/collection/record positions), minus each scheme's own
// quantified row variables.
//
// M-EFFECT-ROW-SHOW-INTERP (#386) Section B: generalizeWithConstraints uses this
// as the HM side condition for rows — a row variable free in the environment
// (owned by an enclosing binder) must NOT be generalized, mirroring the existing
// type-variable withhold logic.
// The result is the caller's to modify.
func (env *TypeEnv) FreeEffectRowVars() map[string]bool {
	return copyVarSet(env.ownedVars().effectRows)
}

func (env *TypeEnv) collectFreeEffectRowVars(free map[string]bool) {
	env.collectFreeVarSets(nil, nil, free)
}
