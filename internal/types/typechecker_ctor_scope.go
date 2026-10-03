package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/importhint"
)

// TC_MATCH_001 codes a constructor pattern whose name no loaded module defines
// (M-CTOR-PATTERN-ALIAS-AND-SCOPE, #1478). Such an arm compiled clean and could
// never match: the runtime compares constructor tags by name.
const TC_MATCH_001 = "TC_MATCH_001"

// checkCtorPatternScope resolves a constructor-pattern name that is NOT in
// direct scope (tc.constructorTypes: local + imported constructors). The
// ladder (#1478):
//
//  1. direct scope            → nil (the caller's existing ADT checks apply)
//  2. transitively loaded     → nil, keeping #323's match-by-name semantics,
//     unless the scrutinee is concretely a different ADT, which is the same
//     foreign-constructor error a directly-imported constructor gets
//  3. known nowhere           → TC_MATCH_001
//
// The gate only runs when the pipeline supplied the transitive registry
// (SetDiagnosticConstructorTypes — the module pipeline). Paths without it
// (single-file, REPL) lack the knowledge to call a name unknown.
func (tc *CoreTypeChecker) checkCtorPatternScope(p *core.ConstructorPattern, scrutType Type) error {
	name := p.Name
	if _, ok := tc.patternADT(p); ok {
		return nil
	}
	if tc.diagnosticCtorTypes == nil {
		return nil
	}
	path := []string{fmt.Sprintf("constructor pattern %s", name)}
	if adt, ok := tc.diagnosticCtorTypes[name]; ok {
		// adt == "" marks a name defined by two different ADTs across loaded
		// modules; which one is meant is unknowable here, so accept.
		if scrutADT := extractADTName(scrutType); adt != "" && scrutADT != "" && scrutADT != adt {
			return NewMatchForeignConstructorError(
				name, adt, scrutADT,
				tc.lookupADTConstructors(adt), tc.lookupADTConstructors(scrutADT),
				path,
			)
		}
		return nil
	}
	return tc.newUnknownConstructorError(name, scrutType, path)
}

// patternADT returns the ADT a constructor pattern belongs to: the one the
// elaborator resolved in scope (exact even when two in-scope ADTs share the
// constructor name, e.g. an aliased import next to a local declaration), else
// the name-keyed registry.
func (tc *CoreTypeChecker) patternADT(p *core.ConstructorPattern) (string, bool) {
	if p.TypeName != "" {
		return p.TypeName, true
	}
	adt, ok := tc.constructorTypes[p.Name]
	return adt, ok
}

// newUnknownConstructorError builds the TC_MATCH_001 diagnostic. The
// suggestion names the scrutinee ADT's constructors when its type is known,
// otherwise the closest known constructor name.
func (tc *CoreTypeChecker) newUnknownConstructorError(name string, scrutType Type, path []string) *TypeCheckError {
	var suggestion string
	if scrutADT := extractADTName(scrutType); scrutADT != "" {
		if ctors := tc.lookupADTConstructors(scrutADT); len(ctors) > 0 {
			sort.Strings(ctors)
			if best := importhint.Closest(name, ctors); best != "" {
				suggestion = fmt.Sprintf("did you mean '%s'? %s's constructors are: %s", best, scrutADT, strings.Join(ctors, ", "))
			} else {
				suggestion = fmt.Sprintf("%s's constructors are: %s", scrutADT, strings.Join(ctors, ", "))
			}
		}
	}
	if suggestion == "" {
		if best := importhint.Closest(name, tc.knownConstructorNames()); best != "" {
			suggestion = fmt.Sprintf("did you mean '%s'?", best)
		}
	}
	hint := "a constructor must be declared in this module or imported, e.g. import std/option (Option, Some, None); an alias must be written in the import, e.g. import std/option (None as Nada)"
	if suggestion == "" {
		suggestion = hint
	} else {
		suggestion += "\n  " + hint
	}
	return &TypeCheckError{
		Kind: MatchUnknownConstructorError,
		Path: path,
		Message: fmt.Sprintf("%s: constructor pattern '%s' does not name any constructor in scope (no local declaration, import, or loaded module defines it), so this arm could never match",
			TC_MATCH_001, name),
		Suggestion: suggestion,
	}
}

// knownConstructorNames returns every constructor name the checker knows,
// direct scope first, deduplicated and sorted.
func (tc *CoreTypeChecker) knownConstructorNames() []string {
	seen := make(map[string]bool, len(tc.constructorTypes)+len(tc.diagnosticCtorTypes))
	for n := range tc.constructorTypes {
		seen[n] = true
	}
	for n := range tc.diagnosticCtorTypes {
		seen[n] = true
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
