package smt

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/verification"
)

// Witness replay (M-VERIFY-SOUNDNESS-COVERAGE M6).
//
// A sat on a bounded encoding (recursion or a HOF specialisation) is reported
// inconclusive because its model may rest on the uninterpreted frontier. But the
// model's parameter values are concrete inputs: running the REAL function on
// them and checking its contracts at runtime decides the question for that
// input. If every requires holds and an ensures fails, the input is a genuine
// counterexample, whatever the frontier did. Anything else — a value that does
// not convert, a runtime error, contracts that hold — leaves it inconclusive.
// Replay can only promote, never demote, so it cannot introduce a false
// verdict as long as the runtime is the ground truth.

// WitnessArg is one parameter of a replayed call: its AILANG source name and
// the model value converted to an AILANG literal.
type WitnessArg struct {
	Name    string
	Literal ast.Expr
	Text    string // the literal, as AILANG source, for display
	Sort    string // the solver's sort, for display
}

// ReplayOutcome is what a WitnessReplayer found for one input.
type ReplayOutcome struct {
	// Violated: every requires held and an ensures was false at runtime.
	Violated bool
	// Detail describes the outcome (which clause failed, or why replay could
	// not decide).
	Detail string
}

// WitnessReplayer runs funcName on args with runtime contract checking. It is
// supplied by the caller (cmd/ailang), so the verifier does not depend on the
// evaluator.
type WitnessReplayer func(funcName string, args []WitnessArg) ReplayOutcome

// witnessArgs converts the model's values for params into AILANG literals. It
// fails (and replay is not attempted) when any parameter is missing from the
// model or has a value/sort this converter does not handle.
func witnessArgs(params []FunctionParam, model []ModelBinding, smtlib string) ([]WitnessArg, error) {
	byName := make(map[string]ModelBinding, len(model))
	for _, b := range model {
		byName[b.Name] = b
	}
	records := recordConstructors(smtlib)
	args := make([]WitnessArg, 0, len(params))
	for _, p := range params {
		b, ok := byName[ParamRenamePrefix+p.Name]
		if !ok || b.Term == nil {
			return nil, fmt.Errorf("no model value for %s", p.Name)
		}
		sorts, err := ParseSExprs(b.Sort)
		if err != nil || len(sorts) != 1 {
			return nil, fmt.Errorf("unreadable sort %q for %s", b.Sort, p.Name)
		}
		lit, err := modelTermToLiteral(expandModelLets(b.Term, nil), sorts[0], records)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Name, err)
		}
		args = append(args, WitnessArg{Name: p.Name, Literal: lit, Text: lit.String(), Sort: b.Sort})
	}
	return args, nil
}

// recordField is one field of a record datatype constructor.
type recordField struct {
	Name string
	Sort *SExpr
}

