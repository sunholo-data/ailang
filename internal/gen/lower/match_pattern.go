package lower

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// Recursive pattern lowering (m-vm-match-lowering).
//
// patternCond and patternBindings are the Statement IR mirror of the
// evaluator's matchPattern (internal/eval/eval_patterns.go), which is the
// reference semantics: every pattern kind, at any nesting depth, in every
// position. Before this pair existed the lowering handled a flat subset and
// silently dropped the rest — nested bindings went unbound (#1420, #1505,
// #1517), literal and nested sub-patterns were never checked (silent wrong
// results), and constructor tags were read as record fields (#1503).
//
// Both functions walk the same access paths from the scrutinee:
//
//	constructor arg j   FieldAccess{s, "_j"}     (positional GET_FIELD)
//	tuple element i     FieldAccess{s, "_i"}     (positional GET_FIELD)
//	record field k      FieldAccess{s, k}        (resolved by name)
//	list element i      _list_get(s, i)
//	list tail after n   _list_tail(s, n)
//
// The condition is one short-circuit && chain, ordered so every access is
// guarded by the checks that make it safe (the length check precedes the
// element reads, the tag check precedes the field reads). Bindings are only
// evaluated after the condition holds, so they never trap.
//
// Structure the type checker guarantees (tuple arity, record field
// presence, constructor arity) is not re-checked — the evaluator's checks
// for those can only fail on ill-typed input.

// patternCond returns the boolean "scrutinee s matches pat" expression, or
// nil when the pattern always matches. t is s's type when known (nil
// otherwise); it only feeds ADTTagEq.TypeName.
func patternCond(s stmt.Expr, t types.Type, pat core.CorePattern) stmt.Expr {
	switch p := pat.(type) {
	case *core.VarPattern, *core.WildcardPattern:
		return nil

	case *core.LitPattern:
		return lowerLitPatternCond(s, p)

	case *core.ConstructorPattern:
		cond := stmt.Expr(stmt.ADTTagEq{Value: s, TypeName: extractADTName(t), Tag: p.Name})
		for j, arg := range p.Args {
			cond = andCond(cond, patternCond(ctorField(s, j), nil, arg))
		}
		return cond

	case *core.TuplePattern:
		var cond stmt.Expr
		for i, el := range p.Elements {
			cond = andCond(cond, patternCond(tupleField(s, i), tupleElemType(t, i), el))
		}
		return cond

	case *core.RecordPattern:
		var cond stmt.Expr
		for _, k := range sortedFieldNames(p) {
			cond = andCond(cond, patternCond(recordField(s, k), recordFieldType(t, k), p.Fields[k]))
		}
		return cond

	case *core.ListPattern:
		p = normalizeListPattern(p)
		n := int64(len(p.Elements))
		var cond stmt.Expr
		if p.Tail == nil {
			cond = stmt.BinOp{Op: stmt.OpEq, Left: listLen(s), Right: stmt.LitInt{Value: n}}
		} else if n > 0 {
			cond = stmt.BinOp{Op: stmt.OpGte, Left: listLen(s), Right: stmt.LitInt{Value: n}}
		}
		et := listElemType(t)
		for i, el := range p.Elements {
			cond = andCond(cond, patternCond(listGet(s, i), et, el))
		}
		if p.Tail != nil {
			cond = andCond(cond, patternCond(listTail(s, len(p.Elements)), t, *p.Tail))
		}
		return cond
	}
	panic(fmt.Sprintf("lower: unsupported pattern %T in match lowering", pat))
}

