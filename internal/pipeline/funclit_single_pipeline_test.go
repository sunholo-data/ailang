package pipeline

import (
	"strings"
	"testing"
)

// #2238 through the single-file pipeline (runSingle: a Source with no file
// name), which plumbs the elaborator's annotations to the checker separately
// from the module pipeline. A literal's annotation is enforced there too, and
// a callback parameter with a concrete effect contract passed to an
// effect-polymorphic callee checks (it needs SetLiteralLambdas in this path).
func TestFuncLitAnnotations_SinglePipeline(t *testing.T) {
	for _, c := range []struct {
		name, src, want string
	}{
		{"annotation enforced",
			"module m\nexport func main() -> () ! {IO} {\n  let f = func(x: int) -> int { 0 };\n  println(show(f(\"str\")))\n}\n",
			"int vs string"},
		{"concrete-contract callback passed to an effect-polymorphic callee",
			// One top-level function: the single-file path rejects any program
			// with two (CoreTypeInfo validation), pre-existing and unrelated.
			"module m\n" +
				"export func live() -> int ! {IO, Trace} {\n" +
				"  let poly = func(n: int, cb: int -> () ! {e}) -> int ! {e} { cb(n); n };\n" +
				"  let step = func(n: int, cb: int -> () ! {IO}) -> int ! {IO, Trace} { poly(n, cb) };\n" +
				"  step(1, \\x. ())\n}\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Run(Config{Mode: ModeCheck}, Source{Code: c.src})
			if c.want == "" {
				if err != nil {
					t.Fatalf("must check: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error containing %q, got %v", c.want, err)
			}
		})
	}
}
