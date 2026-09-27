package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/traverse"
)

// Package effect ceiling (M-PKG, M-EFFECT-CEILING-REACHABLE).
//
// RULE: the [effects].max ceiling bounds the effects the package's OWN code
// can reach. It is checked on every module that belongs to the package being
// compiled (isOwnPackageModule), and on nothing else. For each such module the
// charged set is the union of
//
//	(a) every function's declared effect row (the surface signature), and
//	(b) every effect label anywhere in the inferred type of each AUTHORITY
//	    ENTRY POINT in the module's Core: a reference to a global outside the
//	    package (a VarGlobal into std/, pkg/<dep>/, $builtin, $adt) — whether
//	    it is called, passed to a higher-order function, stored in a record or
//	    returned — plus every Intrinsic and DictRef node.
//
// Soundness argument. Effectful behaviour enters a program only through a
// builtin or a function defined somewhere. Every name the package's code uses
// is either (i) a local — a parameter or let binding, whose value came from
// the caller or from another expression of this module, which is itself
// walked; (ii) a top-level function of an own module, whose body is walked
// and whose declared row is checked when THAT module is checked (own modules
// are compiled in the same run, or served from a cache entry keyed on this
// very ceiling — ceilingCacheDigest); or (iii) anything else, which the
// elaborator resolves to a VarGlobal (imports and builtins alike — see
// AddBuiltinsToGlobalEnv) whose instantiated type at that node is in
// CoreTypeInfo (the M-DX4 invariant, enforced by ValidateCoreTypeInfo just
// before this check). (b) charges every (iii) wherever it occurs — including
// inside a lambda that is never called — so the ceiling does not depend on how
// the effect checker attributes a closure's effects, and stays sound if
// ailang-core-triage/stored-lambda-effects-charge-enclosing-fn.md later stops
// charging a never-called closure to its enclosing function. A caller-supplied
// callback (case i) is the caller's authority; performing it still requires
// declaring its effects, which (a) checks.
//
// Why not charge every node's type: a hook-framework package (the motoko ext
// ABI) must give its hooks the ABI's fixed rows ({AI, Clock, IO, Net, ...})
// even when their bodies use only {Process, FS, Env}. Charging the types of
// its own local references flagged seven published motoko-ext packages that
// reach none of those effects.
//
// What is NOT charged: the bodies of imported std/ or dependency modules.
// Their effects reach the package only through a reference in the package's
// own code, which (b) sees. Charging every function of every imported module
// — the pre-fix behaviour, which fired while compiling std/stream itself —
// made a package that imports only `disconnect` declare Process.

// isOwnPackageModule reports whether modID belongs to the package being
// compiled. $builtin/$adt are compiler-provided; std/ modules are the stdlib;
// pkg/<dep>/... are dependencies, whose own ceilings are theirs;
// pkg/<self>/... is an intra-package import of the package's own sibling
// module and IS checked. Every other module ID is a local path inside the
// package and is checked (conservative: an unexpected ID is treated as own
// code, never silently skipped).
func isOwnPackageModule(modID, pkgName string) bool {
	if strings.HasPrefix(modID, "$") {
		return false
	}
	if modID == "std" || strings.HasPrefix(modID, "std/") {
		return false
	}
	if strings.HasPrefix(modID, "pkg/") {
		return pkgName != "" && strings.HasPrefix(modID, "pkg/"+pkgName+"/")
	}
	return true
}

// ceilingCacheDigest is the compile-cache key component for modID's ceiling.
// A module that passed a wide ceiling must not be served after the manifest
// narrows it, so every own-package module's cache key carries the package
// name and its sorted [effects].max. Returns ok=false when no ceiling applies
// to modID (its key is then unchanged).
func ceilingCacheDigest(modID string) (string, bool) {
	if currentPackageManifest == nil || currentPackageManifest.Effects.Max == nil {
		return "", false
	}
	pkgName := currentPackageManifest.Package.Name
	if !isOwnPackageModule(modID, pkgName) {
		return "", false
	}
	maxEffects := append([]string(nil), currentPackageManifest.Effects.Max...)
	sort.Strings(maxEffects)
	return pkgName + ":[" + strings.Join(maxEffects, ",") + "]", true
}

