package testing

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// TestDerive_ListMediatedRecursiveRecordBounded pins the depth budget across
// the list arms. A record type that references itself through a list
// (`type Tree = { val: int, kids: [Tree] }`) is only reachable since M3 made
// named types derivable; if the list arms derive their element with a fresh
// root context instead of the descended one, every pass through the cycle
// resets the budget and derivation overflows the stack (measured: goroutine
// stack exhaustion at iteration-170 review). At the bound the derivation must
// refuse — an honest vacuous skip, never a crash.
func TestDerive_ListMediatedRecursiveRecordBounded(t *testing.T) {
	src := `module derive_cycle

export type Tree = { val: int, kids: [Tree] }

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	gen, shrink := r.createGeneratorForType(&ast.SimpleType{Name: "Tree"})
	if gen != nil || shrink != nil {
		t.Fatalf("expected nil, nil for list-mediated recursive record, got generator=%T shrinker=%T", gen, shrink)
	}
}
