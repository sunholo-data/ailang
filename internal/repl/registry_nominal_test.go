package repl

import (
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
