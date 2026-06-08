package types

import (
	"fmt"
	"sort"
)

// BuildConstructorFactoryScheme builds the polymorphic type scheme for an ADT
// constructor factory ($adt.make_<Type>_<Ctor>), i.e. the function
// `field0 -> field1 -> ... -> Type[t0, t1, ...]` (or just `Type[...]` for a
// nullary constructor).
//
// It maps every occurrence of a declared type-parameter name inside each field
// type to the ADT's canonical result-type variables (t0, t1, ...), so a compound
// field such as [a], (a, a) or Option[a] stays tied to the result type Type[t0].
//
// M-TYPE-LIST-SOUND round 3: the previous logic (duplicated in
// pipeline.buildConstructorFactoryTypes and repl.registerAdtFactory) only
// remapped a field when the WHOLE field was a bare type var. Compound fields
// embedding a type param passed through verbatim carrying the *declared* param
// name (e.g. `a`), while the result type used renamed vars (`t0`) and only `t0`
// was quantified. On Instantiate the field var stayed stale and disconnected
// from the result var, so the concrete scrutinee arg (e.g. int in Bag[int])
// never reached the field and a destructured `[a]` unified freely with
// `[string]` — a runtime-crashing soundness hole. Substituting the param->t_i
// mapping across the ENTIRE field type closes this for every field shape, and is
// a no-op for the already-correct bare-var and concrete-field cases.
//
// Cycle-safety: field types come from the elaborator's astTypeToInternalType,
// which produces finite trees; Substitute and collectFreeVars are both
// cycle-safe regardless.
func BuildConstructorFactoryScheme(
	typeName string,
	arity int,
	typeParamCount int,
	typeParamNames []string,
	fieldTypes []Type,
) *Scheme {
	var typeVars []string
	seen := make(map[string]bool)
	addVar := func(name string) {
		if !seen[name] {
			seen[name] = true
			typeVars = append(typeVars, name)
		}
	}

	// Canonical result-type vars t0, t1, ... — these are the variables that
	// appear in the result type and define the ADT's polymorphism.
	adtTypeVars := make([]Type, typeParamCount)
	for i := 0; i < typeParamCount; i++ {
		varName := fmt.Sprintf("t%d", i)
		adtTypeVars[i] = &TVar2{Name: varName, Kind: Star}
		addVar(varName)
	}

	// Map declared param names (e.g. "a") -> canonical result vars (t0, t1, ...).
	typeParamToVar := make(map[string]Type)
	for i, name := range typeParamNames {
		if i < len(adtTypeVars) {
			typeParamToVar[name] = adtTypeVars[i]
		}
	}

	paramTypes := make([]Type, 0, arity)
	for i := 0; i < arity; i++ {
		var fieldType Type
		switch {
		case i < len(fieldTypes) && fieldTypes[i] != nil:
			// Remap EVERY declared-param var inside the field type. Handles bare
			// `a`, compound `[a]`/`(a,a)`/`Option[a]`, nested combinations, and
			// concrete types (no vars to substitute -> unchanged).
			fieldType = fieldTypes[i].Substitute(typeParamToVar)
			// Defensive: quantify any residual free var that is not a declared
			// param so a stray var can never leak in as an orphaned free var
			// (preserves the prior "unknown type var -> fresh quantified var"
			// behaviour). Sorted for deterministic scheme TypeVars ordering.
			residual := make(map[string]bool)
			collectFreeVars(fieldType, residual)
			extra := make([]string, 0, len(residual))
			for v := range residual {
				if !seen[v] {
					extra = append(extra, v)
				}
			}
			sort.Strings(extra)
			for _, v := range extra {
				addVar(v)
			}
		case i < typeParamCount:
			// No field type available: fall back to the positional ADT var.
			fieldType = adtTypeVars[i]
		default:
			// Extra field with no type info: fresh quantified var.
			varName := fmt.Sprintf("a%d", i)
			addVar(varName)
			fieldType = &TVar2{Name: varName, Kind: Star}
		}
		paramTypes = append(paramTypes, fieldType)
	}

	// Result type: TApp(TypeName, [t0, t1, ...]) for parameterized ADTs,
	// plain TCon for non-parameterized ones.
	var resultType Type
	if typeParamCount > 0 {
		resultType = &TApp{Constructor: &TCon{Name: typeName}, Args: adtTypeVars}
	} else {
		resultType = &TCon{Name: typeName}
	}

	var factoryType Type
	if arity == 0 {
		// Nullary constructor: just the result type.
		factoryType = resultType
	} else {
		factoryType = &TFunc2{
			Params:    paramTypes,
			EffectRow: nil, // Constructors are pure.
			Return:    resultType,
		}
	}

	return &Scheme{TypeVars: typeVars, Type: factoryType}
}
