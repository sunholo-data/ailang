package smt

import (
	"fmt"
	"strings"
)

// validateDeclarations checks that all sort references in the assembled declarations
// resolve to either primitive sorts or declared sorts. Returns ErrUnresolvableTypes
// if any declaration references an undeclared non-primitive sort.
func validateDeclarations(decls []string, ctx *SMTContext) error {
	// Walk declarations IN ORDER, tracking which sorts have been declared so far.
	// This catches forward references that Z3 would reject.
	//
	// We seed with ctx.DeclaredTypes BUT exclude sorts that are being declared
	// in this batch — those must be checked positionally.
	declaredInBatch := make(map[string]bool)
	for _, decl := range decls {
		for _, name := range pluralDatatypeSortNames(decl) {
			declaredInBatch[name] = true
		}
		if name := ExtractSortNameFromDecl(decl); name != "" {
			declaredInBatch[name] = true
		}
	}
	declared := make(map[string]bool)
	for k, v := range ctx.DeclaredTypes {
		if v && !declaredInBatch[k] {
			declared[k] = true
		}
	}

	for _, decl := range decls {
		// Defense-in-depth (M-SMT-CALLEE-SORT-GATE): validate that cross-function
		// callee define-funs reference only declared/primitive sorts in their
		// signature. The caller-side AST gate is the primary defense; this catches
		// any leak that slips past it (e.g. a callee with no AST FuncDecl) so we
		// return ErrUnresolvableTypes (→ graceful skip) instead of crashing Z3.
		if strings.HasPrefix(decl, "(define-fun ") {
			for _, sortStr := range extractDefineFunSigSorts(decl) {
				for _, ref := range sortRefPattern.FindAllString(sortStr, -1) {
					if primitiveSorts[ref] || declared[ref] || strings.HasPrefix(ref, "mk_") {
						continue
					}
					return fmt.Errorf("%w: sort %q referenced in signature of a callee define-fun is not declared", ErrUnresolvableTypes, ref)
				}
			}
			continue
		}
		// A helper's uninterpreted function (a contract or an opaque helper):
		// a sort only its signature names was a Z3 ERROR, not a skip.
		if strings.HasPrefix(decl, "(declare-fun ") && !strings.Contains(decl, "\n") {
			es, err := ParseSExprs(decl)
			if err == nil && len(es) == 1 && len(es[0].List) == 4 {
				sig := es[0].List[2].String() + " " + es[0].List[3].String()
				for _, ref := range sortRefPattern.FindAllString(sig, -1) {
					if primitiveSorts[ref] || declared[ref] || strings.HasPrefix(ref, "mk_") {
						continue
					}
					return fmt.Errorf("%w: sort %q in the signature of %s is not declared", ErrUnresolvableTypes, ref, es[0].List[1].String())
				}
			}
			continue
		}
		if strings.HasPrefix(decl, "(declare-datatypes ") {
			sortNames := pluralDatatypeSortNames(decl)
			groupSorts := make(map[string]bool, len(sortNames))
			for _, name := range sortNames {
				groupSorts[name] = true
			}
			parts := smtListParts(decl)
			if len(parts) != 3 {
				return fmt.Errorf("%w: malformed datatype group", ErrUnresolvableTypes)
			}
			for _, body := range smtListParts(parts[2]) {
				if ref := datatypeUndeclaredSort(body, groupSorts, declared); ref != "" {
					return fmt.Errorf("%w: sort %q referenced in mutual datatype declaration is not declared", ErrUnresolvableTypes, ref)
				}
			}
			for _, name := range sortNames {
				declared[name] = true
			}
			continue
		}
		if strings.HasPrefix(decl, "(declare-const ") || strings.HasPrefix(decl, "(define-const ") {
			name, sortStr, ok := constantDeclarationHeader(decl)
			if !ok {
				return fmt.Errorf("%w: malformed constant declaration %q", ErrUnresolvableTypes, decl)
			}
			if ref := firstUnresolvedSort(sortStr, declared); ref != "" {
				return fmt.Errorf("%w: constant %q references undeclared sort %q", ErrUnresolvableTypes, name, ref)
			}
			continue
		}
		if !strings.HasPrefix(decl, "(declare-datatype ") {
			// Other declaration shapes have no typed constant header.
			continue
		}
		sortName, body := splitDeclareDatatype(decl)
		if ref := datatypeUndeclaredSort(body, map[string]bool{sortName: true}, declared); ref != "" {
			return fmt.Errorf("%w: sort %q referenced in declaration of %q is not declared", ErrUnresolvableTypes, ref, sortName)
		}
		// Mark this sort as declared for subsequent declarations
		if sortName != "" {
			declared[sortName] = true
		}
	}
	return nil
}
