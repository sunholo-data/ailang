package pipeline

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
)

// M-CHECK-STRICT-FALLBACKS: static "Ok contains a default/empty value" detector.
//
// A Result-returning function whose Ok branch carries an empty/zero value
// (`Ok("")`, `Ok([])`, `Ok({})`, an all-zero record, or a known-empty builder
// like `std/json.jo([])`) silently lies about success: the caller's `Ok(v) =>`
// arm fires with zero data instead of falling through to `Err(e) =>`. This is
// the firestore@0.7.1 incident (see the design doc).
//
// The pass runs over the *elaborated* Core program, mirroring
// warn_split_args.go's DetectArgOrderWarnings. Two facts drive the design:
//
//  1. `Ok(x)` elaborates to `App{Func: VarGlobal{$adt.make_Result_Ok}, Args:[x]}`
//     (see internal/elaborate/expr_calls.go). Detection keys on that resolved
//     constructor identity — sound, not a bare name match.
//
//  2. ANF normalization makes App args ATOMIC: `Ok([])` becomes
//     `let t = [] in make_Result_Ok(t)` and `Ok(jo([]))` becomes
//     `let t = jo([]) in make_Result_Ok(t)`. So Args[0] is a `core.Var`, not
//     the literal/call. The detector threads an `env` of enclosing `Let`
//     bindings and resolves Vars back to their bound value before classifying.
//
// A user-defined LOCAL `jo` elaborates to a plain `core.Var` (never a
// VarGlobal), so the module-qualified registry structurally excludes it.

// The resolved factory identity for `Ok(...)`. `Err(...)` is make_Result_Err,
// which we never flag (an empty Err is not a lie about success).
var resultOkFactory = core.GlobalRef{Module: "$adt", Name: "make_Result_Ok"}

// maxResolveDepth bounds let-chain / value-classification recursion so a
// pathological or cyclic Core graph cannot loop the pass.
const maxResolveDepth = 64

// resolveFn resolves a possibly-Var atomic value through the accumulated
// enclosing Let bindings. It is passed to registry detectors so they can peer
// through ANF `Var` args.
type resolveFn func(v core.CoreExpr) core.CoreExpr

// knownEmptyBuilder describes a resolved builder call that yields a
// semantically "empty" value when invoked with empty arguments
// (e.g. std/json.jo([]) => {}). Keyed by module-qualified identity so a
// user-local builder with the same bare name is never matched.
type knownEmptyBuilder struct {
	module string
	name   string
	// isEmptyCall reports whether THIS particular App{VarGlobal, args} is the
	// empty form. Kept sound: jo([realKV]) must NOT be flagged.
	isEmptyCall func(app *core.App, resolve resolveFn) bool
}

// knownEmptyBuilders is the extensible, curated registry of empty-builder
// identities — the mirror of warn_split_args.go's swapTraps. Keyed by
// GlobalRef{Module,Name}, NEVER by bare name.
var knownEmptyBuilders = map[core.GlobalRef]knownEmptyBuilder{
	{Module: "std/json", Name: "jo"}: {module: "std/json", name: "jo", isEmptyCall: joEmpty},
	// Future entries added here (extensible table, like swapTraps).
}

// joEmpty reports whether a std/json.jo(...) application is the empty-object
// form: exactly one argument that resolves to an empty list.
func joEmpty(app *core.App, resolve resolveFn) bool {
	if len(app.Args) != 1 {
		return false
	}
	return isEmptyList(resolve(app.Args[0]))
}

// isEmptyList reports whether a resolved value is an empty list literal.
func isEmptyList(v core.CoreExpr) bool {
	lst, ok := v.(*core.List)
	return ok && len(lst.Elements) == 0
}

// resolveAtomic resolves an atomic value through the enclosing Let bindings.
// If v is a `core.Var{Name}` bound in env, it returns the bound value
// (recursively, up to maxResolveDepth). Otherwise it returns v unchanged.
func resolveAtomic(v core.CoreExpr, env map[string]core.CoreExpr, depth int) core.CoreExpr {
	if depth <= 0 {
		return v
	}
	if vr, ok := v.(*core.Var); ok {
		if bound, found := env[vr.Name]; found {
			return resolveAtomic(bound, env, depth-1)
		}
	}
	return v
}