// validateEffectCeiling enforces the current package's [effects].max on one
// compiled module. Only applies when an ailang.toml with a ceiling exists.
func validateEffectCeiling(surfaceAST *ast.File, prog *core.Program, coreTI types.CoreTypeInfo, modID string) error {
	if currentPackageManifest == nil || surfaceAST == nil {
		return nil
	}
	maxEffects := currentPackageManifest.Effects.Max
	if maxEffects == nil {
		return nil // No ceiling declared
	}
	pkgName := currentPackageManifest.Package.Name
	if !isOwnPackageModule(modID, pkgName) {
		return nil
	}

	// (a) Declared rows — the documented contract, reported first because the
	// fix (widen max or narrow the signature) is the most direct.
	for _, fn := range surfaceAST.Funcs {
		if err := pkg.CheckEffectCeiling(pkgName, ast.EffectNames(fn.Effects), maxEffects); err != nil {
			return fmt.Errorf("in function %s in %s: %w", fn.Name, modID, err)
		}
	}

	// (b) Authority entry points anywhere in the module.
	if prog == nil {
		return nil
	}
	w := &ceilingWalker{coreTI: coreTI, pkgName: pkgName}
	for _, decl := range prog.Decls {
		w.walkTopLevel(decl)
		if w.err != nil {
			return w.err
		}
	}
	for _, h := range w.hits {
		if err := pkg.CheckEffectCeiling(pkgName, h.effects, maxEffects); err != nil {
			return fmt.Errorf("in %s in %s (%s): %w", h.owner, modID, h.what, err)
		}
	}
	return nil
}

// ceilingHit is one authority entry point whose type carries effects.
type ceilingHit struct {
	owner   string   // enclosing top-level binding
	what    string   // human description of the expression
	effects []string // concrete effect labels in its type
}

// ceilingWalker visits every Core node exhaustively and records each
// authority entry point whose inferred type carries concrete effect labels.
type ceilingWalker struct {
	coreTI  types.CoreTypeInfo
	pkgName string
	hits    []ceilingHit
	err     error
}

// walkTopLevel descends the Let/LetRec chain of a top-level declaration so
// each binding's value is attributed to its own name.
func (w *ceilingWalker) walkTopLevel(decl core.CoreExpr) {
	for decl != nil && w.err == nil {
		switch d := decl.(type) {
		case *core.Let:
			w.walk(d.Value, "function "+d.Name)
			decl = d.Body
		case *core.LetRec:
			for _, b := range d.Bindings {
				w.walk(b.Value, "function "+b.Name)
			}
			decl = d.Body
		default:
			w.walk(decl, "top-level expression")
			return
		}
	}
}

// isAuthorityEntry reports whether expr can introduce authority from outside
// the package: a reference to a non-own global, or an Intrinsic/DictRef.
func (w *ceilingWalker) isAuthorityEntry(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.VarGlobal:
		return !isOwnPackageModule(e.Ref.Module, w.pkgName)
	case *core.Intrinsic, *core.DictRef:
		return true
	default:
		return false
	}
}

// record charges the effect labels of an authority entry point's inferred
// type to owner. An entry point without type info would break the soundness
// argument, so it fails loudly instead of being skipped.
func (w *ceilingWalker) record(expr core.CoreExpr, owner string) {
	if !w.isAuthorityEntry(expr) {
		return
	}
	t, ok := w.coreTI.Get(expr.ID())
	if !ok {
		w.err = fmt.Errorf("effect ceiling: %s in %s has no type information; cannot bound its effects", describeCeilingExpr(expr), owner)
		return
	}
	if effs := typeEffectLabels(t); len(effs) > 0 {
		w.hits = append(w.hits, ceilingHit{owner: owner, what: describeCeilingExpr(expr), effects: effs})
	}
}

