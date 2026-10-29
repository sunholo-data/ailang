package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

func hasNestedConstructorPattern(p core.CorePattern) bool {
	c, ok := p.(*core.ConstructorPattern)
	if !ok {
		return false
	}
	for _, arg := range c.Args {
		if _, ok := arg.(*core.ConstructorPattern); ok {
			return true
		}
	}
	return false
}

func encodeNestedMatch(m *core.Match, scrutinee string) (string, error) {
	c := activeADTContext
	if c == nil {
		return "", fmt.Errorf("%w: nested constructor match requires checked types", ErrUnresolvableTypes)
	}
	sort := ""
	if typ, ok := c.ti.GetForExpr(m.Scrutinee); ok {
		sort, _ = MapType(typ)
	}
	if sort == "" {
		if v, ok := m.Scrutinee.(*core.Var); ok && v.Name == "result" {
			sort = c.resultSort
		}
	}
	if sort == "" {
		return "", fmt.Errorf("%w: nested match scrutinee lacks a ground type", ErrUnresolvableTypes)
	}
	var rows [][]core.CorePattern
	for _, arm := range m.Arms {
		if arm.Guard == nil {
			rows = append(rows, []core.CorePattern{arm.Pattern})
		}
	}
	if !patternCoverage(rows, []string{sort}, 0) {
		return "", fmt.Errorf("%w: nested constructor match is not provably exhaustive", ErrUnresolvableTypes)
	}
	tests := make([]string, len(m.Arms))
	bodies := make([]string, len(m.Arms))
	for i, arm := range m.Arms {
		test, bindings, err := nestedPatternTest(arm.Pattern, scrutinee, sort)
		if err != nil {
			return "", err
		}
		body, err := encodePatternScoped(arm.Body, arm.Pattern)
		if err != nil {
			return "", err
		}
		if arm.Guard != nil {
			guard, err := encodePatternScoped(arm.Guard, arm.Pattern)
			if err != nil {
				return "", err
			}
			test = "(and " + test + " " + bindPatternVariables(bindings, guard) + ")"
		}
		tests[i] = test
		bodies[i] = bindPatternVariables(bindings, body)
	}
	// Coverage was checked independently. The final arm's test is therefore true
	// whenever prior ordered tests fail; retaining its body preserves first-match.
	out := bodies[len(bodies)-1]
	for i := len(bodies) - 2; i >= 0; i-- {
		out = fmt.Sprintf("(ite %s %s %s)", tests[i], bodies[i], out)
	}
	return out, nil
}

