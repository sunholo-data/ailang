package smt

import (
	"fmt"
	"sort"
	"strings"
)

// smtListParts splits one parenthesized declaration/sort list at its outer
// whitespace boundaries. Type declarations contain symbols and lists, not strings.
func smtListParts(s string) []string {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return nil
	}
	s = s[1 : len(s)-1]
	var parts []string
	depth, start := 0, -1
	for i, c := range s {
		if depth == 0 && (c == ' ' || c == '\n' || c == '\r' || c == '\t') {
			if start >= 0 {
				parts = append(parts, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
		if c == '(' {
			depth++
		}
		if c == ')' {
			depth--
			if depth < 0 {
				return nil
			}
		}
	}
	if depth != 0 {
		return nil
	}
	if start >= 0 {
		parts = append(parts, s[start:])
	}
	return parts
}

func datatypeBody(body string) (params []string, variants string) {
	parts := smtListParts(body)
	if len(parts) == 3 && parts[0] == "par" {
		return smtListParts(parts[1]), parts[2]
	}
	return nil, body
}

func sortSymbols(s string) []string {
	return strings.Fields(strings.NewReplacer("(", " ", ")", " ").Replace(s))
}

// datatypeSortReferences examines field SORTS, not constructor/accessor names.
// Formal binders are local and must never be treated as undeclared concrete sorts.
func datatypeSortReferences(body string) map[string]bool {
	params, variants := datatypeBody(body)
	formal := make(map[string]bool)
	for _, p := range params {
		formal[p] = true
	}
	refs := make(map[string]bool)
	for _, variant := range smtListParts(variants) {
		fields := smtListParts(variant)
		if len(fields) == 0 {
			refs["<invalid-variant>"] = true
			continue
		}
		for _, field := range fields[1:] {
			pair := smtListParts(field)
			if len(pair) != 2 {
				refs["<invalid-field>"] = true
				continue
			}
			for _, symbol := range sortSymbols(pair[1]) {
				if !formal[symbol] && !primitiveSorts[symbol] {
					refs[symbol] = true
				}
			}
		}
	}
	return refs
}

func datatypeUndeclaredSort(body string, own map[string]bool, declared map[string]bool) string {
	refs := datatypeSortReferences(body)
	names := make([]string, 0, len(refs))
	for name := range refs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, ref := range names {
		if !own[ref] && !declared[ref] {
			return ref
		}
	}
	return ""
}

func validateSortApplication(s string, declared map[string]bool) string {
	for _, symbol := range sortSymbols(s) {
		if !primitiveSorts[symbol] && !declared[symbol] {
			return symbol
		}
	}
	return ""
}

func validateParametricDefinitions(adts map[string][]ADTVariant, binders map[string][]string) error {
	names := make([]string, 0, len(binders))
	for name := range binders {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		params := binders[name]
		if _, ok := adts[name]; !ok {
			continue
		}
		seen := make(map[string]bool)
		for _, p := range params {
			if p == "" || seen[p] {
				return fmt.Errorf("%w: invalid/duplicate formal sort in %s", ErrUnresolvableTypes, name)
			}
			seen[p] = true
		}
		for _, v := range adts[name] {
			for _, f := range v.Fields {
				if !wellFormedParametricSort(f.Sort, binders, seen) {
					return fmt.Errorf("%w: unsupported field type in %s.%s", ErrUnresolvableTypes, name, f.Name)
				}
			}
		}
	}
	return nil
}

func wellFormedParametricSort(s string, arities map[string][]string, formal map[string]bool) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if s[0] != '(' {
		if s == "Seq" || s == "Array" {
			return false
		}
		return formal[s] || len(arities[s]) == 0
	}
	parts := smtListParts(s)
	if len(parts) < 2 {
		return false
	}
	want := len(arities[parts[0]])
	if parts[0] == "Seq" {
		want = 1
	}
	if want == 0 || len(parts) != want+1 {
		return false
	}
	for _, arg := range parts[1:] {
		if !wellFormedParametricSort(arg, arities, formal) {
			return false
		}
	}
	return true
}
