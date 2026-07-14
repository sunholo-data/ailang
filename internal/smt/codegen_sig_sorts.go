package smt

import "strings"

// M-SMT-CALLEE-SORT-GATE defense-in-depth helpers: signature-sort extraction for
// validateDeclarations. Split out of codegen.go to keep it under the 800-line
// AI-maintainability CI gate (make check-file-sizes) — no logic changes.

// extractDefineFunSigSorts extracts the parameter sorts and return sort from an
// SMT-LIB (define-fun name ((v1 S1) (v2 S2)) RetSort body) declaration. Only the
// signature is parsed — never the body — so constructor/field references inside the
// body cannot produce false positives. Returns nil on any parse difficulty
// (fail-open: the caller-side AST gate is the primary defense).
func extractDefineFunSigSorts(decl string) []string {
	const prefix = "(define-fun "
	if !strings.HasPrefix(decl, prefix) {
		return nil
	}
	s := decl[len(prefix):]
	// Skip the function name (no spaces) up to the first space.
	sp := strings.IndexByte(s, ' ')
	if sp < 0 {
		return nil
	}
	s = strings.TrimLeft(s[sp+1:], " ")
	if len(s) == 0 || s[0] != '(' {
		return nil
	}
	// Read the balanced parameter list group.
	paramList, rest, ok := readBalancedParen(s)
	if !ok {
		return nil
	}
	var sorts []string
	// paramList is the inside of the outer parens: "(v1 S1) (v2 S2) ...".
	inner := paramList
	for {
		inner = strings.TrimLeft(inner, " ")
		if len(inner) == 0 || inner[0] != '(' {
			break
		}
		var pair string
		pair, inner, ok = readBalancedParen(inner)
		if !ok {
			return nil
		}
		// pair is "v S" (or "v (Seq X)"); the sort is everything after the first token.
		pair = strings.TrimSpace(pair)
		if psp := strings.IndexByte(pair, ' '); psp >= 0 {
			sorts = append(sorts, strings.TrimSpace(pair[psp+1:]))
		}
	}
	// Read the return sort: a parenthesized group or a bare token.
	rest = strings.TrimLeft(rest, " ")
	if len(rest) == 0 {
		return sorts
	}
	if rest[0] == '(' {
		if ret, _, okRet := readBalancedParen(rest); okRet {
			sorts = append(sorts, "("+ret+")")
		}
	} else {
		end := strings.IndexByte(rest, ' ')
		if end < 0 {
			end = len(rest)
		}
		sorts = append(sorts, rest[:end])
	}
	return sorts
}

// readBalancedParen reads one balanced parenthesized group from the start of s
// (which must begin with '('). It returns the group's inner content (without the
// outer parens) and the remainder after the closing paren.
func readBalancedParen(s string) (inner, rest string, ok bool) {
	if len(s) == 0 || s[0] != '(' {
		return "", "", false
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[1:i], s[i+1:], true
			}
		}
	}
	return "", "", false
}
