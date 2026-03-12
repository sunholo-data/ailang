package smt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/core"
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
func encodeMatch(m *core.Match) (string, error) {
	scrutinee, err := EncodeExpr(m.Scrutinee)
	if err != nil {
		return "", fmt.Errorf("match scrutinee: %w", err)
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
		return fmt.Sprintf("%v", p.Value), nil
	case *core.RecordPattern:
		return encodeRecordPattern(p)
	default:
		return "", fmt.Errorf("unsupported pattern type %T in SMT encoding", pat)
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