// patternBindings returns the VarDecls that bind pat's variables from s.
// They are only valid once patternCond(s, pat) has held.
func patternBindings(s stmt.Expr, pat core.CorePattern) []stmt.Stmt {
	switch p := pat.(type) {
	case *core.VarPattern:
		if p.Name == "_" {
			return nil
		}
		return []stmt.Stmt{stmt.VarDecl{Name: p.Name, Value: s}}

	case *core.WildcardPattern, *core.LitPattern:
		return nil

	case *core.ConstructorPattern:
		var out []stmt.Stmt
		for j, arg := range p.Args {
			out = append(out, patternBindings(ctorField(s, j), arg)...)
		}
		return out

	case *core.TuplePattern:
		var out []stmt.Stmt
		for i, el := range p.Elements {
			out = append(out, patternBindings(tupleField(s, i), el)...)
		}
		return out

	case *core.RecordPattern:
		var out []stmt.Stmt
		for _, k := range sortedFieldNames(p) {
			out = append(out, patternBindings(recordField(s, k), p.Fields[k])...)
		}
		return out

	case *core.ListPattern:
		p = normalizeListPattern(p)
		var out []stmt.Stmt
		for i, el := range p.Elements {
			out = append(out, patternBindings(listGet(s, i), el)...)
		}
		if p.Tail != nil {
			out = append(out, patternBindings(listTail(s, len(p.Elements)), *p.Tail)...)
		}
		return out
	}
	panic(fmt.Sprintf("lower: unsupported pattern %T in match lowering", pat))
}

// normalizeListPattern flattens a chain of cons tails into one list
// pattern: `a :: b :: rest` elaborates to [a, ...[b, ...rest]] and becomes
// [a, b, ...rest]; `x :: y :: []` becomes the closed [x, y]. The two forms
// match exactly the same lists (a closed inner pattern pins the total
// length), and the flat form needs one length check and no nested
// _list_tail copies.
func normalizeListPattern(p *core.ListPattern) *core.ListPattern {
	if p.Tail == nil {
		return p
	}
	inner, ok := (*p.Tail).(*core.ListPattern)
	if !ok {
		return p
	}
	inner = normalizeListPattern(inner)
	elems := make([]core.CorePattern, 0, len(p.Elements)+len(inner.Elements))
	elems = append(elems, p.Elements...)
	elems = append(elems, inner.Elements...)
	return &core.ListPattern{Elements: elems, Tail: inner.Tail}
}

func andCond(a, b stmt.Expr) stmt.Expr {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	}
	return stmt.BinOp{Op: stmt.OpAnd, Left: a, Right: b}
}

func ctorField(s stmt.Expr, j int) stmt.Expr {
	return stmt.FieldAccess{Record: s, Field: fmt.Sprintf("_%d", j)}
}

func tupleField(s stmt.Expr, i int) stmt.Expr {
	return stmt.FieldAccess{Record: s, Field: fmt.Sprintf("_%d", i)}
}

func recordField(s stmt.Expr, k string) stmt.Expr {
	return stmt.FieldAccess{Record: s, Field: k}
}

func listLen(s stmt.Expr) stmt.Expr {
	return stmt.BuiltinCall{Name: "_len", Args: []stmt.Expr{s}}
}

func listGet(s stmt.Expr, i int) stmt.Expr {
	return stmt.BuiltinCall{Name: "_list_get", Args: []stmt.Expr{s, stmt.LitInt{Value: int64(i)}}}
}

func listTail(s stmt.Expr, n int) stmt.Expr {
	return stmt.BuiltinCall{Name: "_list_tail", Args: []stmt.Expr{s, stmt.LitInt{Value: int64(n)}}}
}

func sortedFieldNames(p *core.RecordPattern) []string {
	keys := make([]string, 0, len(p.Fields))
	for k := range p.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Type descent: best effort, nil when unknown. Only used to name the ADT in
// a tag check; an unknown type falls back to the compiler's deterministic
// tag→ADT inference.

func listElemType(t types.Type) types.Type {
	if l, ok := t.(*types.TList); ok {
		return l.Element
	}
	return nil
}

func tupleElemType(t types.Type, i int) types.Type {
	if tt, ok := t.(*types.TTuple); ok && i < len(tt.Elements) {
		return tt.Elements[i]
	}
	return nil
}

func recordFieldType(t types.Type, k string) types.Type {
	switch r := t.(type) {
	case *types.TRecord:
		return r.Fields[k]
	case *types.TRecordOpen:
		return r.Fields[k]
	}
	return nil
}
