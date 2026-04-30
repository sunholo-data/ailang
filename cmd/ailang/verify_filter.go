package main

// verify_filter.go — per-function demand-driven filtering for SMT verification.
//
// When `ailang verify` runs against a multi-module file, the naive emission of
// ALL ADT types, record aliases, and inline-record declarations causes cascade
// Z3 errors: an unrelated alias's missing dependency makes the verifier reject
// every function in the module.
//
// The helpers in this file compute, per-function, the minimum set of sorts a
// function's contracts and body actually need — from its parameters, return
// type, and body expressions — and filter the type-declaration inputs to that
// set before encoding. Extracted from verify.go in the M-SMT-CROSS-MODULE-TYPES
// follow-up (v0.14.3) to keep verify.go under the 800-line organisation budget.

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/types"
)

// buildNeededSortSet returns the set of sort names reachable from the function's
// seeds (params, return, body) through ADT variant fields, record-alias field
// types, AND sorts referenced in the bodies of ExtraDeclarations (inline records
// from ADT constructor fields).
//
// Used to filter ExtraDeclarations: any inline record whose sort name is NOT in
// this set is dropped from the preamble. Also used to widen the needed set so
// that aliases referenced only via inline-record fields (e.g., TableCell
// referenced from Record_headers_rows) are retained.
func buildNeededSortSet(
	params []smt.FunctionParam,
	returnSort string,
	body core.CoreExpr,
	adtTypes map[string][]smt.ADTVariant,
	aliases map[string]*types.TRecord,
	extraDecls []string,
) map[string]bool {
	// Index extra decls by their sort name for O(1) lookup during the walk.
	declBySort := make(map[string]string, len(extraDecls))
	for _, decl := range extraDecls {
		if name := smt.ExtractSortNameFromDecl(decl); name != "" {
			declBySort[name] = decl
		}
	}

	seeds := collectSortSeeds(params, returnSort, body)
	needed := make(map[string]bool)
	queue := make([]string, 0, len(seeds))
	for s := range seeds {
		queue = append(queue, s)
	}
	for len(queue) > 0 {
		sort := queue[0]
		queue = queue[1:]
		if needed[sort] {
			continue
		}
		needed[sort] = true
		// Walk ADT variant fields.
		if variants, ok := adtTypes[sort]; ok {
			for _, v := range variants {
				for _, f := range v.Fields {
					dep := extractBaseSortName(f.Sort)
					if !needed[dep] && !isPrimitiveSMTSort(dep) {
						queue = append(queue, dep)
					}
				}
			}
		}
		// Walk alias field types.
		if alias, ok := aliases[sort]; ok {
			for _, fieldType := range alias.Fields {
				fieldSeeds := make(map[string]bool)
				collectSortsFromType(fieldType, fieldSeeds)
				for s := range fieldSeeds {
					if !needed[s] {
						queue = append(queue, s)
					}
				}
			}
		}
		// Walk sorts mentioned in the body of an extra declaration. Inline
		// records (e.g., Record_headers_rows) live only in ExtraDeclarations,
		// so without this step the aliases they reference (e.g., TableCell)
		// would not be retained.
		if decl, ok := declBySort[sort]; ok {
			for s := range extractReferencedSorts(decl, adtTypes, aliases) {
				if !needed[s] {
					queue = append(queue, s)
				}
			}
		}
	}
	return needed
}

// extractReferencedSorts scans an SMT-LIB declaration string for tokens that
// match a known ADT name or record-alias name. Used to widen the needed-sort
// set when an inline record (kept by the per-function filter) references types
// declared elsewhere.
func extractReferencedSorts(decl string, adtTypes map[string][]smt.ADTVariant, aliases map[string]*types.TRecord) map[string]bool {
	out := make(map[string]bool)
	for name := range adtTypes {
		if strings.Contains(decl, name) {
			out[name] = true
		}
	}
	for name := range aliases {
		if strings.Contains(decl, name) {
			out[name] = true
		}
	}
	return out
}

