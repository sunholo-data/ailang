// Package pipeline provides compilation passes for AILANG
package pipeline

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-SMT-INTERP-SHOW
//
// String interpolation is desugared by the parser (parser_literals.go) into a
// concat_String chain with every hole wrapped in `show(...)`. That happens at
// PARSE time, before any type is known, so the wrapper lands unconditionally —
// including around holes that are already `string`, where `show` is the
// identity (internal/builtins/show.go:118-120, "Return string without quotes").
//
// `show` is a polymorphic $builtin with no SMT-LIB encoding, so its presence put
// every string-building function outside the Z3-decidable fragment: `"X${a}"`
// skipped where the identical `concat_String("X", a)` verified.
//
// ShowNormalizer removes `show` wherever the ARGUMENT TYPE makes it redundant or
// trivially encodable. It runs after monomorphization, where CoreTypeInfo has
// resolved a generic helper's instantiations, and it rewrites into Core shapes
// the SMT encoder already accepts:
//
//	string -> the argument itself           (identity; no verifier change needed)
//	bool   -> if x then "true" else "false" (core.If + string literals already encode)
//
// Anything else — float, list, record, ADT, a type variable — is left alone.
// That is the honest residue, not a failure.
//
// The pass is deliberately in the shared pipeline rather than local to `verify`:
// a verify-local rewrite would have the verifier prove properties of a Core tree
// the evaluator never executes. Keeping it here means "what was proved" and
// "what runs" are the same tree, and the SMT layer stays untyped.
type ShowNormalizer struct {
	coreTI     *types.CoreTypeInfo
	nextNodeID uint64

	// Elided counts show calls removed outright (string holes).
	Elided int
	// Rewritten counts show calls replaced by an encodable equivalent (bool holes).
	Rewritten int
	// Residue counts show calls deliberately left in place (unsupported types).
	Residue int

	// residueByFunc collects, per enclosing function, the argument types of the
	// `show` calls left unrewritten. Written into DeclMeta.ShowResidue so the
	// SMT layer — which has no type information of its own — can name the type
	// in its skip message instead of emitting a bare "unencodable builtin: show"
	// that sends readers hunting a call the source does not contain.
	residueByFunc map[string][]core.ShowResidueNote
}

// NewShowNormalizer creates the pass. coreTI must be the live type table for the
// program being normalized: the pass both reads argument types from it and
// registers types for the nodes it mints.
func NewShowNormalizer(coreTI *types.CoreTypeInfo) *ShowNormalizer {
	return &ShowNormalizer{coreTI: coreTI, residueByFunc: make(map[string][]core.ShowResidueNote)}
}

// Normalize returns a new program with redundant `show` calls removed.
//
// It returns an error — it does not degrade — when a `show` argument has no
// CoreTypeInfo entry. ValidateCoreTypeInfo runs earlier in the same pipeline and
// already declares that every Core node must have one, calling a miss "a
// compiler bug"; absorbing it here would surface later as an ordinary
// SMT-encodability skip and mask the real fault. A type that is PRESENT but
// unsupported is a different case entirely, and leaves `show` in place.
func (s *ShowNormalizer) Normalize(prog *core.Program) (*core.Program, error) {
	if prog == nil || s.coreTI == nil {
		return prog, nil
	}

	// Mint IDs above everything already in the tree, so a minted node can never
	// collide with one the elaborator or the specializer allocated.
	s.nextNodeID = maxCoreNodeID(prog) + 1

	out := &core.Program{
		Decls: make([]core.CoreExpr, len(prog.Decls)),
		Meta:  prog.Meta,
	}
	for i, decl := range prog.Decls {
		rewritten, err := s.rewriteDecl(decl)
		if err != nil {
			return nil, err
		}
		out.Decls[i] = rewritten
	}

	// Publish the residue notes so the SMT layer can name the blocking type.
	// Assigned (not appended) so re-running the pass on the same program is
	// idempotent rather than accumulating duplicates.
	for fnName, notes := range s.residueByFunc {
		if meta, ok := out.Meta[fnName]; ok && meta != nil {
			meta.ShowResidue = notes
		}
	}
	return out, nil
}

