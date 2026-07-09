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

// FoldBodyExprs collapses a slice of AST expressions — as produced by parsing a
// named test block where semicolons separate statements — into a single nested
// expression.  The parser emits standalone `let x = val` nodes (Body == nil)
// for each binding; this function re-chains them:
//
//	[let x = val, let y = ..., finalExpr]
//	  → let x = val in (let y = ... in finalExpr)
//
// Non-let expressions that are not the last item are dropped (they were
// evaluated for side-effects in the original source, which is meaningless for
// pure bodies, so stripping them is safe).
func FoldBodyExprs(exprs []ast.Expr) ast.Expr {
	if len(exprs) == 0 {
		return nil
	}
	// Walk in reverse, threading each let-binding around the accumulated tail.
	result := exprs[len(exprs)-1]
	for i := len(exprs) - 2; i >= 0; i-- {
		switch e := exprs[i].(type) {
		case *ast.Let:
			// Clone to avoid mutating the parser's AST in place.
			result = &ast.Let{
				Name:  e.Name,
				Type:  e.Type,
				Value: e.Value,
				Body:  result,
				Pos:   e.Pos,
			}
		case *ast.LetRec:
			result = &ast.LetRec{
				Name:  e.Name,
				Type:  e.Type,
				Value: e.Value,
				Body:  result,
				Pos:   e.Pos,
			}
		default:
			// Side-effect expression before the final value — wrap in a let_ binding.
			// AILANG has no sequencing operator, so we emit "let _ = expr in rest".
			// This is the closest approximation; pure bodies shouldn't have real side
			// effects anyway, so semantics are preserved.
			result = &ast.Let{
				Name:  "_seq",
				Value: exprs[i],
				Body:  result,
				Pos:   exprs[i].Position(),
			}
		}
	}
	return result
}