// isEmptyValue classifies a RESOLVED Core value as empty/zero. It returns
// (true, kindDescription) when the value is a strict-fallback default, else
// (false, ""). It resolves Vars through env as it descends and is depth-bounded.
//
// Empty/collection/record/known-builder forms flag unconditionally. Bare zero
// scalars (Ok(0), Ok("")) are handled by the caller's context guard: only the
// empty-STRING form flags here (an empty string is a canonical empty value);
// numeric/bool zeroes are intentionally NOT classified as empty by this
// function to avoid false-flagging a legitimate `Ok(0)` success value.
func isEmptyValue(v core.CoreExpr, env map[string]core.CoreExpr, depth int) (bool, string) {
	if depth <= 0 {
		return false, ""
	}
	resolve := func(x core.CoreExpr) core.CoreExpr {
		return resolveAtomic(x, env, maxResolveDepth)
	}
	rv := resolveAtomic(v, env, maxResolveDepth)

	switch n := rv.(type) {
	case *core.Lit:
		// Only the empty string is treated as an empty value. Numeric/bool
		// zeroes are legitimate success values and are NOT flagged here.
		if n.Kind == core.StringLit {
			if s, ok := n.Value.(string); ok && s == "" {
				return true, "empty string"
			}
		}
		return false, ""

	case *core.List:
		if len(n.Elements) == 0 {
			return true, "empty list"
		}
		return false, ""

	case *core.Record:
		// Empty record, OR a record where EVERY field resolves to a zero value
		// (empty string / zero scalar / empty collection / empty record).
		if len(n.Fields) == 0 {
			return true, "empty record"
		}
		for _, fv := range n.Fields {
			if !isZeroField(fv, env, depth-1) {
				return false, ""
			}
		}
		return true, "all-zero record"

	case *core.App:
		// Known-empty builder (registry, module-qualified identity).
		if vg, ok := n.Func.(*core.VarGlobal); ok {
			if b, found := knownEmptyBuilders[vg.Ref]; found {
				if b.isEmptyCall(n, resolve) {
					return true, fmt.Sprintf("empty %s.%s()", b.module, b.name)
				}
			}
			// Zero-argument data-constructor with all-zero args (Pattern C):
			// $adt.make_<Type>_<Ctor> whose every arg resolves to a zero value.
			if isADTFactory(vg.Ref) {
				if len(n.Args) == 0 {
					return false, "" // nullary ctor carries no default data
				}
				for _, a := range n.Args {
					if !isZeroField(a, env, depth-1) {
						return false, ""
					}
				}
				return true, "zero-valued constructor"
			}
		}
		return false, ""

	default:
		return false, ""
	}
}

// isZeroField reports whether a record field / constructor argument resolves to
// a zero-default value. Unlike isEmptyValue, this treats numeric/bool zeroes as
// zero (a record of {name:"", age:0, active:false} is the Pattern B all-zero
// case). A non-literal (Var to a real computation, function call) is NOT zero.
func isZeroField(v core.CoreExpr, env map[string]core.CoreExpr, depth int) bool {
	if depth <= 0 {
		return false
	}
	rv := resolveAtomic(v, env, maxResolveDepth)
	switch n := rv.(type) {
	case *core.Lit:
		switch n.Kind {
		case core.StringLit:
			s, ok := n.Value.(string)
			return ok && s == ""
		case core.IntLit:
			return isZeroNumber(n.Value)
		case core.FloatLit:
			return isZeroNumber(n.Value)
		case core.BoolLit:
			b, ok := n.Value.(bool)
			return ok && !b
		default:
			return false
		}
	case *core.List:
		return len(n.Elements) == 0
	case *core.Record:
		if len(n.Fields) == 0 {
			return true
		}
		for _, fv := range n.Fields {
			if !isZeroField(fv, env, depth-1) {
				return false
			}
		}
		return true
	default:
		// Non-literal expression (function call, unresolved Var, etc.): not zero.
		return false
	}
}

// isZeroNumber reports whether a numeric literal value equals zero, tolerating
// the int64/float64/int representations produced by elaboration.
func isZeroNumber(v interface{}) bool {
	switch x := v.(type) {
	case int64:
		return x == 0
	case int:
		return x == 0
	case float64:
		return x == 0
	default:
		return false
	}
}