// filterRecordAliasesForFunction returns only the named record type aliases
// that a function actually references — directly via its params/return/body,
// or transitively via fields of an alias it does reference.
//
// Without this filter, every function in a module receives ALL record aliases
// from all imported modules, causing cascade Z3 errors when any single alias
// references an undeclared sort (e.g., ParsedDocument referencing Block, where
// Block was correctly filtered out for a function that uses only TableCell).
//
// The transitive closure walks alias field types to find further aliases that
// must be retained. ADT types referenced from alias fields are NOT pulled in
// here — those are handled separately by filterADTTypesForFunction. The seed
// set should already include any ADT types reachable from the function body.
func filterRecordAliasesForFunction(
	params []smt.FunctionParam,
	returnSort string,
	body core.CoreExpr,
	allAliases map[string]*types.TRecord,
	adtTypes map[string][]smt.ADTVariant,
) map[string]*types.TRecord {
	if len(allAliases) == 0 {
		return allAliases
	}

	seeds := collectSortSeeds(params, returnSort, body)
	if len(seeds) == 0 {
		return map[string]*types.TRecord{}
	}

	// Transitive closure: walk alias field types. A seed sort that is itself
	// an alias pulls in any further aliases its fields reference.
	needed := make(map[string]bool)
	queue := make([]string, 0, len(seeds))
	for s := range seeds {
		queue = append(queue, s)
	}
	for len(queue) > 0 {
		sort := queue[0]
		queue = queue[1:]
		if needed[sort] {
			continue
		}
		needed[sort] = true
		// If this sort is a known alias, walk its fields to find more aliases.
		if alias, ok := allAliases[sort]; ok {
			for _, fieldType := range alias.Fields {
				fieldSeeds := make(map[string]bool)
				collectSortsFromType(fieldType, fieldSeeds)
				for s := range fieldSeeds {
					if !needed[s] {
						queue = append(queue, s)
					}
				}
			}
		}
		// If this sort is an ADT, walk its variants' fields too — an ADT field
		// may be a record alias we still need (e.g., Block.SomeVariant has a
		// {field: SomeAlias} record-typed field).
		if variants, ok := adtTypes[sort]; ok {
			for _, v := range variants {
				for _, f := range v.Fields {
					dep := extractBaseSortName(f.Sort)
					if !needed[dep] && !isPrimitiveSMTSort(dep) {
						queue = append(queue, dep)
					}
				}
			}
		}
	}

	result := make(map[string]*types.TRecord, len(needed))
	for name, rec := range allAliases {
		if needed[name] {
			result[name] = rec
		}
	}
	return result
}

// filterExtraDeclarationsForFunction returns only those ExtraDeclarations
// (inline record types from ADT constructor fields, e.g., Record_blocks_kind)
// that the function actually needs. A declaration is kept if its sort name is
// in the seed-derived `needed` set.
//
// Without this filter, inline-record declarations leak into every function's
// preamble. When such a record references an ADT sort that was correctly
// filtered out (e.g., Record_blocks_kind references Block), Z3 raises
// "sort 'Block' is not declared" and the function is marked unverified.
func filterExtraDeclarationsForFunction(allDecls []string, needed map[string]bool) []string {
	if len(allDecls) == 0 {
		return allDecls
	}
	out := make([]string, 0, len(allDecls))
	for _, decl := range allDecls {
		sortName := smt.ExtractSortNameFromDecl(decl)
		if sortName == "" {
			// Unknown declaration shape — keep it to be safe.
			out = append(out, decl)
			continue
		}
		if needed[sortName] {
			out = append(out, decl)
		}
	}
	return out
}

// filterADTTypesForFunction returns only the ADT types that a function actually
// references through its parameter types, return type, and body expressions.
// This prevents cross-module type pollution where unrelated ADTs (e.g., Json)
// cause cascade Z3 errors for functions that only use primitive types.
func filterADTTypesForFunction(
	params []smt.FunctionParam,
	returnSort string,
	body core.CoreExpr,
	allADTTypes map[string][]smt.ADTVariant,
) map[string][]smt.ADTVariant {
	if len(allADTTypes) == 0 {
		return allADTTypes
	}

	// Collect sort names referenced by this function
	seeds := collectSortSeeds(params, returnSort, body)

	// If no non-primitive sorts referenced, return empty map
	if len(seeds) == 0 {
		return map[string][]smt.ADTVariant{}
	}

	// Compute transitive closure: ADT variants may reference other ADT types
	needed := make(map[string]bool)
	queue := make([]string, 0, len(seeds))
	for s := range seeds {
		queue = append(queue, s)
	}
	for len(queue) > 0 {
		sort := queue[0]
		queue = queue[1:]
		if needed[sort] {
			continue
		}
		needed[sort] = true
		// Check if this sort is an ADT; if so, collect sorts from its variant fields
		if variants, ok := allADTTypes[sort]; ok {
			for _, v := range variants {
				for _, f := range v.Fields {
					dep := extractBaseSortName(f.Sort)
					if !needed[dep] && !isPrimitiveSMTSort(dep) {
						queue = append(queue, dep)
					}
				}
			}
		}
	}

	// Filter ADT types to only those needed
	result := make(map[string][]smt.ADTVariant, len(needed))
	for name, variants := range allADTTypes {
		if needed[name] {
			result[name] = variants
		}
	}
	return result
}

// collectSortSeeds gathers non-primitive sort names from function params, return type, and body.
func collectSortSeeds(params []smt.FunctionParam, returnSort string, body core.CoreExpr) map[string]bool {
	seeds := make(map[string]bool)

	// From parameter types
	for _, p := range params {
		collectSortsFromType(p.Type, seeds)
	}

	// From return sort
	base := extractBaseSortName(returnSort)
	if !isPrimitiveSMTSort(base) {
		seeds[base] = true
	}

	// From body: walk for constructor patterns and ADT constructor applications
	if body != nil {
		collectSortsFromBody(body, seeds)
	}

	return seeds
}

