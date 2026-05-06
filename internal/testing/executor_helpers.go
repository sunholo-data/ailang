package testing

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/runtime"
)

// CombinedResolver resolves both builtin functions and user-defined functions from the environment.
// Used for inline test harness evaluation to support functions that depend on imports.
// It handles:
// - $adt synthetic module (ADT constructors from imported modules, e.g. Some/None from std/option)
// - Builtin references (module="$builtin" or name starts with "_")
// - Module-qualified references (module="std/list" name="filter")
// - Local references (module="" or module matches current file)
type CombinedResolver struct {
	Builtins *runtime.BuiltinRegistry
	Env      *eval.Environment               // Environment containing user-defined and imported functions
	Modules  map[string]*loader.LoadedModule // Loaded modules for module-qualified lookup
}

// ResolveValue implements eval.GlobalResolver for combined resolution.
func (r *CombinedResolver) ResolveValue(ref core.GlobalRef) (eval.Value, error) {
	// Case 0: $adt synthetic module — ADT constructors from imported modules.
	// The elaborator generates VarGlobal{Module:"$adt", Name:"make_Option_Some"} for
	// constructor calls on imported ADTs. Search r.Modules[*].Iface.Constructors.
	if ref.Module == "$adt" {
		return r.resolveAdtFactory(ref.Name)
	}

	// Case 1: Builtin references (module="$builtin" or name starts with "_")
	if ref.Module == "$builtin" || strings.HasPrefix(ref.Name, "_") {
		if val, ok := r.Builtins.Get(ref.Name); ok {
			return val, nil
		}
		// Not found in builtins - might be in environment
		if val, ok := r.Env.Get(ref.Name); ok {
			return val, nil
		}
		return nil, fmt.Errorf("builtin %s not found", ref.Name)
	}

	// Case 2: Module-qualified reference (e.g., std/list.filter)
	if ref.Module != "" {
		// Prefer module-qualified key to avoid alias collision when two modules
		// export the same function name (e.g. std/string.length vs std/list.length).
		qualifiedKey := ref.Module + "." + ref.Name
		if val, ok := r.Env.Get(qualifiedKey); ok {
			return val, nil
		}
		// Fall back to bare name lookup for modules whose path wasn't captured
		if mod, ok := r.Modules[ref.Module]; ok && mod != nil {
			for _, decl := range mod.Core.Decls {
				if let, ok := decl.(*core.Let); ok && let.Name == ref.Name {
					if val, ok := r.Env.Get(ref.Name); ok {
						return val, nil
					}
					return nil, fmt.Errorf("function %s.%s not yet evaluated in environment", ref.Module, ref.Name)
				}
				if letRec, ok := decl.(*core.LetRec); ok {
					for _, binding := range letRec.Bindings {
						if binding.Name == ref.Name {
							if val, ok := r.Env.Get(ref.Name); ok {
								return val, nil
							}
							return nil, fmt.Errorf("function %s.%s not yet evaluated in environment", ref.Module, ref.Name)
						}
					}
				}
			}
		}
		return nil, fmt.Errorf("module %s not found or function %s not in module", ref.Module, ref.Name)
	}

	// Case 3: Unqualified reference - look in environment
	// This includes both the test function being tested and any imported functions
	// that were elaborated and bound during pipeline execution.
	if val, ok := r.Env.Get(ref.Name); ok {
		return val, nil
	}

	// Case 4: Not found - return error (will be caught during harness evaluation)
	return nil, fmt.Errorf("undefined reference: %s (module: %s)", ref.Name, ref.Module)
}

