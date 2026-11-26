package smt

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// formatModelBindings converts raw Z3 model bindings into human/AI-readable
// AILANG syntax for counterexample and caveat display (M-AI-PREFERRED-LANGUAGE M1):
//   - Drops internal HOF/uninterpreted Array stub tables (`(Array ...)`).
//   - Strips the solver's `$p_` parameter rename prefix (`$p_xs` → `xs`).
//   - Decodes concrete Int, Real, Bool, String, Seq (`[e1, e2]`), Record
//     (`{ a: 1, b: "x" }`), Tuple (`(a, b)`), and ADT (`Some(42)`, `None`)
//     values into valid AILANG literals.
//   - Leaves symbolic terms (e.g. expressions referencing an uninterpreted
//     frontier stub) untouched rather than corrupting them.
func formatModelBindings(model []ModelBinding, smtlib string) []ModelBinding {
	if len(model) == 0 {
		return model
	}
	dtors := datatypeConstructors(smtlib)
	out := make([]ModelBinding, 0, len(model))
	for _, b := range model {
		if strings.HasPrefix(b.Sort, "(Array ") {
			continue // internal uninterpreted function stub table
		}
		fb := b
		fb.Name = strings.TrimPrefix(b.Name, ParamRenamePrefix)
		formatted := false
		if b.Term != nil {
			if sorts, err := ParseSExprs(b.Sort); err == nil && len(sorts) == 1 {
				expanded := expandModelLets(b.Term, nil)
				if rendered, ok := formatModelTerm(expanded, sorts[0], dtors); ok {
					fb.Value = rendered
					formatted = true
				}
			}
		}
		// Drop un-evaluated symbolic `result` definitions (e.g. Z3's raw `(let ...)`
		// body referencing `$p_` or `_map_spec_` stubs) when concrete inputs exist.
		if !formatted && fb.Name == "result" && (strings.Contains(b.Value, ParamRenamePrefix) || strings.Contains(b.Value, "_spec_")) {
			continue
		}
		out = append(out, fb)
	}
	return out
}

// ctorInfo describes a declared SMT datatype constructor for display formatting.
type ctorInfo struct {
	Fields   []recordField
	IsRecord bool
	IsTuple  bool
	Nullary  bool
}

// datatypeConstructors parses all `declare-datatype` and `declare-datatypes`
// declarations in smtlib so formatModelTerm can render records, tuples, and ADTs.
func datatypeConstructors(smtlib string) map[string]ctorInfo {
	out := make(map[string]ctorInfo)
	exprs, err := ParseSExprs(smtlib)
	if err != nil {
		return out
	}
	addCtors := func(ctors *SExpr) {
		if ctors == nil || !ctors.IsList {
			return
		}
		for _, c := range ctors.List {
			if !c.IsList && c.Str == nil && c.Atom != "" {
				out[c.Atom] = ctorInfo{Nullary: true}
				continue
			}
			name := c.Head()
			if name == "" {
				continue
			}
			if len(c.List) == 1 {
				out[name] = ctorInfo{Nullary: true}
				continue
			}
			var fields []recordField
			for _, f := range c.List[1:] {
				if f.IsList && len(f.List) == 2 {
					fields = append(fields, recordField{Name: f.List[0].String(), Sort: f.List[1]})
				}
			}
			isRec := strings.HasPrefix(name, "mk_Record") || strings.HasPrefix(name, "mk_rec_")
			isTup := strings.HasPrefix(name, "mk_Tuple") || strings.HasPrefix(name, "Tuple")
			out[name] = ctorInfo{
				Fields:   fields,
				IsRecord: isRec,
				IsTuple:  isTup,
			}
		}
	}
	for _, e := range exprs {
		switch e.Head() {
		case "declare-datatype":
			if len(e.List) == 3 {
				addCtors(e.List[2])
			}
		case "declare-datatypes":
			if len(e.List) == 3 && e.List[2].IsList {
				for _, ctors := range e.List[2].List {
					addCtors(ctors)
				}
			}
		}
	}
	return out
}

