package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-V1-RIGID-SIGNATURES (#2273): the elaborator decides which type variables
// of a declaration's signature are rigid, and gives each a name unique to the
// declaration (types/rigid_sig.go has the rule and the check).
//
// A top-level `func` introduces its explicit `[a]` parameters and every
// lowercase variable of its parameter and return annotations that is not
// already in scope. While its body is elaborated those names stay in scope, so
// a `let x: a` or a function literal's `(y: a)` inside it denotes the same
// variable. A variable an enclosing signature introduced is that variable;
// an explicit `[a]` shadows it.

// sigTyvarScope is one declaration's written-name -> unique-name frame.
type sigTyvarScope map[string]string

// lookupSigTyvar returns the unique name a written type variable denotes in
// the open signature scopes, innermost first.
func (e *Elaborator) lookupSigTyvar(name string) (string, bool) {
	for i := len(e.sigTyvars) - 1; i >= 0; i-- {
		if u, ok := e.sigTyvars[i][name]; ok {
			return u, true
		}
	}
	return "", false
}

// openSignature pushes the signature scope of fn and returns its rigid
// variables (nil when it introduces none). The caller pops with
// closeSignature once the body and the annotations are elaborated.
func (e *Elaborator) openSignature(fn *ast.FuncDecl) *types.RigidSig {
	frame := sigTyvarScope{}
	e.sigTyvars = append(e.sigTyvars, frame)
	if fn == nil {
		return nil
	}
	explicit := make(map[string]bool, len(fn.TypeParams))
	for _, p := range fn.TypeParams {
		explicit[p] = true
	}
	var occ []*ast.TypeVar
	rowVars := map[string]bool{}
	for _, p := range fn.Params {
		e.collectSigTypeVars(p.Type, &occ, rowVars)
	}
	e.collectSigTypeVars(fn.ReturnType, &occ, rowVars)
	// Only variables that occur in the parameter and return types are rigid: an
	// explicit parameter used only as an effect row (`[a, e]` with `! {e}`) or
	// not at all has no type position to keep general.
	first := map[string]ast.Pos{}
	uses := map[string]int{}
	var order []string
	for _, tv := range occ {
		uses[tv.Name]++
		if _, ok := first[tv.Name]; ok {
			continue
		}
		first[tv.Name] = tv.Pos
		order = append(order, tv.Name)
	}
	sig := &types.RigidSig{Func: fn.Name, Pos: fn.Pos}
	var candidates []string
	for _, name := range order {
		if !explicit[name] {
			if _, outer := e.lookupSigTyvar(name); outer {
				continue // an enclosing signature's variable: rigid there
			}
		}
		if e.rigidIssued == nil {
			e.rigidIssued = map[string]int{}
		}
		key := name + "@" + fn.Name
		e.rigidIssued[key]++
		unique := types.RigidTypeVarName(name, fn.Name, e.rigidIssued[key])
		frame[name] = unique
		v := types.RigidVar{Name: unique, Display: name, Explicit: explicit[name], Pos: first[name], Row: rowVars[name]}
		if !v.Explicit {
			// A value name used as a type (`-> (stack, int)`). Not for a single
			// letter or a name used as a type more than once: `cmp(a: a, b: b)`
			// means type variables, it just reuses the parameter names.
			for _, p := range fn.Params {
				if p.Name == name && len(name) > 1 && uses[name] == 1 {
					v.ParamName = true
				}
			}
			if candidates == nil {
				candidates = e.rigidHintCandidates()
			}
			v.Suggest = types.SuggestTypeForVar(name, candidates, importhint.EditDistance)
		}
		sig.Vars = append(sig.Vars, v)
	}
	if len(sig.Vars) == 0 {
		return nil
	}
	return sig
}

// closeSignature pops the scope openSignature pushed and records sig for the
// lambda the declaration elaborated to.
func (e *Elaborator) closeSignature(sig *types.RigidSig, lam core.CoreExpr) {
	e.sigTyvars = e.sigTyvars[:len(e.sigTyvars)-1]
	l, ok := lam.(*core.Lambda)
	if sig == nil || !ok {
		return
	}
	if e.rigidSigs == nil {
		e.rigidSigs = map[uint64]*types.RigidSig{}
	}
	e.rigidSigs[l.ID()] = sig
}

// GetRigidSignatures returns each declaration lambda's rigid signature
// variables (Lambda NodeID -> signature), for CoreTypeChecker.SetRigidSignatures.
func (e *Elaborator) GetRigidSignatures() map[uint64]*types.RigidSig {
	return e.rigidSigs
}

// rigidHintCandidates is every type name a misspelled or miscased signature
// variable may have meant: the primitives and the types in scope.
func (e *Elaborator) rigidHintCandidates() []string {
	out := ast.BuiltinTypeNames()
	if e.typeScope != nil {
		out = append(out, e.typeScope.Names()...)
	}
	return out
}

// collectSigTypeVars appends every type variable occurrence of t (record row
// variables included, and also named in rowVars), in source order: the
// TypeVar nodes and each SimpleType that reads as a type variable.
func (e *Elaborator) collectSigTypeVars(t ast.Type, out *[]*ast.TypeVar, rowVars map[string]bool) {
	switch typ := t.(type) {
	case nil:
	case *ast.SimpleType:
		// #2286's contract (types.TypeScope.ReadsAsTypeVar): a lowercase name
		// imported by name that is a value, not a type, is a type variable.
		if !ast.IsBuiltinTypeName(typ.Name) && e.simpleTypeIsVar(typ.Name) {
			*out = append(*out, &ast.TypeVar{Name: typ.Name, Pos: typ.Pos})
		}
	case *ast.TypeVar:
		*out = append(*out, typ)
	case *ast.FuncType:
		for _, p := range typ.Params {
			e.collectSigTypeVars(p, out, rowVars)
		}
		e.collectSigTypeVars(typ.Return, out, rowVars)
	case *ast.ListType:
		e.collectSigTypeVars(typ.Element, out, rowVars)
	case *ast.ArrayType:
		e.collectSigTypeVars(typ.Element, out, rowVars)
	case *ast.TupleType:
		for _, el := range typ.Elements {
			e.collectSigTypeVars(el, out, rowVars)
		}
	case *ast.TypeApp:
		for _, a := range typ.Args {
			e.collectSigTypeVars(a, out, rowVars)
		}
	case *ast.RecordType:
		for _, f := range typ.Fields {
			e.collectSigTypeVars(f.Type, out, rowVars)
		}
		if typ.Row != nil {
			*out = append(*out, typ.Row)
			if rowVars != nil {
				rowVars[typ.Row.Name] = true
			}
		}
	case *ast.LabelledType:
		e.collectSigTypeVars(typ.Base, out, rowVars)
	default:
		panic(fmt.Sprintf("collectSigTypeVars: unhandled ast.Type %T", t))
	}
}
