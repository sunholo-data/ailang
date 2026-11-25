package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// std/list.reverse (M-VERIFY-SOUNDNESS-COVERAGE). Z3's sequence theory has no
// reverse, so each call is an uninterpreted function of its list sort, with
// the facts every reversal has:
//
//   - the same length;
//   - the first element of the result is the last of the input, and the last
//     is the first.
//
// As for the string builtins, a model that rests on its value is
// inconclusive (EncodeResult.Uninterpreted) and witness replay decides it.
// Before, every function calling reverse was skipped ("unsupported
// application"), although contracts such as a flush's
// `listLength(result.textLines) == 0` never depend on its value.

const listReverse = "_list_reverse"

// encodeListReverse encodes std/list.reverse(xs). ok is false for any other
// builtin.
func encodeListReverse(name string, app *core.App) (string, bool, error) {
	if name != listReverse {
		return "", false, nil
	}
	if len(app.Args) != 1 {
		return "", true, fmt.Errorf("%w: reverse expects 1 argument, got %d", ErrUnsupportedConstruct, len(app.Args))
	}
	sort := seqSortOf(app)
	if sort == "" {
		sort = seqSortOf(app.Args[0])
	}
	if sort == "" {
		return "", true, fmt.Errorf("%w: the list sort of a reverse call is unknown", ErrUnsupportedConstruct)
	}
	xs, err := encodeExprWithSortHint(app.Args[0], sort)
	if err != nil {
		return "", true, err
	}
	symbol := reverseSymbol(sort)
	if err := useUninterpreted(symbol, fmt.Sprintf("(declare-fun %s (%s) %s)", symbol, sort, sort)); err != nil {
		return "", true, err
	}
	r := fmt.Sprintf("(%s %s)", symbol, xs)
	for _, fact := range reverseFacts(r, xs) {
		addCallFact(fact)
	}
	return r, true, nil
}

// reverseSymbol is the uninterpreted function reversing lists of sort: one
// per sort, so equal arguments share a result. `(Seq String)` →
// `$list_reverse<Seq.String>` (no SMT sort name contains <, > or .).
func reverseSymbol(sort string) string {
	return "$list_reverse" + strings.NewReplacer("(", "<", ")", ">", " ", ".").Replace(sort)
}

// reverseFacts hold for r = reverse(xs), for every xs.
func reverseFacts(r, xs string) []string {
	return []string{
		fmt.Sprintf("(= (seq.len %s) (seq.len %s))", r, xs),
		fmt.Sprintf("(=> (> (seq.len %[2]s) 0) (and (= (seq.nth %[1]s 0) (seq.nth %[2]s (- (seq.len %[2]s) 1))) (= (seq.nth %[1]s (- (seq.len %[2]s) 1)) (seq.nth %[2]s 0))))", r, xs),
	}
}