// PrintAILANGSource converts an AST expression to valid AILANG source text.
//
// This is needed because ast.FuncCall.String() uses prefix notation
// "(f arg1 arg2)" which is not valid AILANG syntax.  Named test bodies
// are re-elaborated through the pipeline from source text, so we need
// proper AILANG syntax rather than the debug representation from String().
//
// All known ast.Expr variants are handled.  An unknown variant returns a
// descriptive error string that will cause the pipeline to fail with a
// parse error — surfaced as a test FAIL, never a crash.
//
// IMPORTANT: always call with a non-nil expr; the nil guard at the top
// returns a safe fallback so recursive callers (e.g. Let.Body == nil) never
// dereference a nil interface.
func PrintAILANGSource(expr ast.Expr) string {
	if expr == nil {
		return "<nil-expr>"
	}
	switch e := expr.(type) {
	case *ast.Literal:
		// UnitLit: Literal.String() returns "<nil>" via fmt.Sprintf("%v", nil).
		// We need "()" which is the AILANG unit literal syntax.
		if e.Kind == ast.UnitLit {
			return "()"
		}
		// StringLit: Literal.String() only escapes \ and " but not \n, \t, \r.
		// Re-escape control characters so the round-tripped source parses correctly.
		if e.Kind == ast.StringLit {
			if s, ok := e.Value.(string); ok {
				var buf strings.Builder
				buf.WriteByte('"')
				for _, ch := range s {
					switch ch {
					case '\\':
						buf.WriteString(`\\`)
					case '"':
						buf.WriteString(`\"`)
					case '\n':
						buf.WriteString(`\n`)
					case '\t':
						buf.WriteString(`\t`)
					case '\r':
						buf.WriteString(`\r`)
					default:
						buf.WriteRune(ch)
					}
				}
				buf.WriteByte('"')
				return buf.String()
			}
		}
		return e.String() // bool, int, float all have correct String()
	case *ast.Identifier:
		return e.Name
	case *ast.BinaryOp:
		return fmt.Sprintf("(%s %s %s)",
			PrintAILANGSource(e.Left), e.Op, PrintAILANGSource(e.Right))
	case *ast.UnaryOp:
		return fmt.Sprintf("(%s %s)", e.Op, PrintAILANGSource(e.Expr))
	case *ast.FuncCall:
		// Produce "f(arg1, arg2)" — the standard AILANG call syntax.
		// Special case: f(()) — a call with a single unit literal — should print
		// as f() because that is how zero-arg functions are called in AILANG source.
		funcStr := PrintAILANGSource(e.Func)
		if len(e.Args) == 0 {
			return funcStr + "()"
		}
		if len(e.Args) == 1 {
			if lit, ok := e.Args[0].(*ast.Literal); ok && lit.Kind == ast.UnitLit {
				return funcStr + "()"
			}
		}
		args := make([]string, len(e.Args))
		for i, a := range e.Args {
			args[i] = PrintAILANGSource(a)
		}
		return fmt.Sprintf("%s(%s)", funcStr, strings.Join(args, ", "))
	case *ast.Tuple:
		elems := make([]string, len(e.Elements))
		for i, el := range e.Elements {
			elems[i] = PrintAILANGSource(el)
		}
		return "(" + strings.Join(elems, ", ") + ")"
	case *ast.List:
		elems := make([]string, len(e.Elements))
		for i, el := range e.Elements {
			elems[i] = PrintAILANGSource(el)
		}
		return "[" + strings.Join(elems, ", ") + "]"
	case *ast.Array:
		elems := make([]string, len(e.Elements))
		for i, el := range e.Elements {
			elems[i] = PrintAILANGSource(el)
		}
		return "#[" + strings.Join(elems, ", ") + "]"
	case *ast.If:
		return fmt.Sprintf("if %s then %s else %s",
			PrintAILANGSource(e.Condition),
			PrintAILANGSource(e.Then),
			PrintAILANGSource(e.Else))
	case *ast.Let:
		if e.Body == nil {
			// Standalone let-binding (no 'in'): emit as "let name = val" — callers
			// that need a complete expression should use FoldBodyExprs first.
			return fmt.Sprintf("let %s = %s", e.Name, PrintAILANGSource(e.Value))
		}
		return fmt.Sprintf("let %s = %s in\n%s",
			e.Name, PrintAILANGSource(e.Value), PrintAILANGSource(e.Body))
	case *ast.LetRec:
		if e.Body == nil {
			return fmt.Sprintf("letrec %s = %s", e.Name, PrintAILANGSource(e.Value))
		}
		return fmt.Sprintf("letrec %s = %s in\n%s",
			e.Name, PrintAILANGSource(e.Value), PrintAILANGSource(e.Body))
	case *ast.Match:
		cases := make([]string, len(e.Cases))
		for i, c := range e.Cases {
			if c.Guard != nil {
				cases[i] = fmt.Sprintf("%s if %s => %s",
					c.Pattern, PrintAILANGSource(c.Guard), PrintAILANGSource(c.Body))
			} else {
				cases[i] = fmt.Sprintf("%s => %s", c.Pattern, PrintAILANGSource(c.Body))
			}
		}
		return fmt.Sprintf("match %s { %s }", PrintAILANGSource(e.Expr), strings.Join(cases, ", "))
	case *ast.Block:
		parts := make([]string, len(e.Exprs))
		for i, ex := range e.Exprs {
			parts[i] = PrintAILANGSource(ex)
		}
		return "{ " + strings.Join(parts, "; ") + " }"
	case *ast.Record:
		fields := make([]string, len(e.Fields))
		for i, f := range e.Fields {
			fields[i] = fmt.Sprintf("%s: %s", f.Name, PrintAILANGSource(f.Value))
		}
		return "{ " + strings.Join(fields, ", ") + " }"
	case *ast.RecordAccess:
		return fmt.Sprintf("%s.%s", PrintAILANGSource(e.Record), e.Field)
	case *ast.RecordUpdate:
		fields := make([]string, len(e.Fields))
		for i, f := range e.Fields {
			fields[i] = fmt.Sprintf("%s: %s", f.Name, PrintAILANGSource(f.Value))
		}
		return fmt.Sprintf("{ %s | %s }", PrintAILANGSource(e.Base), strings.Join(fields, ", "))
	case *ast.Lambda:
		params := make([]string, len(e.Params))
		for i, p := range e.Params {
			if p.Type != nil {
				params[i] = fmt.Sprintf("%s: %s", p.Name, p.Type)
			} else {
				params[i] = p.Name
			}
		}
		return fmt.Sprintf("\\%s. %s", strings.Join(params, " "), PrintAILANGSource(e.Body))
	case *ast.FuncLit:
		params := make([]string, len(e.Params))
		for i, p := range e.Params {
			if p.Type != nil {
				params[i] = fmt.Sprintf("%s: %s", p.Name, p.Type)
			} else {
				params[i] = p.Name
			}
		}
		retStr := ""
		if e.ReturnType != nil {
			retStr = fmt.Sprintf(" -> %s", e.ReturnType)
		}
		return fmt.Sprintf("func(%s)%s { %s }", strings.Join(params, ", "), retStr, PrintAILANGSource(e.Body))
	case *ast.Error:
		return fmt.Sprintf("<printer-error: unrepresentable *ast.Error node: %s>", e.Msg)
	case *ast.AssertStmt:
		return fmt.Sprintf("assert %s", PrintAILANGSource(e.Condition))
	case *ast.Send:
		return fmt.Sprintf("%s <- %s", PrintAILANGSource(e.Channel), PrintAILANGSource(e.Value))
	case *ast.Recv:
		return fmt.Sprintf("<- %s", PrintAILANGSource(e.Channel))
	case *ast.ForallExpr:
		return fmt.Sprintf("forall %s: %s..%s => %s",
			e.Var, PrintAILANGSource(e.Lo), PrintAILANGSource(e.Hi), PrintAILANGSource(e.Body))
	case *ast.QuasiQuote:
		return e.String()
	default:
		// Unknown variant: return a descriptive error string.
		// The pipeline will fail with a parse error (test FAIL), not a process crash.
		return fmt.Sprintf("<printer-error: unhandled ast.Expr type %T>", expr)
	}
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

	// PASS 1: Inject Let (non-recursive) lambdas and wire LetRec groups with proper
	// self-referential environments.
	//
	// PROBLEM: Naively injecting every lambda with Env=env causes two bugs:
	//
	// (a) Name collision: both std/list and std/string export "concat" (different arity).
	//     The last writer wins for env["concat"], so whichever module is processed last
	//     (random map iteration order) becomes "concat", corrupting cross-module calls.
	//
	// (b) Broken self-recursion: std/list.concat body contains Var{concat} (recursive
	//     call to itself).  If env["concat"] was overwritten by std/string.concat (1-param),
	//     the recursive call uses the wrong function → "expects 1 arguments, got 2".
	//
	// FIX:
	//   - For LetRec groups (self/mutual recursion), set up a recEnv child per group
	//     using IndirectValue cells, exactly like evalCoreLetRec does.  The closures
	//     capture recEnv so Var{name} within the group resolves back to the cell.
	//   - Only bind the QUALIFIED key in the outer env (e.g. "pkg/std/list.concat") —
	//     NOT the bare name.  This prevents collisions between modules.
	//   - After ALL modules are processed, resolve VarGlobal re-exports into bare names
	//     in a final pass so engine.ail's `let concat = VarGlobal{pkg/std/list,concat}`
	//     correctly sets env["concat"] = 2-param FunctionValue.

	type DeferredVarGlobal struct {
		name string
		ref  *core.VarGlobal
	}
	var deferredVarGlobals []DeferredVarGlobal

	// Use a deterministic module order: sort module paths so results are reproducible
	// regardless of Go's random map iteration order.
	sortedPaths := make([]string, 0, len(e.modules))
	for modPath := range e.modules {
		sortedPaths = append(sortedPaths, modPath)
	}
	// Sort: shorter paths (std/*) before longer (pkg/sunholo/...) for stable dependency order
	sortStrings(sortedPaths)

	for _, modulePath := range sortedPaths {
		mod := e.modules[modulePath]
		if mod == nil || mod.Core == nil {
			continue
		}

		for _, decl := range mod.Core.Decls {
			switch d := decl.(type) {
			case *core.Let:
				if lambda, ok := d.Value.(*core.Lambda); ok {
					// Non-recursive function: create closure with the outer env
					// and bind ONLY under the qualified key.  The bare name will
					// be resolved via VarGlobal re-exports or the qualified lookup.
					funcVal := &eval.FunctionValue{
						Params: lambda.Params,
						Body:   lambda.Body,
						Env:    env,
						Typed:  true,
					}
					// Qualified key (never collides with same-named functions from
					// other modules because module paths are unique).
					if modulePath != "" {
						env.Set(modulePath+"."+d.Name, funcVal)
					}
					// Bare name: set unconditionally.  For Let (non-recursive)
					// lambdas there is no self-reference issue; the only concern is
					// ordering, which is now deterministic (sorted paths above).
					env.Set(d.Name, funcVal)
				} else if vg, ok := d.Value.(*core.VarGlobal); ok {
					// Re-export of another module's function: defer until all
					// lambdas are wired so the qualified key is already in env.
					deferredVarGlobals = append(deferredVarGlobals, DeferredVarGlobal{
						name: d.Name,
						ref:  vg,
					})
				}

			case *core.LetRec:
				// Mutual/self-recursive group: use IndirectValue cells so Var
				// references within the group resolve to the correct version of
				// each function — not whatever happened to be last in env.
				//
				// Algorithm mirrors evalCoreLetRec phases 1-2.5:
				//   Phase A: allocate cells in recEnv
				//   Phase B: build FunctionValues that capture recEnv
				//   Phase C: fill cells; register qualified keys in outer env;
				//             also set bare names for non-conflicting functions
				recEnv := env.NewChildEnvironment()
				cells := make(map[string]*eval.RefCell, len(d.Bindings))

				// Phase A
				for _, binding := range d.Bindings {
					cell := &eval.RefCell{}
					cells[binding.Name] = cell
					recEnv.Set(binding.Name, &eval.IndirectValue{Cell: cell})
				}

				// Phase B+C
				for _, binding := range d.Bindings {
					lambda, ok := binding.Value.(*core.Lambda)
					if !ok {
						continue
					}
					funcVal := &eval.FunctionValue{
						Params: lambda.Params,
						Body:   lambda.Body,
						Env:    recEnv, // captures the group's own env for self-ref
						Typed:  true,
					}
					// Fill the indirect cell so in-group Var{name} resolves correctly
					cells[binding.Name].Val = funcVal
					cells[binding.Name].Init = true

					// Register under qualified key in outer env (for CombinedResolver)
					if modulePath != "" {
						env.Set(modulePath+"."+binding.Name, funcVal)
					}
					// Also expose the actual FunctionValue (not IndirectValue) under
					// bare name in the outer env, so EvalCoreProgram and cross-module
					// callers that look up bare names find the correct version.
					// This is safe: recEnv's self-reference via IndirectValue is still
					// intact regardless of what env["name"] holds.
					env.Set(binding.Name, funcVal)
				}
			}
		}
	}

	// PASS 2: Resolve deferred VarGlobal re-exports now that all lambdas (including
	// qualified keys) are in env.  For example, engine.ail's
	//   let concat = VarGlobal{pkg/std/list, concat}
	// resolves to the 2-param FunctionValue after env["pkg/std/list.concat"] is set.
	// This overwrites any bare-name "concat" set above, so engine functions that
	// reference Var{concat} get the correct (2-param) version.
	for _, dv := range deferredVarGlobals {
		val, err := evaluator.Eval(dv.ref)
		if err == nil && val != nil {
			env.Set(dv.name, val)
		}
	}
}

// sortStrings sorts a slice of strings in-place (ascending lexicographic order).
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		key := ss[i]
		j := i - 1
		for j >= 0 && ss[j] > key {
			ss[j+1] = ss[j]
			j--
		}
		ss[j+1] = key
	}
}