func bindPatternVariables(bindings []string, body string) string {
	if len(bindings) == 0 {
		return body
	}
	return "(let (" + strings.Join(bindings, " ") + ") " + body + ")"
}
func nestedPatternTest(p core.CorePattern, value, sort string) (string, []string, error) {
	switch x := p.(type) {
	case *core.WildcardPattern:
		return "true", nil, nil
	case *core.VarPattern:
		return "true", []string{"(" + x.Name + " " + value + ")"}, nil
	case *core.LitPattern:
		lit, err := litPatternValue(x)
		return "(= " + value + " " + lit + ")", nil, err
	case *core.ConstructorPattern:
		fields, ok := constructorFields(sort, x.Name)
		if !ok || len(fields) != len(x.Args) {
			return "", nil, fmt.Errorf("%w: invalid constructor pattern %s for %s", ErrUnresolvableTypes, x.Name, sort)
		}
		fieldSorts := make([]string, len(fields))
		for i, f := range fields {
			fieldSorts[i] = f.Sort
		}
		tests := []string{fmt.Sprintf("((_ is (%s (%s) %s)) %s)", x.Name, strings.Join(fieldSorts, " "), sort, value)}
		var bindings []string
		for i, arg := range x.Args {
			test, b, err := nestedPatternTest(arg, "("+fields[i].Name+" "+value+")", fields[i].Sort)
			if err != nil {
				return "", nil, err
			}
			tests = append(tests, test)
			bindings = append(bindings, b...)
		}
		return "(and " + strings.Join(tests, " ") + ")", bindings, nil
	}
	return "", nil, fmt.Errorf("%w: unsupported nested pattern %T", ErrUnresolvableTypes, p)
}
func constructorFields(sort, ctor string) ([]ADTField, bool) {
	c := activeADTContext
	if c == nil {
		return nil, false
	}
	name := sort
	var args []string
	if strings.HasPrefix(sort, "(") {
		parts := smtListParts(sort)
		if len(parts) == 0 {
			return nil, false
		}
		name = parts[0]
		args = parts[1:]
	}
	params := c.params[name]
	if len(params) != len(args) {
		return nil, false
	}
	sub := make(map[string]string)
	for i, p := range params {
		sub[p] = args[i]
	}
	for _, v := range c.adts[name] {
		if v.Name == ctor {
			fields := make([]ADTField, len(v.Fields))
			for i, f := range v.Fields {
				fields[i] = ADTField{Name: f.Name, Sort: substituteSort(f.Sort, sub)}
			}
			return fields, true
		}
	}
	return nil, false
}
func substituteSort(s string, sub map[string]string) string {
	if v, ok := sub[s]; ok {
		return v
	}
	if !strings.HasPrefix(s, "(") {
		return s
	}
	parts := smtListParts(s)
	for i, p := range parts {
		parts[i] = substituteSort(p, sub)
	}
	return "(" + strings.Join(parts, " ") + ")"
}
func irrefutablePattern(p core.CorePattern) bool {
	switch p.(type) {
	case *core.VarPattern, *core.WildcardPattern:
		return true
	}
	return false
}

// patternCoverage is a bounded pattern-matrix exhaustiveness check. It neither
// invents a catch-all nor treats a guarded constructor as covering its sort.
func patternCoverage(rows [][]core.CorePattern, sorts []string, depth int) bool {
	if len(rows) == 0 || depth > 64 || len(rows) > 256 {
		return false
	}
	if len(sorts) == 0 {
		return true
	}
	allWild := true
	for _, r := range rows {
		if len(r) == 0 {
			return false
		}
		allWild = allWild && irrefutablePattern(r[0])
	}
	if allWild {
		next := make([][]core.CorePattern, len(rows))
		for i, r := range rows {
			next[i] = r[1:]
		}
		return patternCoverage(next, sorts[1:], depth+1)
	}
	name := sorts[0]
	if strings.HasPrefix(name, "(") {
		parts := smtListParts(name)
		if len(parts) == 0 {
			return false
		}
		name = parts[0]
	}
	variants := activeADTContext.adts[name]
	if len(variants) == 0 {
		var next [][]core.CorePattern
		for _, r := range rows {
			if irrefutablePattern(r[0]) {
				next = append(next, r[1:])
			}
		}
		return patternCoverage(next, sorts[1:], depth+1)
	}
	for _, v := range variants {
		fields, ok := constructorFields(sorts[0], v.Name)
		if !ok {
			return false
		}
		nextSorts := make([]string, 0, len(fields)+len(sorts)-1)
		for _, f := range fields {
			nextSorts = append(nextSorts, f.Sort)
		}
		nextSorts = append(nextSorts, sorts[1:]...)
		var nextRows [][]core.CorePattern
		for _, r := range rows {
			var args []core.CorePattern
			if irrefutablePattern(r[0]) {
				for range fields {
					args = append(args, &core.WildcardPattern{})
				}
			} else if p, ok := r[0].(*core.ConstructorPattern); ok && p.Name == v.Name {
				if len(p.Args) != len(fields) {
					return false
				}
				args = append(args, p.Args...)
			} else {
				continue
			}
			nextRows = append(nextRows, append(args, r[1:]...))
		}
		if !patternCoverage(nextRows, nextSorts, depth+1) {
			return false
		}
	}
	return true
}
