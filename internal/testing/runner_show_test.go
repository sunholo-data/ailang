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
}

// The inline-test failure line ("test N: expected X, got Y") pins the show
// spelling end to end: nested strings quoted, record fields in name order.
// Review mutation survivor: reverting runner.go to %v passed the suite.
func TestInlineTestFailureMessageUsesShow(t *testing.T) {
	src := `module fail_msg

func tag(s: string) -> {name: string, tags: [string]}
  tests [
    ("a b", {name: "a b", tags: ["x", "y, z"]})
  ]
  { {name: s, tags: ["x"]} }
`
	result := runInlineTestsOnSource(t, src)
	if result.FailedTests != 1 {
		t.Fatalf("want 1 failing test, got %d failed / %d passed", result.FailedTests, result.PassedTests)
	}
	got := firstFailureError(result)
	want := `test 0: expected {name: "a b", tags: ["x", "y, z"]}, got {name: "a b", tags: ["x"]}`
	if got != want {
		t.Fatalf("failure message:\n got %s\nwant %s", got, want)
	}
}

func TestEnsuresInputsUseShow(t *testing.T) {
	strs := &eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "a"}, &eval.StringValue{Value: "b, c"}}}
	params := []*ast.Param{{Name: "xs"}, {Name: "n"}}
	if got, want := formatEnsuresInputs(params, []eval.Value{strs, &eval.IntValue{Value: 3}}), `xs=["a", "b, c"], n=3`; got != want {
		t.Errorf("formatEnsuresInputs = %s, want %s", got, want)
	}
}
