package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// Constants for the prescriptive M-TYPECHECK-NO-AUTO-UNWRAP-RESULT
// error message templates. Centralized so the three error-site
// builders (early gate in inferRecordAccess, deferred verifier
// post-substitution, and the unifier-side detector) all produce
// byte-identical messages — and so Sonar doesn't flag duplicated
// literals.
const (
	tufaTwoArmTemplate = "match expr {\n    %s(x) => x.%s,    -- access on the success payload\n    %s(e) => /* handle the error case */\n  }"
	tufaOneArmTemplate = "match expr {\n    %s(x) => x.%s\n  }"
	tufaNoArmTemplate  = "match expr { /* arms for each %s constructor */ }"
	tufaNoCtorsLabel   = "(no constructors registered)"
	tufaMessageFormat  = "cannot access field `.%s` on a value of type `%s` — `%s` is a tagged union (constructors: %s) and the record-access auto-unwrap behavior was a runtime footgun (silent on the success variant, crash on the error variant). Use `match` to destructure the variant first:\n  %s\n\nOr pass `--allow-unsafe-field-access` to downgrade this error to a warning during the v0.20.x migration window."
)

// isTaggedUnion reports whether the given type is a multi-constructor
// algebraic data type — i.e. a tagged union like Result[T, E], Option[T],
// or any user-defined ADT with more than one constructor.
//
// Returns false for:
//   - Type variables (TVar, TVar2 — polymorphic / unresolved)
//   - Primitives (int, float, string, bool, unit)
//   - Records (TRecord, TRecordOpen)
//   - Function types (TFunc, TFunc2)
//   - Lists, tuples, arrays
//   - Single-constructor ADTs (e.g. `type Wrap = Wrap({x:int})`)
//   - Unregistered type names
//
// This is the gate used by inferRecordAccess to reject
// `result.field` access on a value typed as `Result[T, E]` (which would
// otherwise silently auto-unwrap the variant payload at runtime — silent
// on Ok, crash on Err — see M-TYPECHECK-NO-AUTO-UNWRAP-RESULT design doc).
//
// Cycle-safety: only inspects the head type-constructor name (one level
// deep via extractADTName which is itself non-recursive on Type values).
// No need for a visited map.
//
// `constructorTypes` is the constructor → ADT-name registry (the same
// map used by lookupADTConstructors). Pass nil to short-circuit to false
// (treats every type as non-tagged-union — used in code paths that don't
// have a typechecker context).
func isTaggedUnion(t Type, constructorTypes map[string]string) bool {
	if t == nil || constructorTypes == nil {
		return false
	}
	adtName := extractADTName(t)
	if adtName == "" {
		// Not a named ADT (TVar, primitive, record, function, list, ...).
		return false
	}
	// Skip the built-in list constructor — `[T]` is parsed as
	// TApp("list", T) and extractADTName returns "list", but lists
	// aren't tagged unions for the purposes of field access.
	if adtName == "list" {
		return false
	}
	count := 0
	for _, adt := range constructorTypes {
		if adt == adtName {
			count++
			if count > 1 {
				return true
			}
		}
	}
	return false
}

// buildTufaTemplate constructs the prescriptive `match` template using
// the actual constructor names of the receiver's ADT. Used by all three
// error-site builders.
func buildTufaTemplate(ctors []string, field, adtName string) string {
	switch {
	case len(ctors) >= 2:
		return fmt.Sprintf(tufaTwoArmTemplate, ctors[0], field, ctors[1])
	case len(ctors) == 1:
		return fmt.Sprintf(tufaOneArmTemplate, ctors[0], field)
	default:
		return fmt.Sprintf(tufaNoArmTemplate, adtName)
	}
}

// collectAndSortCtors returns the constructor names of `adtName`, sorted
// alphabetically for stable error messages (map iteration is non-
// deterministic).
func collectAndSortCtors(adtName string, registry map[string]string) []string {
	var out []string
	for ctor, adt := range registry {
		if adt == adtName {
			out = append(out, ctor)
		}
	}
	sort.Strings(out)
	return out
}

// buildTufaMessage constructs the full prescriptive error message.
// Used by all three error-site builders so they emit byte-identical
// messages (only Position / Path differ at the TypeCheckError level).
func buildTufaMessage(field string, recvType Type, ctors []string, adtName string) string {
	ctorList := strings.Join(ctors, " | ")
	if ctorList == "" {
		ctorList = tufaNoCtorsLabel
	}
	template := buildTufaTemplate(ctors, field, adtName)
	return fmt.Sprintf(tufaMessageFormat, field, recvType.String(), adtName, ctorList, template)
}