// walk visits expr and all its sub-expressions. Exhaustive over Core node
// kinds: an unknown kind is an error, never a silent skip.
func (w *ceilingWalker) walk(expr core.CoreExpr, owner string) {
	if expr == nil || w.err != nil {
		return
	}
	w.record(expr, owner)
	switch e := expr.(type) {
	case *core.Var, *core.VarGlobal, *core.Lit, *core.DictRef:
		// leaves
	case *core.Lambda:
		w.walk(e.Body, owner)
	case *core.Let:
		w.walk(e.Value, owner)
		w.walk(e.Body, owner)
	case *core.LetRec:
		for _, b := range e.Bindings {
			w.walk(b.Value, owner)
		}
		w.walk(e.Body, owner)
	case *core.App:
		w.walk(e.Func, owner)
		w.walkAll(e.Args, owner)
	case *core.If:
		w.walk(e.Cond, owner)
		w.walk(e.Then, owner)
		w.walk(e.Else, owner)
	case *core.Match:
		w.walk(e.Scrutinee, owner)
		for _, arm := range e.Arms {
			w.walk(arm.Guard, owner)
			w.walk(arm.Body, owner)
		}
	case *core.BinOp:
		w.walk(e.Left, owner)
		w.walk(e.Right, owner)
	case *core.UnOp:
		w.walk(e.Operand, owner)
	case *core.Intrinsic:
		w.walkAll(e.Args, owner)
	case *core.Record:
		for _, name := range sortedFieldNames(e.Fields) {
			w.walk(e.Fields[name], owner)
		}
	case *core.RecordAccess:
		w.walk(e.Record, owner)
	case *core.RecordUpdate:
		w.walk(e.Base, owner)
		for _, name := range sortedFieldNames(e.Updates) {
			w.walk(e.Updates[name], owner)
		}
	case *core.List:
		w.walkAll(e.Elements, owner)
	case *core.Array:
		w.walkAll(e.Elements, owner)
	case *core.Tuple:
		w.walkAll(e.Elements, owner)
	case *core.DictAbs:
		w.walk(e.Body, owner)
	case *core.DictApp:
		w.walk(e.Dict, owner)
		w.walkAll(e.Args, owner)
	default:
		w.err = fmt.Errorf("effect ceiling: unhandled Core expression %T in %s; cannot bound its effects", expr, owner)
	}
}

func (w *ceilingWalker) walkAll(exprs []core.CoreExpr, owner string) {
	for _, e := range exprs {
		w.walk(e, owner)
	}
}

func sortedFieldNames(m map[string]core.CoreExpr) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// typeEffectLabels returns every concrete effect label of every function type
// nested anywhere in t (parameters, returns, record fields, type arguments).
// Row-variable tails are polymorphic and contribute nothing: a concrete
// instantiation is its own node wherever it happens. Cycle-safe: uses
// traverse.Walk.
func typeEffectLabels(t types.Type) []string {
	seen := map[string]bool{}
	traverse.Walk(t, func(n types.Type) {
		fn, ok := n.(*types.TFunc2)
		if !ok || fn.EffectRow == nil {
			return
		}
		for label := range fn.EffectRow.Labels {
			seen[label] = true
		}
	})
	out := make([]string, 0, len(seen))
	for l := range seen {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

func describeCeilingExpr(expr core.CoreExpr) string {
	switch e := expr.(type) {
	case *core.VarGlobal:
		return fmt.Sprintf("reference to %s.%s at %s", e.Ref.Module, e.Ref.Name, e.OriginalSpan())
	case *core.DictRef:
		return fmt.Sprintf("dictionary %s[%s] at %s", e.ClassName, e.TypeName, e.OriginalSpan())
	default:
		return fmt.Sprintf("%T at %s", expr, expr.OriginalSpan())
	}
}
