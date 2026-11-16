package repl

import (
	"strings"
	"testing"
)

// #2238 through the REPL/WASM module registry, which plumbs the elaborator's
// annotations to the checker separately from the CLI pipeline: a literal's
// annotation is enforced, and a callback parameter with a concrete effect
// contract passed to an effect-polymorphic callee checks (it needs
// SetLiteralLambdas in this path too).
func TestRegistryFuncLitAnnotations(t *testing.T) {
	reg := NewModuleRegistry()
	_, err := reg.LoadModule("m/bad", "module m/bad\nexport pure func run() -> int {\n  let f = func(x: int) -> int { 0 };\n  f(\"str\")\n}\n")
	if err == nil || !strings.Contains(err.Error(), "int vs string") {
		t.Fatalf("a literal's annotation must be enforced in the registry, got %v", err)
	}
	ok := "module m/ok\ntype Ex = { v: int }\n" +
		"func poly(n: int, cb: int -> () ! {e}) -> int ! {e} { cb(n); n }\n" +
		"export func live() -> int ! {IO, Trace} {\n" +
		"  let step = func(n: int, cb: int -> () ! {IO}) -> Ex ! {IO, Trace} { let m = poly(n, cb); { v: m } };\n" +
		"  step(1, \\x. println(show(x))).v\n}\n"
	if _, err := reg.LoadModule("m/ok", ok); err != nil {
		t.Fatalf("concrete-contract callback literal must load: %v", err)
	}
}