// rewriteDecl walks a top-level declaration, tracking the enclosing function
// name so diagnostics can name it. Decls are *core.LetRec (the common case) or
// *core.Let — the same shapes findFunctionBody keys on.
func (s *ShowNormalizer) rewriteDecl(decl core.CoreExpr) (core.CoreExpr, error) {
	switch d := decl.(type) {
	case *core.LetRec:
		bindings := make([]core.RecBinding, len(d.Bindings))
		for i, b := range d.Bindings {
			v, err := s.expr(b.Name, b.Value)
			if err != nil {
				return nil, err
			}
			bindings[i] = core.RecBinding{Name: b.Name, Value: v}
		}
		body, err := s.expr("", d.Body)
		if err != nil {
			return nil, err
		}
		return &core.LetRec{CoreNode: d.CoreNode, Bindings: bindings, Body: body}, nil

	case *core.Let:
		value, err := s.expr(d.Name, d.Value)
		if err != nil {
			return nil, err
		}
		body, err := s.expr("", d.Body)
		if err != nil {
			return nil, err
		}
		return &core.Let{CoreNode: d.CoreNode, Name: d.Name, Value: value, Body: body}, nil

	default:
		return s.expr("", decl)
	}
}

// showArg returns the sole argument of an `$builtin.show` application.
//
// The match is structural — a VarGlobal whose Ref is exactly $builtin.show —
// never by bare name. A name-only match would also catch a user's own binding
// called `show`, and the REPL's Show instance for strings is NOT the identity
// (it quotes, repl_commands.go:336), so keying off the name would be unsound.
func showArg(app *core.App) (core.CoreExpr, bool) {
	vg, ok := app.Func.(*core.VarGlobal)
	if !ok {
		return nil, false
	}
	if vg.Ref.Module != "$builtin" || vg.Ref.Name != "show" || len(app.Args) != 1 {
		return nil, false
	}
	return app.Args[0], true
}

// expr rewrites one expression. fnName is the enclosing function, used only for
// diagnostics.
func (s *ShowNormalizer) expr(fnName string, e core.CoreExpr) (core.CoreExpr, error) {
	if e == nil {
		return nil, nil
	}

	switch n := e.(type) {
	case *core.App:
		if arg, isShow := showArg(n); isShow {
			return s.rewriteShow(fnName, n, arg)
		}
		fn, err := s.expr(fnName, n.Func)
		if err != nil {
			return nil, err
		}
		args, err := s.exprs(fnName, n.Args)
		if err != nil {
			return nil, err
		}
		return &core.App{CoreNode: n.CoreNode, Func: fn, Args: args}, nil

	case *core.Let:
		value, err := s.expr(fnName, n.Value)
		if err != nil {
			return nil, err
		}
		body, err := s.expr(fnName, n.Body)
		if err != nil {
			return nil, err
		}
		return &core.Let{CoreNode: n.CoreNode, Name: n.Name, Value: value, Body: body}, nil

	case *core.LetRec:
		bindings := make([]core.RecBinding, len(n.Bindings))
		for i, b := range n.Bindings {
			v, err := s.expr(b.Name, b.Value)
			if err != nil {
				return nil, err
			}
			bindings[i] = core.RecBinding{Name: b.Name, Value: v}
		}
		body, err := s.expr(fnName, n.Body)
		if err != nil {
			return nil, err
		}
		return &core.LetRec{CoreNode: n.CoreNode, Bindings: bindings, Body: body}, nil

	case *core.Lambda:
		body, err := s.expr(fnName, n.Body)
		if err != nil {
			return nil, err
		}
		return &core.Lambda{CoreNode: n.CoreNode, Params: n.Params, Body: body}, nil

	case *core.If:
		cond, err := s.expr(fnName, n.Cond)
		if err != nil {
			return nil, err
		}
		then, err := s.expr(fnName, n.Then)
		if err != nil {
			return nil, err
		}
		els, err := s.expr(fnName, n.Else)
		if err != nil {
			return nil, err
		}
		return &core.If{CoreNode: n.CoreNode, Cond: cond, Then: then, Else: els}, nil

	case *core.Match:
		scrutinee, err := s.expr(fnName, n.Scrutinee)
		if err != nil {
			return nil, err
		}
		arms := make([]core.MatchArm, len(n.Arms))
		for i, arm := range n.Arms {
			guard, err := s.expr(fnName, arm.Guard)
			if err != nil {
				return nil, err
			}
			body, err := s.expr(fnName, arm.Body)
			if err != nil {
				return nil, err
			}
			arms[i] = core.MatchArm{Pattern: arm.Pattern, Guard: guard, Body: body}
		}
		return &core.Match{CoreNode: n.CoreNode, Scrutinee: scrutinee, Arms: arms, Exhaustive: n.Exhaustive}, nil

	case *core.Record:
		fields := make(map[string]core.CoreExpr, len(n.Fields))
		for k, v := range n.Fields {
			rv, err := s.expr(fnName, v)
			if err != nil {
				return nil, err
			}
			fields[k] = rv
		}
		return &core.Record{CoreNode: n.CoreNode, Fields: fields}, nil

	case *core.RecordAccess:
		rec, err := s.expr(fnName, n.Record)
		if err != nil {
			return nil, err
		}
		return &core.RecordAccess{CoreNode: n.CoreNode, Record: rec, Field: n.Field}, nil

	case *core.List:
		elems, err := s.exprs(fnName, n.Elements)
		if err != nil {
			return nil, err
		}
		return &core.List{CoreNode: n.CoreNode, Elements: elems}, nil

	case *core.BinOp:
		left, err := s.expr(fnName, n.Left)
		if err != nil {
			return nil, err
		}
		right, err := s.expr(fnName, n.Right)
		if err != nil {
			return nil, err
		}
		return &core.BinOp{CoreNode: n.CoreNode, Op: n.Op, Left: left, Right: right}, nil

	case *core.UnOp:
		operand, err := s.expr(fnName, n.Operand)
		if err != nil {
			return nil, err
		}
		return &core.UnOp{CoreNode: n.CoreNode, Op: n.Op, Operand: operand}, nil

	case *core.Intrinsic:
		args, err := s.exprs(fnName, n.Args)
		if err != nil {
			return nil, err
		}
		return &core.Intrinsic{CoreNode: n.CoreNode, Op: n.Op, Args: args}, nil

	default:
		// Leaves (Var, VarGlobal, Lit, ...) and any node with no sub-expressions
		// pass through untouched.
		return e, nil
	}
}

