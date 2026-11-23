package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
)

// optionEncodedBuiltins are the encoder names whose SMT form uses the Option
// datatype; collectSortsFromBody seeds Option for them.
var optionEncodedBuiltins = map[string]bool{
	listHeadOpt: true, listNthOpt: true, listLastOpt: true,
	optGetOrElse: true, optIsSome: true, optIsNone: true,
}

// encodeOptionAccess encodes the Option-shaped stdlib helpers with the runtime's
// semantics (std/list.ail head/nth/last, std/option.ail getOrElse/isSome/isNone):
//
//	head(xs)          = if len(xs) == 0 then None else Some(xs[0])
//	nth(xs, i)        = if i < 0 || i >= len(xs) then None else Some(xs[i])
//	last(xs)          = if len(xs) == 0 then None else Some(xs[len(xs)-1])
//	getOrElse(o, d)   = match o { Some(v) => v, None => d }
//	isSome(o)/isNone  = constructor tests
//
// head/nth/last build an Option, whose constructors need the concrete sort
// ((as None (Option Int))); it comes from the call's elaborated type and an
// unknown one refuses the call. ok is false when name is not one of them.
func encodeOptionAccess(name string, app *core.App) (string, bool, error) {
	arity := map[string]int{listHeadOpt: 1, listNthOpt: 2, listLastOpt: 1, optGetOrElse: 2, optIsSome: 1, optIsNone: 1}[name]
	if arity == 0 {
		return "", false, nil
	}
	if len(app.Args) != arity {
		return "", true, fmt.Errorf("%w: %s expects %d args, got %d", ErrUnsupportedConstruct, name, arity, len(app.Args))
	}
	enc := make([]string, arity)
	for i, a := range app.Args {
		e, err := EncodeExpr(a)
		if err != nil {
			return "", true, err
		}
		enc[i] = e
	}
	switch name {
	case optGetOrElse:
		return fmt.Sprintf("(match %s (((Some $opt_v) $opt_v) (None %s)))", enc[0], enc[1]), true, nil
	case optIsSome:
		return fmt.Sprintf("((_ is Some) %s)", enc[0]), true, nil
	case optIsNone:
		return fmt.Sprintf("((_ is None) %s)", enc[0]), true, nil
	}
	t := hofExprType(app)
	if t == nil {
		return "", true, fmt.Errorf("%w: the Option result type of %s is unknown", ErrUnsupportedConstruct, name)
	}
	sort, err := MapType(t)
	if err != nil {
		return "", true, fmt.Errorf("%w: the result type of %s has no SMT sort: %v", ErrUnsupportedConstruct, name, err)
	}
	none := fmt.Sprintf("(as None %s)", sort)
	some := func(v string) string { return fmt.Sprintf("((as Some %s) %s)", sort, v) }
	xs := enc[0]
	switch name {
	case listHeadOpt:
		return fmt.Sprintf("(ite (= (seq.len %[1]s) 0) %[2]s %[3]s)", xs, none, some("(seq.nth "+xs+" 0)")), true, nil
	case listNthOpt:
		i := enc[1]
		return fmt.Sprintf("(ite (or (< %[2]s 0) (>= %[2]s (seq.len %[1]s))) %[3]s %[4]s)", xs, i, none, some(fmt.Sprintf("(seq.nth %s %s)", xs, i))), true, nil
	default: // listLastOpt
		return fmt.Sprintf("(ite (= (seq.len %[1]s) 0) %[2]s %[3]s)", xs, none, some(fmt.Sprintf("(seq.nth %[1]s (- (seq.len %[1]s) 1))", xs))), true, nil
	}
}
