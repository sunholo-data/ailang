package repl

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// The REPL echo spells a value exactly as `show` does (M-V1-SHOW-TOTAL M4):
// it used Value.String(), which left nested strings bare and spelled floats
// with %g, so the REPL and `ailang run` disagreed on the same value.
func TestFormatValueIsShow(t *testing.T) {
	cases := []struct {
		v    interface{}
		want string
	}{
		{&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "a"}, &eval.StringValue{Value: "b, c"}}}, `["a", "b, c"]`},
		{&eval.FloatValue{Value: 1e21}, "1000000000000000000000.0"},
		{float64(1e21), "1000000000000000000000.0"},
		{&eval.StringValue{Value: "bare"}, "bare"},
		{&eval.TaggedValue{CtorName: "Some", Fields: []eval.Value{&eval.StringValue{Value: "x"}}}, `Some("x")`},
	}
	for _, c := range cases {
		if got := formatValue(c.v); got != c.want {
			t.Errorf("formatValue(%v) = %q, want %q", c.v, got, c.want)
		}
		if ev, ok := c.v.(eval.Value); ok && formatValue(c.v) != eval.Show(ev) {
			t.Errorf("formatValue(%v) differs from eval.Show", c.v)
		}
	}
}
