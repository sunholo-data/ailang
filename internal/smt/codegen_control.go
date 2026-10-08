package smt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// encodeLet encodes a let binding as SMT-LIB let expression.
func encodeLet(let *core.Let) (string, error) {
	value, err := EncodeExpr(let.Value)
	if err != nil {
		return "", fmt.Errorf("let value: %w", err)
	}
	body, err := EncodeExpr(let.Body)
	if err != nil {
		return "", fmt.Errorf("let body: %w", err)
	}
	return fmt.Sprintf("(let ((%s %s)) %s)", let.Name, value, body), nil
}

// encodeMatch encodes a match expression.
// For enum ADTs: (match var ((Variant1 body1) (Variant2 body2)))
// For ADTs with fields: (match var (((Ctor field1 field2) body)))
//
// SMT-LIB match only accepts datatype constructors, and has no guards. A
// match with literal patterns (on Int/String/Bool) or with any guard goes
// through encodeMatchAsIte instead; emitting either into an SMT match is a Z3
// error for literals and silently drops the guard otherwise (unsound).
func encodeMatch(m *core.Match) (string, error) {
	scrutinee, err := EncodeExpr(m.Scrutinee)
	if err != nil {
		return "", fmt.Errorf("match scrutinee: %w", err)
	}
	if needsIteMatch(m) {
		return encodeMatchAsIte(scrutinee, m.Arms)
	}

	var arms []string
	for _, arm := range m.Arms {
		pattern, err := encodePattern(arm.Pattern)
		if err != nil {
			return "", fmt.Errorf("match pattern: %w", err)
		}
		body, err := EncodeExpr(arm.Body)
		if err != nil {
			return "", fmt.Errorf("match body: %w", err)
		}
		arms = append(arms, fmt.Sprintf("(%s %s)", pattern, body))
	}

	return fmt.Sprintf("(match %s (%s))", scrutinee, strings.Join(arms, " ")), nil
}

// needsIteMatch reports whether a match must be lowered to an ite chain:
// any guarded arm, or any top-level literal pattern.
func needsIteMatch(m *core.Match) bool {
	for _, arm := range m.Arms {
		if arm.Guard != nil {
			return true
		}
		if _, ok := arm.Pattern.(*core.LitPattern); ok {
			return true
		}
	}
	return false
}

// encodeMatchAsIte lowers a match to a first-match-wins ite chain. Each arm
// contributes a test (pattern test ∧ guard) and a body; variable patterns
// bind the scrutinee with a let around both. The chain must end in an
// unguarded irrefutable arm (wildcard or variable), or unguarded `true` and
// `false` arms between them — otherwise there is no value for the
// fall-through case and the function is skipped, not encoded with an
// invented default. (Exhaustiveness is only a warning in the elaborator, so
// constructor coverage cannot be assumed here.)
func encodeMatchAsIte(scrutinee string, arms []core.MatchArm) (string, error) {
	type iteArm struct{ test, body string }
	var chain []iteArm
	fallback := ""
	seenBool := map[bool]bool{}
	for _, arm := range arms {
		if lp, ok := arm.Pattern.(*core.LitPattern); ok && arm.Guard == nil {
			if b, isBool := lp.Value.(bool); isBool && seenBool[!b] {
				body, err := EncodeExpr(arm.Body)
				if err != nil {
					return "", fmt.Errorf("match body: %w", err)
				}
				fallback = body // the other boolean value is already handled
				break
			} else if isBool {
				seenBool[b] = true
			}
		}
		test, bindVar, err := encodeIteTest(scrutinee, arm.Pattern)
		if err != nil {
			return "", err
		}
		body, err := EncodeExpr(arm.Body)
		if err != nil {
			return "", fmt.Errorf("match body: %w", err)
		}
		if arm.Guard != nil {
			guard, err := EncodeExpr(arm.Guard)
			if err != nil {
				return "", fmt.Errorf("match guard: %w", err)
			}
			if bindVar != "" {
				guard = fmt.Sprintf("(let ((%s %s)) %s)", bindVar, scrutinee, guard)
			}
			if test == "true" {
				test = guard
			} else {
				test = fmt.Sprintf("(and %s %s)", test, guard)
			}
		}
		if bindVar != "" {
			body = fmt.Sprintf("(let ((%s %s)) %s)", bindVar, scrutinee, body)
		}
		if test == "true" {
			fallback = body
			break // later arms are unreachable
		}
		chain = append(chain, iteArm{test, body})
	}
	if fallback == "" {
		return "", fmt.Errorf("%w: literal or guarded match without an unguarded catch-all arm", ErrUnsupportedConstruct)
	}
	out := fallback
	for i := len(chain) - 1; i >= 0; i-- {
		out = fmt.Sprintf("(ite %s %s %s)", chain[i].test, chain[i].body, out)
	}
	return out, nil
}