func (s *ShowNormalizer) exprs(fnName string, in []core.CoreExpr) ([]core.CoreExpr, error) {
	out := make([]core.CoreExpr, len(in))
	for i, e := range in {
		r, err := s.expr(fnName, e)
		if err != nil {
			return nil, err
		}
		out[i] = r
	}
	return out, nil
}

// rewriteShow decides what to do with one `show(arg)` application.
func (s *ShowNormalizer) rewriteShow(fnName string, app *core.App, arg core.CoreExpr) (core.CoreExpr, error) {
	argType, has := s.coreTI.Get(arg.ID())
	if !has {
		// A broken compiler invariant, not an unsupported type. Fail loudly.
		return nil, fmt.Errorf(
			"show-normalize: function %q applies show to node %d, which has no CoreTypeInfo entry; "+
				"every Core node must carry a type by this point (see ValidateCoreTypeInfo). "+
				"This is a compiler bug in an earlier pass, not an unencodable program",
			describeFunc(fnName), arg.ID())
	}

	// Rewrite the argument first, so a nested show(show(x)) collapses fully.
	newArg, err := s.expr(fnName, arg)
	if err != nil {
		return nil, err
	}

	switch {
	case isPrimType(argType, "string"):
		// show(s: string) is the identity. Splice the argument in directly; it
		// keeps its own node ID and CoreTypeInfo entry.
		s.Elided++
		return newArg, nil

	case isPrimType(argType, "bool"):
		s.Rewritten++
		return s.boolToString(newArg, app.OrigSpan), nil

	case isPrimType(argType, "int"):
		s.Rewritten++
		return s.intToString(newArg, argType, app.OrigSpan), nil

	default:
		// float, list, record, ADT, type variable: no encodable equivalent.
		// Leave show in place — the residue is honest — and record the type so
		// the verifier can say WHICH type blocked it.
		s.Residue++
		s.recordResidue(fnName, argType)
		return &core.App{CoreNode: app.CoreNode, Func: app.Func, Args: []core.CoreExpr{newArg}}, nil
	}
}

// recordResidue notes one `show` left in place, and the argument type that
// caused it. Only the type is recorded — never the origin, because the same
// Core shape comes from a "${x}" hole and from an explicit user `show(x)` call
// and nothing in Core distinguishes them.
func (s *ShowNormalizer) recordResidue(fnName string, argType types.Type) {
	if fnName == "" {
		return
	}
	s.residueByFunc[fnName] = append(s.residueByFunc[fnName],
		core.ShowResidueNote{ArgType: renderTypeName(argType)})
}

