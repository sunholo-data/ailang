package effects

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/trace"
)

// M-V1-VERIFICATION-HONESTY M2b: the evaluator names the function whose
// contract it checked. Before, the violation read "ensures failed in  at …"
// and the trace event's function was empty, because the only way to set the
// name (SetFunction) had no production caller.

func TestEffContextContracts_NameTheFunction(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Contracts = NewContractContextWithMode(ContractModePanic)
	ctx.Trace = trace.NewCollector()

	if err := ctx.CheckRequires("half", true, "(x >= 0)", "k1.ail:4:12"); err != nil {
		t.Fatalf("passing requires returned %v", err)
	}
	err := ctx.CheckEnsures("half", false, "(result >= 0)", "k1.ail:5:11")
	if err == nil || !strings.Contains(err.Error(), "contract violation: ensures failed in half at k1.ail:5:11: (result >= 0)") {
		t.Fatalf("violation = %v, want it to name half", err)
	}
	err = ctx.CheckRequires("other", false, "(x > 0)", "k1.ail:9:1")
	if err == nil || !strings.Contains(err.Error(), "requires failed in other at ") {
		t.Fatalf("requires violation = %v, want it to name other", err)
	}

	var names []string
	for _, ev := range ctx.Trace.Events() {
		if ev.Contract != nil {
			names = append(names, ev.Contract.Kind+":"+ev.Contract.Function)
		}
	}
	if strings.Join(names, ",") != "requires:half,ensures:half,requires:other" {
		t.Fatalf("trace contract events = %v, want each to carry its function", names)
	}
}

func TestEffContextContracts_DisabledIsANoOp(t *testing.T) {
	ctx := NewEffContext(nil)
	if err := ctx.CheckEnsures("half", false, "x", "f.ail:1"); err != nil {
		t.Fatalf("no contract context: got %v, want nil", err)
	}
	if err := ctx.CheckRequires("half", false, "x", "f.ail:1"); err != nil {
		t.Fatalf("no contract context: got %v, want nil", err)
	}
}

// TestContractContext_InRecordsTheGivenFunction: the *In variants record the
// name they are given and never read or mutate the context's own function.
func TestContractContext_InRecordsTheGivenFunction(t *testing.T) {
	c := NewContractContextWithMode(ContractModeReport)
	c.SetFunction("stale")
	_ = c.CheckRequiresIn("f", false, "a", "x.ail:1")
	_ = c.CheckEnsuresIn("g", false, "b", "x.ail:2")
	v := c.Violations()
	if len(v) != 2 || v[0].Function != "f" || v[1].Function != "g" {
		t.Fatalf("violations = %+v, want functions f and g", v)
	}
	if c.CurrentFunction() != "stale" {
		t.Fatalf("CheckRequiresIn mutated the context's function: %q", c.CurrentFunction())
	}
}