// recordConstructors maps each record constructor (mk_Record_…) declared in the
// script to its fields, in declaration order. Record accessors are the AILANG
// field names, so they name the literal's fields directly.
func recordConstructors(smtlib string) map[string][]recordField {
	out := make(map[string][]recordField)
	exprs, err := ParseSExprs(smtlib)
	if err != nil {
		return out
	}
	addCtors := func(ctors *SExpr) {
		if ctors == nil || !ctors.IsList {
			return
		}
		for _, c := range ctors.List {
			if !c.IsList && c.Str == nil && c.Atom != "" { // bare nullary constructor
				out[nullaryCtorKey(c.Atom)] = nil
				continue
			}
			name := c.Head()
			if len(c.List) == 1 { // (Ctor) — nullary enum constructor
				out[nullaryCtorKey(name)] = nil
				continue
			}
			if !strings.HasPrefix(name, "mk_") {
				continue
			}
			var fields []recordField
			for _, f := range c.List[1:] {
				if f.IsList && len(f.List) == 2 {
					fields = append(fields, recordField{Name: f.List[0].String(), Sort: f.List[1]})
				}
			}
			out[name] = fields
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

// modelTermToLiteral converts a Z3 model value of the given sort to an AILANG
// literal expression.
func modelTermToLiteral(term, sort *SExpr, records map[string][]recordField) (ast.Expr, error) {
	switch {
	case !sort.IsList && sort.Atom == "Int":
		n, err := modelInt(term)
		if err != nil {
			return nil, err
		}
		return &ast.Literal{Kind: ast.IntLit, Value: n}, nil
	case !sort.IsList && sort.Atom == "Real":
		f, err := modelReal(term)
		if err != nil {
			return nil, err
		}
		return &ast.Literal{Kind: ast.FloatLit, Value: f}, nil
	case !sort.IsList && sort.Atom == "Bool":
		if term.IsList || (term.Atom != "true" && term.Atom != "false") {
			return nil, fmt.Errorf("unsupported Bool value %s", term)
		}
		return &ast.Literal{Kind: ast.BoolLit, Value: term.Atom == "true"}, nil
	case !sort.IsList && sort.Atom == "String":
		if term.Str == nil {
			return nil, fmt.Errorf("unsupported String value %s", term)
		}
		s, err := unescapeZ3String(*term.Str)
		if err != nil {
			return nil, err
		}
		return &ast.Literal{Kind: ast.StringLit, Value: s}, nil
	case sort.Head() == "Seq" && len(sort.List) == 2:
		elems, err := modelSeq(term, sort.List[1], records)
		if err != nil {
			return nil, err
		}
		return &ast.List{Elements: elems}, nil
	case !sort.IsList && !term.IsList && term.Str == nil && hasKey(records, nullaryCtorKey(term.Atom)):
		// A nullary constructor of an enum ADT (`Failed`), declared in the script.
		return &ast.Identifier{Name: term.Atom}, nil
	case !sort.IsList:
		head := term.Head()
		if !term.IsList {
			head = term.Atom
		}
		fields, ok := records[head]
		if !ok {
			return nil, fmt.Errorf("unsupported value %s of sort %s", term, sort)
		}
		if len(term.List) != len(fields)+1 {
			return nil, fmt.Errorf("record %s: %d values for %d fields", head, len(term.List)-1, len(fields))
		}
		out := &ast.Record{}
		for i, f := range fields {
			v, err := modelTermToLiteral(term.List[i+1], f.Sort, records)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", f.Name, err)
			}
			out.Fields = append(out.Fields, &ast.Field{Name: f.Name, Value: v})
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported sort %s", sort)
}

func modelInt(term *SExpr) (int, error) {
	if term.IsList && term.Head() == "-" && len(term.List) == 2 {
		n, err := modelInt(term.List[1])
		return -n, err
	}
	if term.IsList || term.Str != nil {
		return 0, fmt.Errorf("unsupported Int value %s", term)
	}
	n, err := strconv.ParseInt(term.Atom, 10, 64) // outside int64: not a runtime value
	if err != nil {
		return 0, fmt.Errorf("Int value %s: %w", term.Atom, err)
	}
	return int(n), nil
}

func modelReal(term *SExpr) (float64, error) {
	if term.IsList {
		switch {
		case term.Head() == "-" && len(term.List) == 2:
			f, err := modelReal(term.List[1])
			return -f, err
		case term.Head() == "/" && len(term.List) == 3:
			a, err := modelReal(term.List[1])
			if err != nil {
				return 0, err
			}
			b, err := modelReal(term.List[2])
			if err != nil || b == 0 {
				return 0, fmt.Errorf("unsupported Real value %s", term)
			}
			return a / b, nil
		}
		return 0, fmt.Errorf("unsupported Real value %s", term)
	}
	r, ok := new(big.Rat).SetString(term.Atom)
	if !ok {
		return 0, fmt.Errorf("Real value %s", term.Atom)
	}
	f, _ := r.Float64()
	return f, nil
}

func modelSeq(term, elemSort *SExpr, records map[string][]recordField) ([]ast.Expr, error) {
	switch term.Head() {
	case "as":
		if len(term.List) == 3 && term.List[1].Atom == "seq.empty" {
			return nil, nil
		}
	case "seq.unit":
		if len(term.List) == 2 {
			v, err := modelTermToLiteral(term.List[1], elemSort, records)
			if err != nil {
				return nil, err
			}
			return []ast.Expr{v}, nil
		}
	case "seq.++":
		var out []ast.Expr
		for _, part := range term.List[1:] {
			elems, err := modelSeq(part, elemSort, records)
			if err != nil {
				return nil, err
			}
			out = append(out, elems...)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported sequence value %s", term)
}

// unescapeZ3String decodes SMT-LIB 2.6 string escapes (\u{XXXX}, \uXXXX); a
// doubled quote was already collapsed by the parser.
func unescapeZ3String(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) || s[i+1] != 'u' {
			b.WriteByte(s[i])
			continue
		}
		rest := s[i+2:]
		var hex string
		if strings.HasPrefix(rest, "{") {
			end := strings.IndexByte(rest, '}')
			if end < 0 {
				return "", fmt.Errorf("bad escape in %q", s)
			}
			hex = rest[1:end]
			i += 2 + end
		} else if len(rest) >= 4 {
			hex = rest[:4]
			i += 5
		} else {
			return "", fmt.Errorf("bad escape in %q", s)
		}
		r, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return "", fmt.Errorf("bad escape in %q", s)
		}
		b.WriteRune(rune(r))
	}
	return b.String(), nil
}

// replayInconclusive tries to decide an inconclusive result by running the
// function on the model's inputs. Only a runtime ensures failure with every
// requires holding changes the verdict (to a counterexample showing exactly
// the replayed inputs); otherwise vr is returned with a note on why replay
// did not decide it.
func replayInconclusive(vr VerifyResult, funcName string, params []FunctionParam, model []ModelBinding, smtlib string, replay WitnessReplayer) VerifyResult {
	args, err := witnessArgs(params, model, smtlib)
	if err != nil {
		vr.Reason += fmt.Sprintf("; witness replay not attempted (%v)", err)
		return vr
	}
	out := replay(funcName, args)
	if !out.Violated {
		if out.Detail != "" {
			vr.Reason += "; witness replay: " + out.Detail
		}
		return vr
	}
	vr.Status = "counterexample"
	vr.ReasonCode = verification.Counterexample
	vr.Reason = "witness replayed at runtime: " + out.Detail
	vr.WitnessReplayed = true
	vr.Model = vr.Model[:0]
	for _, a := range args {
		vr.Model = append(vr.Model, ModelBinding{Name: a.Name, Sort: a.Sort, Value: a.Text})
	}
	return vr
}

// uninterpretedInconclusive downgrades a model to inconclusive when the
// encoding left std/math functions uninterpreted: the solver may have chosen a
// value the real function never returns (exp(x) = 0, sqrt(2) = 5), so the
// model is not a counterexample by itself. Witness replay then runs the real
// function on the inputs and confirms or leaves it.
func uninterpretedInconclusive(vr VerifyResult, funcName string, symbols []string, res *SolverResult, verbose bool) VerifyResult {
	vr.Status = "inconclusive"
	vr.ReasonCode = verification.BoundedInconclusive
	vr.Reason = fmt.Sprintf("the solver's candidate model may rest on a value of %s that the real function never returns "+
		"(the verifier knows these only by sign and range facts), so it is neither a proof nor a confirmed violation",
		strings.Join(symbols, ", "))
	vr.Model = nil
	if verbose {
		vr.Model = res.Model
	}
	return vr
}

// expandModelLets substitutes the bindings of Z3's (let ((a!1 v) …) body)
// model terms, so the value is a plain constructor tree.
func expandModelLets(t *SExpr, env map[string]*SExpr) *SExpr {
	if t == nil {
		return nil
	}
	if !t.IsList {
		if t.Str == nil {
			if v, ok := env[t.Atom]; ok {
				return v
			}
		}
		return t
	}
	if t.Head() == "let" && len(t.List) == 3 && t.List[1].IsList {
		inner := make(map[string]*SExpr, len(env)+len(t.List[1].List))
		for k, v := range env {
			inner[k] = v
		}
		for _, b := range t.List[1].List { // parallel let: values see the outer env
			if b.IsList && len(b.List) == 2 && !b.List[0].IsList {
				inner[b.List[0].Atom] = expandModelLets(b.List[1], env)
			}
		}
		return expandModelLets(t.List[2], inner)
	}
	out := &SExpr{IsList: true, List: make([]*SExpr, len(t.List))}
	for i, c := range t.List {
		out.List[i] = expandModelLets(c, env)
	}
	out.Original = out.String()
	return out
}

// nullaryCtorKey keys enum constructors in the recordConstructors map apart
// from record constructors (a nil field list).
func nullaryCtorKey(name string) string { return "\x00enum:" + name }

func hasKey(m map[string][]recordField, k string) bool {
	_, ok := m[k]
	return ok
}
