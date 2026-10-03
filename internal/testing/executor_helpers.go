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
	// ModuleBindings holds each loaded module's OWN bindings (module path →
	// bare name → value), as injected by injectModuleBindings. It is the
	// module-scoped fallback for qualified references; see module_scope.go.
	ModuleBindings map[string]map[string]eval.Value
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
		// Fall back to the OWNING module's own bindings (modules whose path was
		// not captured under a qualified key). Never the shared env's bare
		// names: those belong to the module under test, and a same-named
		// binding there is a different function (#1461, #1516).
		if own, ok := r.ModuleBindings[ref.Module]; ok {
			if val, ok := own[ref.Name]; ok {
				return val, nil
			}
		}
		if _, ok := r.Modules[ref.Module]; ok {
			return nil, fmt.Errorf("function %s.%s is not bound in the test harness environment", ref.Module, ref.Name)
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
		return e.String() // bool, int, float: ast.Literal.String() spells each re-parseably (#1448)
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
		// TRIPWIRE (#590): `assert` has no prefix parselet in the general
		// expression grammar, so printing it verbatim produced source that could
		// never parse — surfacing as PAR_NO_PREFIX_PARSE from the pipeline, far
		// from the actual cause. FoldTestBody lowers every top-level AssertStmt
		// before printing, so reaching here means the lowering was bypassed.
		// Say so, in the same self-describing form this file uses for other
		// unrepresentable nodes.
		return fmt.Sprintf("<printer-error: AssertStmt reached the printer; FoldTestBody must lower it first (condition: %s)>",
			PrintAILANGSource(e.Condition))
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
