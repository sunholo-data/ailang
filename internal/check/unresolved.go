package check

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

// InterFunctionRefs performs a static check on the compiled Core program
// to detect inter-function reference issues that would fail at consumer load time.
// The resolver evaluates module declarations sequentially as Let/LetRec bindings.
// A core.Var in a function body must be resolvable: either a parameter, a binding
// from an earlier Let/LetRec, or a lambda-local binding. This catches the case where
// check --package passes but consumers get "undefined variable" errors.
func InterFunctionRefs(result pipeline.Result) []string {
	var warnings []string
	for modID, mod := range result.Modules {
		if mod.Core == nil {
			continue
		}
		if w := ModuleDeclRefs(modID, mod.Core); len(w) > 0 {
			warnings = append(warnings, w...)
		}
	}
	return warnings
}

// ModuleDeclRefs walks module-level declarations and verifies that every
// core.Var referenced in a function body is in scope — either a parameter,
// a let-bound name from an enclosing Let/LetRec, or a name from an earlier
// top-level declaration.
func ModuleDeclRefs(modID string, prog *core.Program) []string {
	var warnings []string
	// Track names bound by earlier top-level declarations
	bound := make(map[string]bool)

	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *core.Let:
			// Check the Let value for unresolved Vars
			if unresolved := FindUnresolvedVars(d.Value, bound); len(unresolved) > 0 {
				for _, name := range unresolved {
					warnings = append(warnings, fmt.Sprintf(
						"function %q references %q which is not yet defined (would fail at consumer load time)",
						d.Name, name))
				}
			}
			bound[d.Name] = true

		case *core.LetRec:
			// LetRec bindings are mutually visible
			recNames := make(map[string]bool)
			for _, b := range d.Bindings {
				recNames[b.Name] = true
			}
			// Merge with outer bound for checking
			merged := make(map[string]bool)
			for k := range bound {
				merged[k] = true
			}
			for k := range recNames {
				merged[k] = true
			}
			for _, b := range d.Bindings {
				if unresolved := FindUnresolvedVars(b.Value, merged); len(unresolved) > 0 {
					for _, name := range unresolved {
						warnings = append(warnings, fmt.Sprintf(
							"function %q references %q which is not yet defined (would fail at consumer load time)",
							b.Name, name))
					}
				}
			}
			// All LetRec names become bound for subsequent declarations
			for k := range recNames {
				bound[k] = true
			}
		}
	}
	return warnings
}

// FindUnresolvedVars walks a Core expression and returns any core.Var names
// that are not in the provided scope (bound names) and not lambda parameters.
func FindUnresolvedVars(expr core.CoreExpr, outerScope map[string]bool) []string {
	var unresolved []string
	seen := make(map[string]bool) // deduplicate
	walkForVars(expr, outerScope, nil, seen, &unresolved)
	return unresolved
}

// walkForVars recursively walks a Core expression collecting unresolved Var references.
// scope = names from outer Let/LetRec declarations, locals = names from lambda params
// and inner let bindings.
func walkForVars(expr core.CoreExpr, scope map[string]bool, locals map[string]bool, seen map[string]bool, out *[]string) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.Var:
		name := e.Name
		if !scope[name] && !locals[name] && !seen[name] {
			// Skip common builtins/operators that the evaluator handles
			if !isKnownBuiltin(name) {
				seen[name] = true
				*out = append(*out, name)
			}
		}

	case *core.VarGlobal:
		// Global refs are resolved by the resolver, not local scope — skip

	case *core.Lambda:
		// Lambda params create new local scope
		newLocals := make(map[string]bool)
		for k, v := range locals {
			newLocals[k] = v
		}
		for _, p := range e.Params {
			newLocals[p] = true
		}
		walkForVars(e.Body, scope, newLocals, seen, out)

	case *core.Let:
		// Check the value in current scope
		walkForVars(e.Value, scope, locals, seen, out)
		// Body has the new name in scope
		newLocals := make(map[string]bool)
		for k, v := range locals {
			newLocals[k] = v
		}
		newLocals[e.Name] = true
		walkForVars(e.Body, scope, newLocals, seen, out)

	case *core.LetRec:
		// All bindings visible to each other
		newLocals := make(map[string]bool)
		for k, v := range locals {
			newLocals[k] = v
		}
		for _, b := range e.Bindings {
			newLocals[b.Name] = true
		}
		for _, b := range e.Bindings {
			walkForVars(b.Value, scope, newLocals, seen, out)
		}
		walkForVars(e.Body, scope, newLocals, seen, out)

	case *core.App:
		walkForVars(e.Func, scope, locals, seen, out)
		for _, arg := range e.Args {
			walkForVars(arg, scope, locals, seen, out)
		}

	case *core.If:
		walkForVars(e.Cond, scope, locals, seen, out)
		walkForVars(e.Then, scope, locals, seen, out)
		walkForVars(e.Else, scope, locals, seen, out)

	case *core.BinOp:
		walkForVars(e.Left, scope, locals, seen, out)
		walkForVars(e.Right, scope, locals, seen, out)

	case *core.UnOp:
		walkForVars(e.Operand, scope, locals, seen, out)

	case *core.Match:
		walkForVars(e.Scrutinee, scope, locals, seen, out)
		for _, arm := range e.Arms {
			caseLocals := make(map[string]bool)
			for k, v := range locals {
				caseLocals[k] = v
			}
			collectPatternVars(arm.Pattern, caseLocals)
			if arm.Guard != nil {
				walkForVars(arm.Guard, scope, caseLocals, seen, out)
			}
			walkForVars(arm.Body, scope, caseLocals, seen, out)
		}

	case *core.Record:
		for _, v := range e.Fields {
			walkForVars(v, scope, locals, seen, out)
		}

	case *core.RecordAccess:
		walkForVars(e.Record, scope, locals, seen, out)

	case *core.List:
		for _, elem := range e.Elements {
			walkForVars(elem, scope, locals, seen, out)
		}

	case *core.Tuple:
		for _, elem := range e.Elements {
			walkForVars(elem, scope, locals, seen, out)
		}

	case *core.Lit:
		// No vars in literals
	}
}

// collectPatternVars extracts variable bindings from a pattern.
func collectPatternVars(pat core.CorePattern, locals map[string]bool) {
	if pat == nil {
		return
	}
	switch p := pat.(type) {
	case *core.VarPattern:
		locals[p.Name] = true
	case *core.ConstructorPattern:
		for _, sub := range p.Args {
			collectPatternVars(sub, locals)
		}
	case *core.TuplePattern:
		for _, sub := range p.Elements {
			collectPatternVars(sub, locals)
		}
	case *core.RecordPattern:
		for _, sub := range p.Fields {
			collectPatternVars(sub, locals)
		}
	case *core.ListPattern:
		for _, sub := range p.Elements {
			collectPatternVars(sub, locals)
		}
		if p.Tail != nil {
			collectPatternVars(*p.Tail, locals)
		}
	case *core.LitPattern, *core.WildcardPattern:
		// No variable bindings
	}
}

// isKnownBuiltin returns true for names that are resolved by the runtime
// rather than module-level Let bindings (builtins, dictionary params, etc.)
func isKnownBuiltin(name string) bool {
	// Dictionary parameters start with $dict_
	if strings.HasPrefix(name, "$dict_") || strings.HasPrefix(name, "$") {
		return true
	}
	return false
}
