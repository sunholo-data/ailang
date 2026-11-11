package testing

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Test-failure and counterexample messages spell values as `show` does
// (M-V1-SHOW-TOTAL M4). They used %v (Value.String()): nested strings bare,
// and a []eval.Value counterexample in Go slice syntax "[a b]".
func TestFailureMessagesSpellValuesWithShow(t *testing.T) {
	strs := &eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "a"}, &eval.StringValue{Value: "b, c"}}}
	if got, want := showValues([]eval.Value{strs, &eval.FloatValue{Value: 1e21}}), `[["a", "b, c"], 1000000000000000000000.0]`; got != want {
		t.Errorf("showValues = %s, want %s", got, want)
	}
	params := []*ast.Param{{Name: "xs"}, {Name: "n"}}
	if got, want := formatEnsuresInputs(params, []eval.Value{strs, &eval.IntValue{Value: 3}}), `xs=["a", "b, c"], n=3`; got != want {
		t.Errorf("formatEnsuresInputs = %s, want %s", got, want)
	}
}