// resolveAdtFactory resolves $adt synthetic module references for imported ADT constructors.
// Parses "make_Option_Some" → typeName="Option", ctorName="Some", then searches
// r.Modules for a matching Iface.Constructors entry to determine arity.
func (r *CombinedResolver) resolveAdtFactory(factoryName string) (eval.Value, error) {
	if !strings.HasPrefix(factoryName, "make_") {
		return nil, fmt.Errorf("invalid $adt factory name: %s", factoryName)
	}
	parts := strings.SplitN(factoryName[5:], "_", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid $adt factory format: %s (expected make_TypeName_CtorName)", factoryName)
	}
	typeName, ctorName := parts[0], parts[1]

	for _, mod := range r.Modules {
		if mod == nil || mod.Iface == nil || mod.Iface.Constructors == nil {
			continue
		}
		ctor, ok := mod.Iface.Constructors[ctorName]
		if !ok || ctor.TypeName != typeName {
			continue
		}
		if ctor.Arity == 0 {
			return &eval.TaggedValue{
				TypeName: typeName,
				CtorName: ctorName,
				Fields:   []eval.Value{},
			}, nil
		}
		return &eval.ConstructorClosure{
			TypeName: typeName,
			CtorName: ctorName,
			Arity:    ctor.Arity,
		}, nil
	}
	return nil, fmt.Errorf("constructor %s.%s not found in any loaded module", typeName, ctorName)
}

// equalValues performs deep equality check on eval values.
func equalValues(a, b eval.Value) bool {
	switch av := a.(type) {
	case *eval.IntValue:
		if bv, ok := b.(*eval.IntValue); ok {
			return av.Value == bv.Value
		}
	case *eval.FloatValue:
		if bv, ok := b.(*eval.FloatValue); ok {
			// Float comparison with tolerance for testing
			diff := av.Value - bv.Value
			if diff < 0 {
				diff = -diff
			}
			return diff < 1e-9
		}
	case *eval.BoolValue:
		if bv, ok := b.(*eval.BoolValue); ok {
			return av.Value == bv.Value
		}
	case *eval.StringValue:
		if bv, ok := b.(*eval.StringValue); ok {
			return av.Value == bv.Value
		}
	case *eval.ListValue:
		if bv, ok := b.(*eval.ListValue); ok {
			if len(av.Elements) != len(bv.Elements) {
				return false
			}
			for i := range av.Elements {
				if !equalValues(av.Elements[i], bv.Elements[i]) {
					return false
				}
			}
			return true
		}
	case *eval.TupleValue:
		if bv, ok := b.(*eval.TupleValue); ok {
			if len(av.Elements) != len(bv.Elements) {
				return false
			}
			for i := range av.Elements {
				if !equalValues(av.Elements[i], bv.Elements[i]) {
					return false
				}
			}
			return true
		}
	case *eval.RecordValue:
		if bv, ok := b.(*eval.RecordValue); ok {
			if len(av.Fields) != len(bv.Fields) {
				return false
			}
			for k, av := range av.Fields {
				bv, exists := bv.Fields[k]
				if !exists {
					return false
				}
				if !equalValues(av, bv) {
					return false
				}
			}
			return true
		}
	case *eval.TaggedValue:
		if bv, ok := b.(*eval.TaggedValue); ok {
			// Compare constructor names and fields
			if av.CtorName != bv.CtorName {
				return false
			}
			if len(av.Fields) != len(bv.Fields) {
				return false
			}
			for i := range av.Fields {
				if !equalValues(av.Fields[i], bv.Fields[i]) {
					return false
				}
			}
			return true
		}
	case *eval.UnitValue:
		_, ok := b.(*eval.UnitValue)
		return ok
	}
	return false
}

// Helper: split source into lines
func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		current += string(ch)
		if ch == '\n' {
			result = append(result, current)
			current = ""
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// Helper: join lines back
func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line
	}
	return result
}

// Helper: check if line contains pattern
func containsPattern(line, pattern string) bool {
	return len(line) >= len(pattern) && findSubstring(line, pattern)
}

// Helper: find substring
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// extractLambdaParams extracts parameter names from a Core Lambda
func extractLambdaParams(lambda *core.Lambda) []string {
	return lambda.Params
}

