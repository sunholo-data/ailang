package smt

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// Tuple sorts (M-VERIFY-SOUNDNESS-COVERAGE). A tuple of arity n is the
// built-in parametric datatype
//
//	(declare-datatype Tuple2 (par (T0 T1) ((mk_Tuple2 (Tuple2_0 T0) (Tuple2_1 T1)))))
//
// so `(int, string)` is the sort `(Tuple2 Int String)`. It goes through the
// same machinery as Option and Result: Verify adds the datatypes to the ADT
// table, the per-function filter keeps only the arities a function uses, a
// tuple expression is a constructor application, and a tuple pattern is the constructor pattern mk_TupleN (one
// variant, so it always matches) through the nested-pattern encoder.
//
// A module that declares its own type named TupleN keeps it; a function that
// needs that sort is then skipped (tupleSortShadowed), never encoded over the
// wrong datatype.

const (
	minTupleArity = 2
	maxTupleArity = 8
)

// tupleSortName is the datatype of n-tuples: Tuple2, Tuple3, ...
func tupleSortName(n int) string { return "Tuple" + strconv.Itoa(n) }

// tupleConstructor is its one constructor: mk_Tuple2 (model_format renders
// mk_Tuple* values as `(a, b)`).
func tupleConstructor(n int) string { return "mk_" + tupleSortName(n) }

// tupleAccessor is the accessor of component i (0-based): Tuple2_0.
func tupleAccessor(n, i int) string { return tupleSortName(n) + "_" + strconv.Itoa(i) }

// tupleArity is n for the sort name TupleN within the supported arities, else 0.
func tupleArity(name string) int {
	rest, ok := strings.CutPrefix(name, "Tuple")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < minTupleArity || n > maxTupleArity || strconv.Itoa(n) != rest {
		return 0
	}
	return n
}

// tupleSortOf builds the sort of a tuple from its component sorts.
func tupleSortOf(elems []string) (string, error) {
	n := len(elems)
	if n < minTupleArity || n > maxTupleArity {
		return "", fmt.Errorf("a %d-tuple has no SMT sort (supported: %d to %d components)", n, minTupleArity, maxTupleArity)
	}
	return "(" + tupleSortName(n) + " " + strings.Join(elems, " ") + ")", nil
}

// tupleComponentSorts splits (TupleN s0 … sn-1) into its component sorts; nil
// for any other sort.
func tupleComponentSorts(sort string) []string {
	parts := smtListParts(sort)
	if len(parts) < 1+minTupleArity || tupleArity(parts[0]) != len(parts)-1 {
		return nil
	}
	return parts[1:]
}

// TupleDatatypes returns the tuple datatypes of every supported arity and
// their type parameters, in the shape of ExtractADTTypesWithRecords.
func TupleDatatypes() (map[string][]ADTVariant, map[string][]string) {
	adts := make(map[string][]ADTVariant, maxTupleArity-minTupleArity+1)
	params := make(map[string][]string, maxTupleArity-minTupleArity+1)
	for n := minTupleArity; n <= maxTupleArity; n++ {
		binders := make([]string, n)
		fields := make([]ADTField, n)
		for i := range binders {
			binders[i] = "T" + strconv.Itoa(i)
			fields[i] = ADTField{Name: tupleAccessor(n, i), Sort: binders[i]}
		}
		adts[tupleSortName(n)] = []ADTVariant{{Name: tupleConstructor(n), Fields: fields}}
		params[tupleSortName(n)] = binders
	}
	return adts, params
}

// addTupleDatatypes adds the tuple datatypes to a module's ADT tables. A
// declared type of the same name wins, and is returned as shadowed.
func addTupleDatatypes(adts map[string][]ADTVariant, params map[string][]string) map[string]bool {
	shadowed := make(map[string]bool)
	tupleADTs, tupleParams := TupleDatatypes()
	for name, variants := range tupleADTs {
		if _, exists := adts[name]; exists {
			shadowed[name] = true
			continue
		}
		adts[name] = variants
		params[name] = tupleParams[name]
	}
	return shadowed
}