// VerifyTaggedUnionFieldAccesses runs the deferred-field-access pass
// for M-TYPECHECK-NO-AUTO-UNWRAP-RESULT. Called by the pipeline after
// FinalizeSubstitutions has populated CoreTI with resolved (post-
// substitution) types. Walks every site recorded by inferRecordAccess
// during type inference; for each, looks up the receiver's resolved
// type and re-checks isTaggedUnion. Returns the first error found, or
// nil if all sites are clean.
//
// Why deferred: most receivers are still fresh type variables at
// constraint-emission time because of let-generalization +
// instantiation. The receiver type is only known concretely after the
// constraint solver runs — which is here. This is the gate that
// catches the motoko_ext_compaction_ai 0.1.3 bug shape.
//
// Returns nil when allowUnsafeFieldAccess is set (migration flag).
func (tc *CoreTypeChecker) VerifyTaggedUnionFieldAccesses() error {
	if tc.allowUnsafeFieldAccess || len(tc.deferredFieldAccesses) == 0 {
		return nil
	}
	for _, d := range tc.deferredFieldAccesses {
		recvType, ok := tc.CoreTI.Get(d.receiverID)
		if !ok || recvType == nil {
			continue
		}
		if isTaggedUnion(recvType, tc.constructorTypes) {
			return tc.makeDeferredRecordAccessError(d, recvType)
		}
	}
	return nil
}

// makeDeferredRecordAccessError builds the prescriptive error for the
// deferred-gate path. Same content as the unifier-side error and the
// inferRecordAccess-side error, just with the deferred site's metadata.
func (tc *CoreTypeChecker) makeDeferredRecordAccessError(d deferredFieldAccess, recvType Type) error {
	adtName := extractADTName(recvType)
	if adtName == "" {
		adtName = recvType.String()
	}
	ctors := collectAndSortCtors(adtName, tc.constructorTypes)
	return &TypeCheckError{
		Kind:     RecordAccessOnTaggedUnionError,
		Path:     []string{fmt.Sprintf("field access `.%s`", d.field)},
		Position: d.position,
		Message:  buildTufaMessage(d.field, recvType, ctors, adtName),
	}
}

// makeUnificationTaggedUnionError builds the prescriptive error fired
// from the unifier when a tagged-union ADT (Result, Option, user
// multi-variant) fails to unify with TRecordOpen — i.e. the user wrote
// `expr.field` where `expr` is a Result/Option value. Pre-v0.20.0 this
// surfaced as the generic "cannot unify type constructor X with
// *types.TRecordOpen" error.
//
// Lives outside CoreTypeChecker because the TCon-vs-TRecordOpen
// mismatch is detected at unification time, well after inferRecordAccess
// has emitted the constraint and added a fresh TVar for the receiver.
func makeUnificationTaggedUnionError(taggedUnion Type, openRec *TRecordOpen, ctors map[string]string) error {
	adtName := extractADTName(taggedUnion)
	if adtName == "" {
		adtName = taggedUnion.String()
	}

	// What field was the user trying to access? The TRecordOpen tells us
	// (the record-access constraint expects {field: T | r}).
	var triedField string
	for f := range openRec.Fields {
		triedField = f
		break // exactly one entry; map iteration over a singleton is fine
	}
	if triedField == "" {
		triedField = "_field_"
	}

	ctorNames := collectAndSortCtors(adtName, ctors)
	return fmt.Errorf("%s [record_access_on_tagged_union]",
		buildTufaMessage(triedField, taggedUnion, ctorNames, adtName))
}

// makeRecordAccessOnTaggedUnionError builds the prescriptive error for
// the early-gate path (concrete receiver type at inferRecordAccess
// time, e.g. direct constructor application like `Yes({...}).field`).
func (tc *CoreTypeChecker) makeRecordAccessOnTaggedUnionError(acc *core.RecordAccess, recvType Type) error {
	adtName := extractADTName(recvType)
	if adtName == "" {
		adtName = recvType.String()
	}
	ctors := collectAndSortCtors(adtName, tc.constructorTypes)
	return &TypeCheckError{
		Kind:     RecordAccessOnTaggedUnionError,
		Path:     []string{fmt.Sprintf("field access `.%s`", acc.Field)},
		Position: acc.Span().String(),
		Message:  buildTufaMessage(acc.Field, recvType, ctors, adtName),
	}
}