// encodeIteTest returns the SMT test for one top-level pattern ("true" for
// irrefutable patterns) and, for a variable pattern, the name to bind.
func encodeIteTest(scrutinee string, pat core.CorePattern) (test, bindVar string, err error) {
	switch p := pat.(type) {
	case *core.WildcardPattern:
		return "true", "", nil
	case *core.VarPattern:
		return "true", p.Name, nil
	case *core.LitPattern:
		lit, err := litPatternValue(p)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("(= %s %s)", scrutinee, lit), "", nil
	case *core.ConstructorPattern:
		if len(p.Args) == 0 {
			return fmt.Sprintf("((_ is %s) %s)", p.Name, scrutinee), "", nil
		}
	}
	return "", "", fmt.Errorf("%w: guarded match arm with pattern %T", ErrUnsupportedConstruct, pat)
}

// litPatternValue encodes a literal pattern's value the way encodeLit encodes
// the same literal in an expression (quoted strings, negative ints).
func litPatternValue(p *core.LitPattern) (string, error) {
	switch v := p.Value.(type) {
	case int64:
		return encodeLit(&core.Lit{Kind: core.IntLit, Value: v})
	case int:
		return encodeLit(&core.Lit{Kind: core.IntLit, Value: int64(v)})
	case float64:
		return encodeLit(&core.Lit{Kind: core.FloatLit, Value: v})
	case bool:
		return encodeLit(&core.Lit{Kind: core.BoolLit, Value: v})
	case string:
		return encodeLit(&core.Lit{Kind: core.StringLit, Value: v})
	}
	return "", fmt.Errorf("%w: literal pattern of type %T", ErrUnsupportedConstruct, p.Value)
}

// encodePattern encodes a Core pattern for SMT-LIB match.
func encodePattern(pat core.CorePattern) (string, error) {
	if pat == nil {
		return "", fmt.Errorf("nil pattern")
	}
	switch p := pat.(type) {
	case *core.ConstructorPattern:
		if len(p.Args) == 0 {
			return p.Name, nil
		}
		var argParts []string
		for _, arg := range p.Args {
			encoded, err := encodePattern(arg)
			if err != nil {
				return "", err
			}
			argParts = append(argParts, encoded)
		}
		return fmt.Sprintf("(%s %s)", p.Name, strings.Join(argParts, " ")), nil
	case *core.VarPattern:
		return p.Name, nil
	case *core.WildcardPattern:
		// SMT-LIB uses _ as wildcard in some dialects; use a fresh variable
		return "_", nil
	case *core.LitPattern:
		// Top-level literal arms are encoded by encodeMatchAsIte; a literal
		// nested inside a constructor or record pattern has no SMT match form.
		return "", fmt.Errorf("%w: literal pattern nested inside a constructor or record pattern", ErrUnsupportedConstruct)
	case *core.RecordPattern:
		return encodeRecordPattern(p)
	default:
		// Wrap the sentinel so errors.Is survives the %w wrapping chain up
		// through the unroller — the verifier classifies this as a SKIP.
		// Known constructs get a user-facing name; #757's complaint was the raw
		// Go type name leaking into output.
		if _, ok := pat.(*core.ListPattern); ok {
			return "", fmt.Errorf("%w: match on list patterns (x :: rest) is outside the decodable fragment", ErrUnsupportedConstruct)
		}
		return "", fmt.Errorf("%w: unsupported pattern type %T in SMT encoding", ErrUnsupportedConstruct, pat)
	}
}

// encodeRecordPattern encodes a record pattern for SMT-LIB match.
// Record pattern {x, y} with Point type → (mk_Point x y)
//
// The encoding:
//  1. Build a canonical key from the pattern's field names (sorted, comma-joined)
//  2. Look up the record type info via activeFieldSetToSort
//  3. Emit (mk_<SortName> field1_pattern field2_pattern ...) in alphabetical field order
//
// Nested record patterns are handled recursively via encodePattern.
func encodeRecordPattern(rp *core.RecordPattern) (string, error) {
	// Look up the record type by matching field names
	info := lookupRecordByFields(rp.Fields)
	if info == nil {
		names := make([]string, 0, len(rp.Fields))
		for name := range rp.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		return "", fmt.Errorf("record pattern: unknown record type with fields %v", names)
	}

	// Encode each field's sub-pattern in alphabetical (sorted) order
	var args []string
	for _, fieldName := range info.FieldNames {
		fieldPat, ok := rp.Fields[fieldName]
		if !ok {
			// Field not present in pattern — use wildcard
			args = append(args, "_")
			continue
		}
		encoded, err := encodePattern(fieldPat)
		if err != nil {
			return "", fmt.Errorf("record pattern field %q: %w", fieldName, err)
		}
		args = append(args, encoded)
	}

	return fmt.Sprintf("(%s %s)", info.CtorName, strings.Join(args, " ")), nil
}