// collectSortsFromType extracts non-primitive sort names from a types.Type.
func collectSortsFromType(t types.Type, sorts map[string]bool) {
	if t == nil {
		return
	}
	switch ty := t.(type) {
	case *types.TCon:
		if !isPrimitiveTypeName(ty.Name) {
			sorts[ty.Name] = true
		}
	case *types.TApp:
		collectSortsFromType(ty.Constructor, sorts)
		for _, arg := range ty.Args {
			collectSortsFromType(arg, sorts)
		}
	case *types.TFunc2:
		for _, p := range ty.Params {
			collectSortsFromType(p, sorts)
		}
		collectSortsFromType(ty.Return, sorts)
	case *types.TList:
		collectSortsFromType(ty.Element, sorts)
	case *types.TTuple:
		for _, elem := range ty.Elements {
			collectSortsFromType(elem, sorts)
		}
	case *types.TRecord:
		for _, fieldType := range ty.Fields {
			collectSortsFromType(fieldType, sorts)
		}
	}
}

// collectSortsFromBody walks a core expression to find ADT constructor references.
func collectSortsFromBody(expr core.CoreExpr, sorts map[string]bool) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.Match:
		collectSortsFromBody(e.Scrutinee, sorts)
		for _, arm := range e.Arms {
			if cp, ok := arm.Pattern.(*core.ConstructorPattern); ok {
				// Constructor name might be the ADT type or variant name;
				// the ADT type is what we need. We add the constructor name
				// and let the transitive closure resolve it.
				sorts[cp.Name] = true
			}
			collectSortsFromBody(arm.Body, sorts)
		}
	case *core.App:
		collectSortsFromBody(e.Func, sorts)
		for _, arg := range e.Args {
			collectSortsFromBody(arg, sorts)
		}
	case *core.Let:
		collectSortsFromBody(e.Value, sorts)
		collectSortsFromBody(e.Body, sorts)
	case *core.LetRec:
		for _, b := range e.Bindings {
			collectSortsFromBody(b.Value, sorts)
		}
		collectSortsFromBody(e.Body, sorts)
	case *core.Lambda:
		collectSortsFromBody(e.Body, sorts)
	case *core.If:
		collectSortsFromBody(e.Cond, sorts)
		collectSortsFromBody(e.Then, sorts)
		collectSortsFromBody(e.Else, sorts)
	case *core.BinOp:
		collectSortsFromBody(e.Left, sorts)
		collectSortsFromBody(e.Right, sorts)
	case *core.UnOp:
		collectSortsFromBody(e.Operand, sorts)
	case *core.Record:
		for _, v := range e.Fields {
			collectSortsFromBody(v, sorts)
		}
	case *core.RecordAccess:
		collectSortsFromBody(e.Record, sorts)
	case *core.RecordUpdate:
		collectSortsFromBody(e.Base, sorts)
		for _, v := range e.Updates {
			collectSortsFromBody(v, sorts)
		}
	case *core.List:
		for _, elem := range e.Elements {
			collectSortsFromBody(elem, sorts)
		}
	case *core.Tuple:
		for _, elem := range e.Elements {
			collectSortsFromBody(elem, sorts)
		}
	case *core.Intrinsic:
		for _, arg := range e.Args {
			collectSortsFromBody(arg, sorts)
		}
	}
}

// extractBaseSortName extracts the base sort name from an SMT sort string.
// e.g., "(Seq Int)" → "Int", "Int" → "Int", "(Seq Block)" → "Block"
func extractBaseSortName(sort string) string {
	sort = strings.TrimSpace(sort)
	if strings.HasPrefix(sort, "(") {
		// Parametric sort like (Seq Block) — extract inner sorts
		inner := strings.TrimPrefix(sort, "(")
		inner = strings.TrimSuffix(inner, ")")
		parts := strings.Fields(inner)
		// Return the last non-keyword part (the element type)
		for i := len(parts) - 1; i >= 0; i-- {
			if !isPrimitiveSMTSort(parts[i]) && parts[i] != "Seq" && parts[i] != "Array" {
				return parts[i]
			}
		}
		if len(parts) > 1 {
			return parts[len(parts)-1]
		}
		return sort
	}
	return sort
}

// isPrimitiveSMTSort returns true for Z3 built-in sorts.
func isPrimitiveSMTSort(sort string) bool {
	switch sort {
	case "Int", "Bool", "String", "Real", "", "()", "Seq", "Array", "ALL":
		return true
	}
	return false
}

// isPrimitiveTypeName returns true for AILANG primitive type names.
func isPrimitiveTypeName(name string) bool {
	switch name {
	case "int", "bool", "string", "float", "Int", "Bool", "String", "Real", "unit", "()":
		return true
	}
	return false
}
