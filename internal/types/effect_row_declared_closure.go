package types

// M-EFFECT-PURE-ROW-OVERGENERALIZATION (#1091): a function whose declaration
// specifies a CLOSED effect row must not export an effect-polymorphic one.
//
// Inference leaves a binding's effect row as an unresolved row variable whenever
// nothing concrete forces it. The common way that happens is recursion: a recursive
// self-call deliberately shares — and does not bind — the enclosing function's
// effect-row variable (see the `outerVars` comment in inferApp), because binding it
// there would leak a partial resolution into the enclosing inference. For a function
// with a CONCRETE declared row (`! {IO}`) the recursive call unifies against that
// concrete set and the row closes on its own. For a `pure` function there is nothing
// to unify against, so the variable survives.
//
// generalizeWithConstraints then quantifies every free effect-row variable of the
// type (#386 Section B, correctly — that is what lets separate imported uses of a
// row-polymorphic function get independent rows). It has no way to know the
// declaration said `pure`, so it quantifies this leftover too, and the module exports
// `(string, string) -> int ! {...ρ2}`. Importers instantiate that row fresh, it
// unifies with whatever effect surrounds the call, and the effect checker blames the
// caller for an effect nothing performs — #1091's "Missing effects: FS".
//
// The fix is a DEFAULTING rule, applied before generalization: if the declaration
// carries no row variable, bind the leftover variable to the declared row. It cannot
// mask a genuine effect, because a genuine effect is a concrete LABEL, not a
// variable — labels are never removed here, and a declared-closed row that is missing
// a label the body requires is still rejected by the effect-checking pass.

import (
	"github.com/sunholo-data/ailang/internal/ast"
)

// declaredRowIsPolymorphic reports whether a declared effect annotation carries a
// row variable, i.e. whether the AUTHOR asked for effect polymorphism.
//
// The discriminator is deliberately the declaration and never the row variable's
// name. Compiler-generated rows are currently named `ρN` and source rows are written
// `e`, but keying on that spelling would silently break the first time somebody names
// a row variable something that looks generated. `ast.EffectAnnotation.IsRowVar` /
// isEffectRowVar is the same test ElaborateEffectRowWithBudgets already uses to build
// a row-variable tail, so the two agree by construction.
func declaredRowIsPolymorphic(annots []ast.EffectAnnotation) bool {
	for _, eff := range annots {
		if eff.IsRowVar || isEffectRowVar(eff.Name) {
			return true
		}
	}
	return false
}

// closeDeclaredEffectRow returns a substitution that binds the top-level effect-row
// variable of fnType to the closed row the declaration promises, or nil when nothing
// should change.
//
// It returns nil (leaving the type untouched) when:
//   - the declaration is effect-polymorphic (`! {e}`) — the row variable is the
//     author's, and quantifying it is exactly right (#386);
//   - the type is not a function type, or its effect row has no variable tail —
//     nothing is unresolved, so there is nothing to default.
//
// An EMPTY annotation list means `pure`/unannotated, which AILANG treats as the
// closed empty row: ElaborateEffectRowWithBudgets returns the purity sentinel for it,
// and the effect-checking pass compares against an empty declared row. So the absence
// of an annotation is a positive statement of purity here, not missing information.
//
// It also returns nil when the outer row's variable occurs ANYWHERE ELSE in the type —
// typically as the tail of a callback parameter's effect row. Such a variable is
// load-bearing even though the author never wrote `! {e}`: sharing one row between a
// callback and the result is how an inferred higher-order function stays
// effect-polymorphic. `std/list.flatMap` is declared with no effect annotation at all,
// yet its callback and result rows share an inferred variable, which is what lets
// callers pass an effectful lambda. Closing that variable would silently make every
// such combinator strictly pure and reject its existing callers — measured: doing so
// broke docparse/services/epub_parser's `flatMap(\entry. epubParseContentFile(...))`
// with "incompatible closed rows: r1 has extra labels [], r2 has extra labels [FS]".
//
// The #1091 shape is precisely the non-shared case: `(string, string) -> int ! {...ρ2}`
// has no function-typed parameter, so ρ2 occurs only in the outer row and closing it is
// safe. Restricting to that case fixes the defect without touching effect polymorphism,
// declared or inferred.
func closeDeclaredEffectRow(fnType Type, annots []ast.EffectAnnotation) Substitution {
	if declaredRowIsPolymorphic(annots) {
		return nil
	}

	fn, ok := fnType.(*TFunc2)
	if !ok {
		return nil
	}
	row := fn.EffectRow
	if row == nil || row.Tail == nil {
		return nil
	}

	// Bail out if the outer row variable is shared with any other row in the type.
	// freeEffectRowVarsInType walks the whole type, so compare it against a copy of
	// the function whose OWN effect row has been removed: anything still mentioning
	// the variable is a parameter or return-type occurrence.
	withoutOwnRow := &TFunc2{
		Params:    fn.Params,
		Return:    fn.Return,
		EffectRow: EmptyEffectRow(),
	}
	if freeEffectRowVarsInType(withoutOwnRow)[row.Tail.Name] {
		return nil
	}

	// Keep every concrete label the inferred row already carries. Dropping one would
	// hide a real effect; only the open TAIL is being closed.
	closed := &Row{
		Kind:       EffectRow,
		Labels:     row.Labels,
		Tail:       nil,
		Budgets:    row.Budgets,
		MinBudgets: row.MinBudgets,
		Params:     row.Params,
	}
	if closed.Labels == nil {
		closed.Labels = make(map[string]Type)
	}

	return Substitution{row.Tail.Name: closed}
}

// closeDeclaredEffectRowForBinding applies closeDeclaredEffectRow using the effect
// annotation recorded for the binding's lambda during elaboration, returning the
// possibly-rewritten type.
//
// Elaboration stores an entry in effectAnnotsFull only when a function declares at
// least one effect (see funcToLambda), so a missing entry is the `pure`/unannotated
// case — which is what we want to close.
func (tc *CoreTypeChecker) closeDeclaredEffectRowForBinding(nodeID uint64, typ Type) Type {
	sub := closeDeclaredEffectRow(typ, tc.effectAnnotsFull[nodeID])
	if len(sub) == 0 {
		return typ
	}
	return ApplySubstitution(sub, typ)
}
