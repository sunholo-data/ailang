package vm_test

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// End-to-end half of the M-V1-SHOW-TOTAL engine differential: real programs
// through the pipeline on the evaluator and on the strict bytecode VM. The
// value-level half (spec table, generated values, cost) is show_engines_test.go.

func e2eStr(s string) eval.Value { return &eval.StringValue{Value: s} }

// TestRunPrintUsesShowOnBothEngines: `ailang run` prints an entry's non-unit
// result. That display path used Value.String() (unquoted nested strings, %g
// floats) and, on the VM, a bridge back to evaluator values; both now print
// exactly what `show` returns (M-V1-SHOW-TOTAL M4).
func TestRunPrintUsesShowOnBothEngines(t *testing.T) {
	src := `module print_result
import std/option (Option, Some, None)
type Pt = Pt(string, float)
export func main() -> [(Option[string], Pt, {name: string, n: int})] {
  [(Some("a, b"), Pt("q\"", 1.0e21), {name: "", n: 1}), (None, Pt("", -0.0), {name: "x", n: 2})]
}
`
	const want = `[(Some("a, b"), Pt("q\"", 1000000000000000000000.0), {n: 1, name: ""}), (None, Pt("", -0.0), {n: 2, name: "x"})]`
	e, eerr, v, verr := runBothEngines(t, "print_result", src)
	if eerr != nil || verr != nil {
		t.Fatalf("eval err %v, vm err %v", eerr, verr)
	}
	if e != want || v != want {
		t.Fatalf("run --print:\n eval %s\n   vm %s\n want %s", e, v, want)
	}
}

// TestShowOutputParsesBackAsStringList closes the loop through the real
// toolchain: show of a [string] (valid UTF-8) is an AILANG list literal, so a
// program whose source IS that rendering re-shows to the same text on both
// engines. This is the round-trip the doc claims for nested strings.
func TestShowOutputParsesBackAsStringList(t *testing.T) {
	strs := []eval.Value{e2eStr(""), e2eStr(`"`), e2eStr(`\`), e2eStr(`\"`), e2eStr("a, b"), e2eStr("${x}"), e2eStr("$${"), e2eStr("\n\t\r"),
		e2eStr("\x00\x1b\x7f\u0085\u200b\ufeff"), e2eStr("é 中文 😀"), e2eStr("]"), e2eStr("Some(x)"), e2eStr(`\u{41}`)}
	rendered := eval.Show(&eval.ListValue{Elements: strs})
	src := "module show_roundtrip\n\nexport func main() -> string {\n  let xs: [string] = " + rendered + ";\n  show(xs)\n}\n"
	e, eerr, v, verr := runBothEngines(t, "show_roundtrip", src)
	if eerr != nil || verr != nil {
		t.Fatalf("eval err %v, vm err %v\nsource:\n%s", eerr, verr, src)
	}
	if e != rendered || v != rendered {
		t.Fatalf("round trip changed the value:\nshown  %s\neval   %s\nvm     %s", rendered, e, v)
	}
}
