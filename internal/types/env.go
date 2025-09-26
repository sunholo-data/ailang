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

// NewTypeEnvWithBuiltins creates a type environment with builtin functions
func NewTypeEnvWithBuiltins() *TypeEnv {
	env := NewTypeEnv()
	
	// Add builtin functions
	// print : ∀α. α -> () ! {IO}
	env.bindBuiltin("print", &Scheme{
		TypeVars: []string{"α"},
		Type: &TFunc2{
			Params: []Type{&TVar2{Name: "α", Kind: Star}},
			EffectRow: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit},
				Tail:   nil,
			},
			Return: TUnit,
		},
	})

	// readFile : string -> string ! {FS}
	env.bindBuiltin("readFile", &Scheme{
		Type: &TFunc2{
			Params: []Type{TString},
			EffectRow: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"FS": TUnit},
				Tail:   nil,
			},
			Return: TString,
		},
	})

	// writeFile : string -> string -> () ! {FS}
	env.bindBuiltin("writeFile", &Scheme{
		Type: &TFunc2{
			Params: []Type{TString, TString},
			EffectRow: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"FS": TUnit},
				Tail:   nil,
			},
			Return: TUnit,
		},
	})

	// httpGet : string -> string ! {Net}
	env.bindBuiltin("httpGet", &Scheme{
		Type: &TFunc2{
			Params: []Type{TString},
			EffectRow: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"Net": TUnit},
				Tail:   nil,
			},
			Return: TString,
		},
	})

	// random : () -> float ! {Rand}
	env.bindBuiltin("random", &Scheme{
		Type: &TFunc2{
			Params: []Type{TUnit},
			EffectRow: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"Rand": TUnit},
				Tail:   nil,
			},
			Return: TFloat,
		},
	})

	// trace : ∀α. string -> α -> α ! {Trace}
	env.bindBuiltin("trace", &Scheme{
		TypeVars: []string{"α"},
		Type: &TFunc2{
			Params: []Type{TString, &TVar2{Name: "α", Kind: Star}},
			EffectRow: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"Trace": TUnit},
				Tail:   nil,
			},
			Return: &TVar2{Name: "α", Kind: Star},
		},
	})

	return env
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

// bindBuiltin adds a builtin to the environment (internal use)
func (env *TypeEnv) bindBuiltin(name string, scheme *Scheme) {
	env.bindings[name] = scheme
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