// neededSortConflict is why a function needing the sorts in needed cannot be
// encoded: a record name two record types share, or a tuple sort a declared
// type shadows. nil when there is neither.
func neededSortConflict(needed, conflictingRecords, tupleShadowed map[string]bool) error {
	if name := firstNeededConflict(needed, conflictingRecords); name != "" {
		return fmt.Errorf("two record types with the fields of %s have different field types, and the verifier names an anonymous record by its field names alone", name)
	}
	if name := firstNeededConflict(needed, tupleShadowed); name != "" {
		return fmt.Errorf("a declared type named %s shadows the verifier's datatype for %d-tuples", name, tupleArity(name))
	}
	return nil
}

// encodeTuple encodes a tuple expression as mk_TupleN of its components, each
// under its component sort from the tuple's checked type (so a `[]` component
// takes the declared element sort). Z3 infers the type parameters from the
// components, so the constructor needs no annotation.
func encodeTuple(t *core.Tuple) (string, error) {
	n := len(t.Elements)
	if n < minTupleArity || n > maxTupleArity {
		return "", fmt.Errorf("%w: a %d-tuple has no SMT sort", ErrUnsupportedConstruct, n)
	}
	components := tupleComponentSorts(checkedSort(t))
	if len(components) != n {
		components = make([]string, n)
	}
	args := make([]string, n)
	for i, el := range t.Elements {
		v, err := encodeExprWithSortHint(el, components[i])
		if err != nil {
			return "", fmt.Errorf("tuple component %d: %w", i, err)
		}
		args[i] = v
	}
	return "(" + tupleConstructor(n) + " " + strings.Join(args, " ") + ")", nil
}

// checkedSort is the SMT sort of expr's checked type, or "" without one.
func checkedSort(expr core.CoreExpr) string {
	c := activeADTContext
	if c == nil {
		return ""
	}
	typ, ok := c.ti.GetForExpr(expr)
	if !ok {
		return ""
	}
	sort, err := MapType(typ)
	if err != nil {
		return ""
	}
	return sort
}

// tuplePatternAsConstructor rewrites every tuple pattern in p into the
// constructor pattern of its datatype, which the nested-pattern encoder and
// the coverage check already handle. Other patterns are returned unchanged.
func tuplePatternAsConstructor(p core.CorePattern) core.CorePattern {
	switch x := p.(type) {
	case *core.TuplePattern:
		args := make([]core.CorePattern, len(x.Elements))
		for i, el := range x.Elements {
			args[i] = tuplePatternAsConstructor(el)
		}
		return &core.ConstructorPattern{Name: tupleConstructor(len(x.Elements)), Args: args}
	case *core.ConstructorPattern:
		if !hasTuplePattern(x) {
			return x
		}
		n := *x
		n.Args = make([]core.CorePattern, len(x.Args))
		for i, a := range x.Args {
			n.Args[i] = tuplePatternAsConstructor(a)
		}
		return &n
	case *core.ListPattern:
		if !hasTuplePattern(x) {
			return x
		}
		n := *x
		n.Elements = make([]core.CorePattern, len(x.Elements))
		for i, el := range x.Elements {
			n.Elements[i] = tuplePatternAsConstructor(el)
		}
		if x.Tail != nil {
			tail := tuplePatternAsConstructor(*x.Tail)
			n.Tail = &tail
		}
		return &n
	}
	return p
}

// hasTuplePattern reports whether p is, or contains, a tuple pattern.
func hasTuplePattern(p core.CorePattern) bool {
	switch x := p.(type) {
	case *core.TuplePattern:
		return true
	case *core.ConstructorPattern:
		for _, a := range x.Args {
			if hasTuplePattern(a) {
				return true
			}
		}
	case *core.ListPattern:
		for _, el := range x.Elements {
			if hasTuplePattern(el) {
				return true
			}
		}
		if x.Tail != nil {
			return hasTuplePattern(*x.Tail)
		}
	}
	return false
}

// matchWithoutTuplePatterns returns m with its tuple patterns rewritten by
// tuplePatternAsConstructor, or m itself when it has none.
func matchWithoutTuplePatterns(m *core.Match) *core.Match {
	found := false
	for _, arm := range m.Arms {
		found = found || hasTuplePattern(arm.Pattern)
	}
	if !found {
		return m
	}
	n := *m
	n.Arms = make([]core.MatchArm, len(m.Arms))
	for i, arm := range m.Arms {
		arm.Pattern = tuplePatternAsConstructor(arm.Pattern)
		n.Arms[i] = arm
	}
	return &n
}