// injectADTConstructors injects ADT constructor bindings into the evaluator
func (e *Executor) injectADTConstructors(evaluator *eval.CoreEvaluator) {
	if e.sourceFile == nil {
		return
	}

	env := evaluator.Env()

	// Type declarations are in Decls ([]Node)
	for _, decl := range e.sourceFile.Decls {
		typeDecl, ok := decl.(*ast.TypeDecl)
		if !ok {
			continue
		}

		// Only process ADTs (algebraic types)
		if adt, ok := typeDecl.Definition.(*ast.AlgebraicType); ok {
			typeName := typeDecl.Name
			for _, ctor := range adt.Constructors {
				ctorName := ctor.Name
				arity := len(ctor.Fields)

				if arity == 0 {
					// Nullary constructor - bind directly to TaggedValue
					env.Set(ctorName, &eval.TaggedValue{
						TypeName: typeName,
						CtorName: ctorName,
						Fields:   []eval.Value{},
					})
				} else {
					// Constructor with data - bind to ConstructorClosure
					env.Set(ctorName, &eval.ConstructorClosure{
						TypeName: typeName,
						CtorName: ctorName,
						Arity:    arity,
					})
				}
			}
		}
	}
}

// injectModuleBindings evaluates all module Core programs and injects their bindings
// into the evaluator's environment. This allows the test harness to reference functions
// that were imported and elaborated (like functions from std/fs, std/net, etc.).
//
// CRITICAL BUG FIX (M-DX25):
// The issue was that FunctionValues were capturing `env` at injection time, before all
// module bindings were added to `env`. When a function's body references another imported
// function, that reference might not be in the captured environment snapshot.
//
// Solution: Use a two-pass approach:
//
//	Pass 1: Collect all lambda bindings to inject, but don't create FunctionValues yet
//	Pass 2: After env is populated with all names, create FunctionValues that capture
//	        the now-complete environment
func (e *Executor) injectModuleBindings(evaluator *eval.CoreEvaluator, env *eval.Environment) {
	if len(e.modules) == 0 {
		return
	}

	// PASS 1: Collect pending bindings (lambdas) and inject non-lambda values
	type PendingLambdaBinding struct {
		name       string
		lambda     *core.Lambda
		modulePath string // used for module-qualified env key to prevent alias collision
	}
	var pendingLambdas []PendingLambdaBinding

	for modulePath, mod := range e.modules {
		if mod.Core == nil {
			continue
		}

		// Process each declaration in the module
		for _, decl := range mod.Core.Decls {
			switch d := decl.(type) {
			case *core.Let:
				// For pure functions, the value is a Lambda - queue it for Pass 2
				if lambda, ok := d.Value.(*core.Lambda); ok {
					pendingLambdas = append(pendingLambdas, PendingLambdaBinding{
						name:       d.Name,
						lambda:     lambda,
						modulePath: modulePath,
					})
				} else if _, ok := d.Value.(*core.VarGlobal); ok {
					// This is a re-export of another module's function
					// We need to evaluate it to get the actual function value
					val, err := evaluator.Eval(d.Value)
					if err == nil && val != nil {
						env.Set(d.Name, val)
					}
				}

			case *core.LetRec:
				// For recursive bindings, queue the lambdas for Pass 2
				for _, binding := range d.Bindings {
					if lambda, ok := binding.Value.(*core.Lambda); ok {
						pendingLambdas = append(pendingLambdas, PendingLambdaBinding{
							name:       binding.Name,
							lambda:     lambda,
							modulePath: modulePath,
						})
					}
				}
			}
		}
	}

	// PASS 2: Now create FunctionValues with the fully-populated environment
	// This ensures all function dependencies can be resolved from env.
	// Also bind under a module-qualified key ("std/string.length") so that
	// CombinedResolver can distinguish same-named functions from different modules
	// (e.g. std/string.length vs std/list.length imported with an alias).
	for _, pending := range pendingLambdas {
		funcVal := &eval.FunctionValue{
			Params: extractLambdaParams(pending.lambda),
			Body:   pending.lambda.Body,
			Env:    env, // Capture the fully-populated environment
			Typed:  true,
		}
		env.Set(pending.name, funcVal)
		if pending.modulePath != "" {
			env.Set(pending.modulePath+"."+pending.name, funcVal)
		}
	}
}
