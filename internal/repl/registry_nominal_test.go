package repl

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-V1-NOMINAL-TYPE-IDENTITY: the REPL/WASM module registry resolves type
// names with the CLI pipeline's rules (registry_type_scope.go). A module that
// imports q/a's T by name gets q/a's T; a module that declares its own T does
// not capture q/a's (row a″).
func TestRegistryNominalIdentity(t *testing.T) {
	reg := NewModuleRegistry()
	if _, err := reg.LoadModule("q/a", "module q/a\nexport type T = A(int)\nexport pure func mk() -> T = A(42)\n"); err != nil {
		t.Fatalf("load q/a: %v", err)
	}
	ok := "module q/ok\nimport q/a (T, A, mk)\nexport pure func use(t: T) -> int = match t { A(n) => n }\nexport pure func run() -> int = use(mk())\n"
	if _, err := reg.LoadModule("q/ok", ok); err != nil {
		t.Fatalf("importing q/a's T by name must check: %v", err)
	}
	capture := "module q/cap\nimport q/a (mk)\ntype T = B(string)\nexport pure func use(t: T) -> string = match t { B(s) => s }\nexport pure func run() -> string = use(mk())\n"
	_, err := reg.LoadModule("q/cap", capture)
	if err == nil {
		t.Fatal("a local T captured q/a's T in the registry loader")
	}
	if !strings.Contains(err.Error(), types.TC_NOMINAL_001) || !strings.Contains(err.Error(), "q/a") {
		t.Fatalf("expected TC_NOMINAL_001 naming q/a, got: %v", err)
	}
}

// Review round 1 (probe repl_registry_probe_test.go.txt): loading a module
// that declares a same-named constructor must not break another module's
// constructor lookups (it used to: the flat table kept the last A loaded),
// and a correct module importing q/a's A by name must load. A module that uses
// A without saying whose is ambiguous, loudly.
func TestRegistryNominalIdentity_SameNamedConstructorsCoexist(t *testing.T) {
	reg := NewModuleRegistry()
	mustLoad := func(n, src string) {
		t.Helper()
		if _, err := reg.LoadModule(n, src); err != nil {
			t.Fatalf("load %s: %v", n, err)
		}
	}
	mustLoad("q/a", "module q/a\nexport type T = A(int)\nexport pure func mk() -> T = A(42)\nexport pure func get(t: T) -> int = match t { A(n) => n }\n")
	mustLoad("q/b", "module q/b\nexport type T = A(string)\nexport pure func mkb() -> T = A(\"s\")\nexport pure func use(t: T) -> string = match t { A(s) => \"${s}!\" }\n")
	if _, err := reg.LoadModule("q/x", "module q/x\nimport q/a (mk)\nimport q/b (use)\nexport pure func run() -> string = use(mk())\n"); err == nil {
		t.Fatal("q/x passes q/a's T to q/b's use: must be rejected")
	}
	mustLoad("q/ok", "module q/ok\nimport q/a (T, A, mk, get)\nexport pure func run() -> int = get(A(5)) + get(mk())\nexport pure func run2() -> int = match mk() { A(n) => n }\n")
	_, err := reg.LoadModule("q/leak", "module q/leak\nexport pure func mkA() = A(\"leaked\")\n")
	if err == nil || !strings.Contains(err.Error(), types.TC_AMBIGUOUS_CTOR_001) {
		t.Fatalf("A without an import is offered by q/a and q/b: want TC_AMBIGUOUS_CTOR_001, got %v", err)
	}
	// Review round 2 (d1/d2): two explicit imports of one bare constructor are
	// MOD015 in the registry too, in both orders (the last one used to win).
	for _, src := range []string{
		"module q/d1\nimport q/a (A, get)\nimport q/b (A)\nexport pure func run() -> int = get(A(1))\n",
		"module q/d2\nimport q/b (A)\nimport q/a (A, get)\nexport pure func run() -> int = get(A(1))\n",
		"module q/d3\nimport q/a (T)\nimport q/b (T)\nexport pure func run() -> int = 0\n",
	} {
		name := strings.Fields(src)[1]
		if _, err := reg.LoadModule(name, src); err == nil || !strings.Contains(err.Error(), "MOD015") {
			t.Fatalf("%s: two explicit imports of one bare name must be MOD015, got %v", name, err)
		}
	}
	// A pattern on q/a's T with q/b's A also loaded resolves by the scrutinee.
	mustLoad("q/pat", "module q/pat\nimport q/a (mk)\nimport q/b (mkb)\nexport pure func run() -> int = match mk() { A(n) => n + 1 }\n")
	for _, c := range []struct {
		mod, fn, want string
	}{{"q/a", "mk", "A(42)"}, {"q/b", "mkb", "A(s)"}, {"q/ok", "run", "47"}, {"q/ok", "run2", "42"}, {"q/pat", "run", "43"}} {
		v, err := reg.InvokeExport(c.mod, c.fn, nil)
		if err != nil {
			t.Fatalf("%s.%s: %v", c.mod, c.fn, err)
		}
		if got := fmt.Sprint(v); got != c.want {
			t.Errorf("%s.%s = %s, want %s", c.mod, c.fn, got, c.want)
		}
	}
}