// renderTypeName gives a short, user-facing name for a type. Named constructors
// render as themselves ("float", "Option"); anything else falls back to the
// type's own rendering rather than a guess.
func renderTypeName(ty types.Type) string {
	if con, ok := ty.(*types.TCon); ok {
		return con.Name
	}
	if ty == nil {
		return "value of unknown type"
	}
	return ty.String()
}

// boolToString builds `if cond then "true" else "false"`, matching show's own
// rendering (internal/builtins/show.go: BoolValue -> "true"/"false").
//
// Every minted node gets a fresh ID AND a CoreTypeInfo entry: ValidateCoreTypeInfo
// requires one for every node it walks, literals included. `cond` is reused
// as-is and is deliberately NOT re-registered — it already has a valid entry,
// and overwriting it with `string` would corrupt the table.
func (s *ShowNormalizer) boolToString(cond core.CoreExpr, span ast.Pos) core.CoreExpr {
	stringType := types.Type(&types.TCon{Name: "string"})

	then := &core.Lit{CoreNode: s.mintNode(span), Kind: core.StringLit, Value: "true"}
	els := &core.Lit{CoreNode: s.mintNode(span), Kind: core.StringLit, Value: "false"}
	ifNode := &core.If{CoreNode: s.mintNode(span), Cond: cond, Then: then, Else: els}

	s.coreTI.Set(then.ID(), stringType)
	s.coreTI.Set(els.ID(), stringType)
	s.coreTI.Set(ifNode.ID(), stringType)

	return ifNode
}

// intToString builds `$builtin._string_intToStr(n)`, whose runtime behavior is
// strconv.Itoa — identical to show's rendering for ints (verified for 42, -5 and
// 0). The SMT side encodes it with an explicit sign branch, because Z3's
// str.from_int returns "" for a negative argument; see
// StringBuiltinSpecial["_string_intToStr"].
//
// The reference is deliberately the $builtin one, NOT the std/string wrapper:
// AddBuiltinsToGlobalEnv binds every registered builtin under $builtin
// (elaborate/core.go:137), so the rewritten Core carries no module dependency
// and a program that never imports std/string still links and runs. Emitting
// `std/string.intToStr` here would break exactly those programs.
//
// Both minted nodes are registered: the App as `string`, the callee as
// `int -> string`. The argument keeps its own node and entry.
func (s *ShowNormalizer) intToString(arg core.CoreExpr, argType types.Type, span ast.Pos) core.CoreExpr {
	stringType := types.Type(&types.TCon{Name: "string"})

	callee := &core.VarGlobal{
		CoreNode: s.mintNode(span),
		Ref:      core.GlobalRef{Module: "$builtin", Name: "_string_intToStr"},
	}
	app := &core.App{
		CoreNode: s.mintNode(span),
		Func:     callee,
		Args:     []core.CoreExpr{arg},
	}

	s.coreTI.Set(callee.ID(), &types.TFunc2{Params: []types.Type{argType}, Return: stringType})
	s.coreTI.Set(app.ID(), stringType)

	return app
}

// mintNode allocates a fresh Core node header, preserving the surface position
// of the interpolation it replaces so diagnostics still point at real source.
func (s *ShowNormalizer) mintNode(span ast.Pos) core.CoreNode {
	id := s.nextNodeID
	s.nextNodeID++
	return core.CoreNode{NodeID: id, CoreSpan: span, OrigSpan: span}
}

// isPrimType reports whether ty is the named primitive type constructor.
// Casing is tolerated because both "string" and "String" circulate in the type
// system (see internal/types/typechecker_operators.go:528).
func isPrimType(ty types.Type, name string) bool {
	con, ok := ty.(*types.TCon)
	if !ok {
		return false
	}
	return strings.EqualFold(con.Name, name)
}

// describeFunc renders a function name for diagnostics, naming the anonymous
// case rather than emitting an empty string.
func describeFunc(name string) string {
	if name == "" {
		return "<top-level>"
	}
	return name
}

// maxCoreNodeID returns the largest node ID in the program, so minted nodes can
// start above every existing one.
func maxCoreNodeID(prog *core.Program) uint64 {
	var max uint64
	for _, decl := range prog.Decls {
		walkCore(decl, func(e core.CoreExpr) {
			if id := e.ID(); id > max {
				max = id
			}
		})
	}
	return max
}
