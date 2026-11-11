package golang

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestEmittedShowMatchesEvaluator compiles the runtime `ailang compile` emits
// and checks its Show / ShowPreview against eval.Show / eval.ShowPreview on
// every shape the Go backend represents (M-V1-SHOW-TOTAL M3): the emitter used
// to carry its own copy of the elision constants and never quoted nested
// strings.
func TestEmittedShowMatchesEvaluator(t *testing.T) {
	str := func(s string) eval.Value { return &eval.StringValue{Value: s} }
	i := func(n int) eval.Value { return &eval.IntValue{Value: n} }
	f := func(x float64) eval.Value { return &eval.FloatValue{Value: x} }
	list := func(xs ...eval.Value) eval.Value { return &eval.ListValue{Elements: xs} }
	ctor := func(name string, xs ...eval.Value) eval.Value { return &eval.TaggedValue{CtorName: name, Fields: xs} }
	wide := make([]eval.Value, 50)
	wideGo := make([]string, 50)
	for k := range wide {
		wide[k] = i(k)
		wideGo[k] = fmt.Sprintf("int64(%d)", k)
	}
	deep, deepGo := eval.Value(i(1)), "int64(1)"
	for k := 0; k < 6; k++ {
		deep, deepGo = list(deep), "[]interface{}{"+deepGo+"}"
	}

	// Each case: Go expression in the emitted representation, and the
	// evaluator value it stands for.
	cases := []struct {
		goExpr string
		v      eval.Value
	}{
		{`int64(-7)`, i(-7)},
		{`3.14`, f(3.14)},
		{`2.0`, f(2)},
		{`math.Copysign(0, -1)`, f(math.Copysign(0, -1))},
		{`1e300`, f(1e300)},
		{`math.NaN()`, f(math.NaN())},
		{`math.Inf(-1)`, f(math.Inf(-1))},
		{`true`, &eval.BoolValue{Value: true}},
		{`nil`, &eval.UnitValue{}},
		{`"top \"level\"\n"`, str("top \"level\"\n")},
		{`[]interface{}{"a", "b, c", ""}`, list(str("a"), str("b, c"), str(""))},
		{`[]interface{}{"\"", "\\", "\n\t\r", "${x}", "$y", "\x00\x7f\u0085\u200b", "é 😀", "\xff"}`,
			list(str(`"`), str(`\`), str("\n\t\r"), str("${x}"), str("$y"), str("\x00\x7f\u0085\u200b"), str("é 😀"), str("\xff"))},
		{`Tuple{int64(1), "x", true}`, &eval.TupleValue{Elements: []eval.Value{i(1), str("x"), &eval.BoolValue{Value: true}}}},
		{`ArrayVal{"s", ""}`, eval.NewArray([]eval.Value{str("s"), str("")})},
		{`FloatArrayVal{1.5, math.Copysign(0, -1)}`, eval.NewArray([]eval.Value{f(1.5), f(math.Copysign(0, -1))})},
		{`map[string]interface{}{"b": "two", "a": int64(1)}`, &eval.RecordValue{Fields: map[string]eval.Value{"b": str("two"), "a": i(1)}}},
		{`map[string]interface{}{"_tag": "Some", "0": "x"}`, ctor("Some", str("x"))},
		{`map[string]interface{}{"_tag": "None"}`, ctor("None")},
		{`&Shape{Kind: "Named", Named: &ShapeNamed{F0: "n\"m", F1: &Shape{Kind: "Dot", Dot: &ShapeDot{}}}}`,
			ctor("Named", str("n\"m"), ctor("Dot"))},
		{`[]interface{}{` + strings.Join(wideGo, ", ") + `}`, list(wide...)},
		{deepGo, deep},
		// The NFC rule of strutil.QuoteTo (second review of #2229): NFD text, a
		// mark after an escape (must not become the invalid escape \ñ), a singleton.
		{`[]interface{}{"e\U00000301", "\n\U00000303", "\U0000212B", "q\U00000303", "\U00000301x"}`,
			list(str("e\U00000301"), str("\n\U00000303"), str("\U0000212B"), str("q\U00000303"), str("\U00000301x"))},
		{`[]byte("hi")`, &eval.BytesValue{Value: []byte("hi")}},
		{`func() {}`, &eval.FunctionValue{}},
	}

	var b strings.Builder
	b.WriteString("package showfixture\n\nimport (\n\t\"math\"\n\t\"testing\"\n)\n\n")
	b.WriteString("type Shape struct {\n\tKind  string\n\tDot   *ShapeDot\n\tNamed *ShapeNamed\n}\ntype ShapeDot struct{}\ntype ShapeNamed struct {\n\tF0 interface{}\n\tF1 interface{}\n}\n\n")
	b.WriteString("var _ = math.Pi\n\nfunc TestShow(t *testing.T) {\n\tcases := []struct {\n\t\tv    interface{}\n\t\twant string\n\t}{\n")
	for _, c := range cases {
		fmt.Fprintf(&b, "\t\t{%s, %q},\n", c.goExpr, eval.Show(c.v))
	}
	b.WriteString("\t}\n\tfor _, c := range cases {\n\t\tif got := Show(c.v); got != c.want {\n\t\t\tt.Errorf(\"Show(%#v)\\n got %q\\nwant %q\", c.v, got, c.want)\n\t\t}\n\t}\n}\n\n")
	b.WriteString("func TestShowPreview(t *testing.T) {\n\tcases := []struct {\n\t\tv    interface{}\n\t\tmax  int64\n\t\twant string\n\t}{\n")
	for _, c := range []struct {
		goExpr string
		v      eval.Value
		max    int
	}{
		{`[]interface{}{` + strings.Join(wideGo, ", ") + `}`, list(wide...), 10},
		{`[]interface{}{"éé", "éé"}`, list(str("éé"), str("éé")), 3},
		{`[]interface{}{int64(1), int64(2), int64(3)}`, list(i(1), i(2), i(3)), 9},
		{`[]interface{}{int64(1), int64(2), int64(3)}`, list(i(1), i(2), i(3)), 0},
		{`""`, str(""), -1},
		{`"abcdef"`, str("abcdef"), 4},
		{`"abc"`, str("abc"), 5},
	} {
		fmt.Fprintf(&b, "\t\t{%s, %d, %q},\n", c.goExpr, c.max, eval.ShowPreview(c.v, c.max))
	}
	b.WriteString("\t}\n\tfor _, c := range cases {\n\t\tif got := ShowPreview(c.v, c.max); got != c.want {\n\t\t\tt.Errorf(\"ShowPreview(%#v, %d)\\n got %q\\nwant %q\", c.v, c.max, got, c.want)\n\t\t}\n\t}\n}\n")

	gen := New("showfixture")
	runtimeCode, err := gen.GenerateRuntime()
	if err != nil {
		t.Fatalf("GenerateRuntime failed: %v", err)
	}
	tmp := t.TempDir()
	for name, contents := range map[string][]byte{
		"go.mod":          []byte("module showfixture\n\ngo 1.21\n"),
		"runtime.go":      runtimeCode,
		"runtime_test.go": []byte(b.String()),
	} {
		if err := os.WriteFile(filepath.Join(tmp, name), contents, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Dir = tmp
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("emitted Show disagrees with the evaluator: %v\n%s", err, output)
	}
}
