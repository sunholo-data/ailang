package bestof

import "testing"

type cannedV struct{ vs []Verdict }

func (c cannedV) Verify(path string) Verdict {
	for i, p := range []string{"a", "b", "c"} {
		if p == path && i < len(c.vs) {
			return c.vs[i]
		}
	}
	return Verdict{}
}

// A runs-but-WRONG candidate (Runs, !ContractsPass) must lose to a runs+contracts one.
func TestContractTierSelectsContractPasser(t *testing.T) {
	v := cannedV{vs: []Verdict{
		{TypeChecks: true, Runs: true, ContractsPass: false}, // runs-but-wrong (a)
		{TypeChecks: true, Runs: true, ContractsPass: true},  // runs + contracts (b)
		{TypeChecks: true, Runs: false},                      // typechecks only (c)
	}}
	best, _ := SelectBest([]string{"a", "b", "c"}, v)
	if best != 1 {
		t.Errorf("best=%d, want 1 (the runs+contracts candidate over runs-but-wrong)", best)
	}
}

// Without contract info (ContractsPass false for all), behaviour is the old runs>typechecks>neither.
func TestContractTierBackwardCompatible(t *testing.T) {
	v := cannedV{vs: []Verdict{
		{TypeChecks: true, Runs: false}, // typechecks (a)
		{TypeChecks: true, Runs: true},  // runs (b) — should win
		{},                              // neither (c)
	}}
	best, _ := SelectBest([]string{"a", "b", "c"}, v)
	if best != 1 {
		t.Errorf("best=%d, want 1 (runs beats typechecks/neither when no contract tier)", best)
	}
}
