package runtime

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
)

// TestAttachContracts_CarriesTheFunctionName (M-V1-VERIFICATION-HONESTY M2b):
// every spec names its declaring function, so a violation reads "ensures
// failed in half at …". Red mutation: drop the Function assignment.
func TestAttachContracts_CarriesTheFunctionName(t *testing.T) {
	fn := &eval.FunctionValue{}
	attachContracts(fn, "half", []*core.Contract{
		{Kind: core.RequiresKind, Message: "x >= 0"},
		{Kind: core.EnsuresKind, Message: "result >= 0"},
		{Kind: core.InvariantKind, Message: "inv"},
	})
	if len(fn.Preconditions) != 2 || len(fn.Postconditions) != 2 {
		t.Fatalf("pre/post = %d/%d, want 2/2 (an invariant is both)", len(fn.Preconditions), len(fn.Postconditions))
	}
	for _, s := range append(fn.Preconditions, fn.Postconditions...) {
		if s.Function != "half" {
			t.Fatalf("spec %+v does not name its function", s)
		}
	}
}