// isADTFactory reports whether a GlobalRef is a $adt constructor factory
// (make_<Type>_<Ctor>), excluding the Result factories which the caller keys on
// separately.
func isADTFactory(ref core.GlobalRef) bool {
	return ref.Module == "$adt" && len(ref.Name) > len("make_") && ref.Name[:len("make_")] == "make_"
}

// StrictFallbackWarning is a non-blocking warning that a Result-returning
// function's Ok branch carries a default/empty value. It satisfies
// elaborate.Warning.
type StrictFallbackWarning struct {
	Location ast.Pos // Surface location of the offending Ok(...) call
	FuncName string  // enclosing function name (best-effort)
	Kind     string  // e.g. "empty list", "empty std/json.jo()"
}

// compile-time assertion that StrictFallbackWarning satisfies the interface.
var _ elaborate.Warning = (*StrictFallbackWarning)(nil)

// Position exposes 1-indexed line/col for LSP diagnostic placement.
func (w *StrictFallbackWarning) Position() (line, col int) {
	return w.Location.Line, w.Location.Column
}

// String renders the STRICT_FALLBACK_001 diagnostic.
func (w *StrictFallbackWarning) String() string {
	return fmt.Sprintf(
		"STRICT_FALLBACK_001 at %s\n"+
			"  Ok branch returns a %s in a Result-returning function.\n"+
			"  A Result's Ok with a default/empty value can't be distinguished from a\n"+
			"  populated success: the caller's `Ok(v) =>` arm fires with zero data\n"+
			"  instead of falling through to its `Err(e) =>` handler.\n"+
			"  Fix: return `Err(\"...\")` with a descriptive message, or if the empty-Ok\n"+
			"  is legitimate, annotate the function with\n"+
			"    @allow_empty_ok(\"rationale for why empty-Ok is correct here\")",
		w.Location.String(), w.Kind,
	)
}

// Code returns the stable diagnostic code (used by the package-mode gate).
func (w *StrictFallbackWarning) Code() string { return "STRICT_FALLBACK_001" }

// DetectStrictFallbacks walks an elaborated Core program and returns
// STRICT_FALLBACK_001 warnings for empty/default `Ok(...)` values in
// Result-returning functions. It reads the surface `file` for the return-type
// filter and the `@allow_empty_ok` suppression (annotations live only on the
// surface FuncDecl, not on Core). Deterministic output (sorted by location).
func DetectStrictFallbacks(file *ast.File, prog *core.Program) []elaborate.Warning {
	if prog == nil {
		return nil
	}

	// Build the candidate set: functions whose declared return type is
	// Result[_,_] AND that lack @allow_empty_ok. Names not in the map are
	// skipped (non-Result functions, or suppressed ones).
	candidates := resultReturningFuncs(file)

	var out []elaborate.Warning
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *core.Let:
			if _, ok := candidates[d.Name]; ok {
				walkStrictFallback(d.Name, d.Value, map[string]core.CoreExpr{}, &out)
			}
		case *core.LetRec:
			for _, b := range d.Bindings {
				if _, ok := candidates[b.Name]; ok {
					walkStrictFallback(b.Name, b.Value, map[string]core.CoreExpr{}, &out)
				}
			}
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		li, ci := warnPos(out[i])
		lj, cj := warnPos(out[j])
		if li != lj {
			return li < lj
		}
		return ci < cj
	})
	return out
}

// warnPos extracts a 1-indexed (line, col) from a StrictFallbackWarning for
// deterministic sorting; unknown warning types sort to the front.
func warnPos(w elaborate.Warning) (line, col int) {
	if sf, ok := w.(*StrictFallbackWarning); ok {
		return sf.Position()
	}
	return 0, 0
}

// resultReturningFuncs returns the set of function names in `file` whose
// declared return type is Result[_,_] and that do NOT carry @allow_empty_ok.
// If file is nil, the set is empty (no candidates → no warnings).
func resultReturningFuncs(file *ast.File) map[string]struct{} {
	out := make(map[string]struct{})
	if file == nil {
		return out
	}
	for _, fn := range file.Funcs {
		if fn == nil {
			continue
		}
		if !isResultReturnType(fn.ReturnType) {
			continue
		}
		if fn.GetAnnotation("allow_empty_ok") != nil {
			continue // suppressed
		}
		out[fn.Name] = struct{}{}
	}
	return out
}