// formatModelTerm renders a single Z3 model S-expression of the given sort as
// canonical AILANG syntax, returning ("", false) when the term is symbolic or
// of an unsupported sort.
func formatModelTerm(term, sort *SExpr, dtors map[string]ctorInfo) (string, bool) {
	if term == nil || sort == nil {
		return "", false
	}
	switch {
	case !sort.IsList && sort.Atom == "Int":
		return formatModelInt(term)
	case !sort.IsList && sort.Atom == "Real":
		f, err := modelReal(term)
		if err != nil {
			return "", false
		}
		s := strconv.FormatFloat(f, 'g', -1, 64)
		if !strings.ContainsAny(s, ".eE") {
			s += ".0"
		}
		return s, true
	case !sort.IsList && sort.Atom == "Bool":
		if !term.IsList && (term.Atom == "true" || term.Atom == "false") {
			return term.Atom, true
		}
		return "", false
	case !sort.IsList && sort.Atom == "String":
		if term.Str == nil {
			return "", false
		}
		s, err := unescapeZ3String(*term.Str)
		if err != nil {
			return "", false
		}
		return strconv.Quote(s), true
	case sort.Head() == "Seq" && len(sort.List) == 2:
		elems, ok := formatModelSeq(term, sort.List[1], dtors)
		if !ok {
			return "", false
		}
		return "[" + strings.Join(elems, ", ") + "]", true
	case isTupleValue(term, sort):
		vals := make([]string, len(sort.List)-1)
		for i := range vals {
			v, ok := formatModelTerm(term.List[i+1], sort.List[i+1], dtors)
			if !ok {
				return "", false
			}
			vals[i] = v
		}
		return "(" + strings.Join(vals, ", ") + ")", true
	case !sort.IsList:
		return formatModelDatatype(term, dtors)
	}
	return "", false
}

// isTupleValue reports whether term is a value (mk_TupleN c0 … cn-1) of the
// tuple sort (TupleN s0 … sn-1). The component sorts come from the sort, as
// the datatype's own fields are its type parameters.
func isTupleValue(term, sort *SExpr) bool {
	n := tupleArity(sort.Head())
	return n > 0 && len(sort.List) == n+1 && term.IsList && term.Head() == tupleConstructor(n) && len(term.List) == n+1
}

// formatModelInt renders arbitrary-precision SMT Int literals (including values
// outside int64 such as overflow counterexamples).
func formatModelInt(term *SExpr) (string, bool) {
	if term.IsList && term.Head() == "-" && len(term.List) == 2 {
		inner, ok := formatModelInt(term.List[1])
		if !ok {
			return "", false
		}
		if strings.HasPrefix(inner, "-") {
			return strings.TrimPrefix(inner, "-"), true
		}
		return "-" + inner, true
	}
	if term.IsList || term.Str != nil {
		return "", false
	}
	n, ok := new(big.Int).SetString(term.Atom, 10)
	if !ok {
		return "", false
	}
	return n.String(), true
}

func formatModelSeq(term, elemSort *SExpr, dtors map[string]ctorInfo) ([]string, bool) {
	switch term.Head() {
	case "as":
		if len(term.List) == 3 && term.List[1].Atom == "seq.empty" {
			return nil, true
		}
	case "seq.unit":
		if len(term.List) == 2 {
			v, ok := formatModelTerm(term.List[1], elemSort, dtors)
			if !ok {
				return nil, false
			}
			return []string{v}, true
		}
	case "seq.++":
		var out []string
		for _, part := range term.List[1:] {
			elems, ok := formatModelSeq(part, elemSort, dtors)
			if !ok {
				return nil, false
			}
			out = append(out, elems...)
		}
		return out, true
	}
	return nil, false
}

func formatModelDatatype(term *SExpr, dtors map[string]ctorInfo) (string, bool) {
	if !term.IsList {
		if term.Str != nil || term.Atom == "" {
			return "", false
		}
		if info, ok := dtors[term.Atom]; ok && info.Nullary {
			return term.Atom, true
		}
		return "", false
	}
	head := term.Head()
	info, ok := dtors[head]
	if !ok {
		return "", false
	}
	if info.Nullary && len(term.List) == 1 {
		return head, true
	}
	if len(term.List) != len(info.Fields)+1 {
		return "", false
	}
	vals := make([]string, len(info.Fields))
	for i, f := range info.Fields {
		v, ok := formatModelTerm(term.List[i+1], f.Sort, dtors)
		if !ok {
			return "", false
		}
		vals[i] = v
	}
	switch {
	case info.IsRecord:
		parts := make([]string, len(info.Fields))
		for i, f := range info.Fields {
			parts[i] = fmt.Sprintf("%s: %s", f.Name, vals[i])
		}
		return "{ " + strings.Join(parts, ", ") + " }", true
	case info.IsTuple:
		return "(" + strings.Join(vals, ", ") + ")", true
	default:
		return fmt.Sprintf("%s(%s)", head, strings.Join(vals, ", ")), true
	}
}