// isResultReturnType reports whether a surface return type is `Result[_, _]`.
func isResultReturnType(t ast.Type) bool {
	app, ok := t.(*ast.TypeApp)
	return ok && app.Constructor == "Result"
}

// walkStrictFallback recursively visits a function body, threading `env` (the
// accumulated enclosing Let bindings) so ANF `Var` args can be resolved back to
// their bound values. On an `Ok(x)` App it resolves + classifies x.
func walkStrictFallback(fnName string, e core.CoreExpr, env map[string]core.CoreExpr, out *[]elaborate.Warning) {
	if e == nil {
		return
	}
	switch n := e.(type) {
	case *core.App:
		if vg, ok := n.Func.(*core.VarGlobal); ok && vg.Ref == resultOkFactory && len(n.Args) == 1 {
			if empty, kind := isEmptyValue(n.Args[0], env, maxResolveDepth); empty {
				*out = append(*out, &StrictFallbackWarning{
					Location: n.OriginalSpan(),
					FuncName: fnName,
					Kind:     kind,
				})
			}
		}
		walkStrictFallback(fnName, n.Func, env, out)
		for _, a := range n.Args {
			walkStrictFallback(fnName, a, env, out)
		}
	case *core.Let:
		// Thread the binding into env for the body walk. The value is visited
		// under the CURRENT env (the binding is not yet in scope for its own value).
		walkStrictFallback(fnName, n.Value, env, out)
		child := cloneEnv(env)
		child[n.Name] = n.Value
		walkStrictFallback(fnName, n.Body, child, out)
	case *core.LetRec:
		child := cloneEnv(env)
		for _, b := range n.Bindings {
			child[b.Name] = b.Value
		}
		for _, b := range n.Bindings {
			walkStrictFallback(fnName, b.Value, child, out)
		}
		walkStrictFallback(fnName, n.Body, child, out)
	case *core.Lambda:
		walkStrictFallback(fnName, n.Body, env, out)
	case *core.If:
		walkStrictFallback(fnName, n.Cond, env, out)
		walkStrictFallback(fnName, n.Then, env, out)
		walkStrictFallback(fnName, n.Else, env, out)
	case *core.Match:
		walkStrictFallback(fnName, n.Scrutinee, env, out)
		for _, arm := range n.Arms {
			walkStrictFallback(fnName, arm.Guard, env, out)
			walkStrictFallback(fnName, arm.Body, env, out)
		}
	case *core.BinOp:
		walkStrictFallback(fnName, n.Left, env, out)
		walkStrictFallback(fnName, n.Right, env, out)
	case *core.UnOp:
		walkStrictFallback(fnName, n.Operand, env, out)
	case *core.Intrinsic:
		for _, a := range n.Args {
			walkStrictFallback(fnName, a, env, out)
		}
	case *core.Record:
		for _, v := range n.Fields {
			walkStrictFallback(fnName, v, env, out)
		}
	case *core.RecordAccess:
		walkStrictFallback(fnName, n.Record, env, out)
	case *core.RecordUpdate:
		walkStrictFallback(fnName, n.Base, env, out)
		for _, v := range n.Updates {
			walkStrictFallback(fnName, v, env, out)
		}
	case *core.List:
		for _, el := range n.Elements {
			walkStrictFallback(fnName, el, env, out)
		}
	case *core.Array:
		for _, el := range n.Elements {
			walkStrictFallback(fnName, el, env, out)
		}
	case *core.Tuple:
		for _, el := range n.Elements {
			walkStrictFallback(fnName, el, env, out)
		}
	default:
		// Atomic leaves (Var, VarGlobal, Lit): nothing to recurse into.
	}
}

// cloneEnv returns a shallow copy of an env map so a nested scope's bindings do
// not leak back into the parent scope.
func cloneEnv(env map[string]core.CoreExpr) map[string]core.CoreExpr {
	out := make(map[string]core.CoreExpr, len(env)+1)
	for k, v := range env {
		out[k] = v
	}
	return out
}
