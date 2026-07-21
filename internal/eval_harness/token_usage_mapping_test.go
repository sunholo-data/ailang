package eval_harness

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestTokenUsageFromResult_CarriesEveryCount guards the executor -> banked-result
// boundary.
//
// Every field is set to a DISTINCT value so a copy-paste mixup (reading
// OutputTokens into ReasonTokens, say) fails instead of silently passing the way
// it would if they shared a value.
func TestTokenUsageFromResult_CarriesEveryCount(t *testing.T) {
	got := tokenUsageFromResult(&executor.Result{
		InputTokens:              11,
		OutputTokens:             22,
		ReasonTokens:             33,
		CacheReadInputTokens:     44,
		CacheCreationInputTokens: 55,
	})

	want := TokenUsage{
		InputTokens:              11,
		OutputTokens:             22,
		ReasonTokens:             33,
		CacheReadInputTokens:     44,
		CacheCreationInputTokens: 55,
	}

	if got != want {
		t.Errorf("tokenUsageFromResult() = %+v, want %+v", got, want)
	}
}

func TestTokenUsageFromResult_NilResult(t *testing.T) {
	if got := tokenUsageFromResult(nil); got != (TokenUsage{}) {
		t.Errorf("tokenUsageFromResult(nil) = %+v, want zero value", got)
	}
}

// TestTokenUsageCoversEveryExecutorTokenField is the systemic guard.
//
// The recurring failure mode in this codebase is not a wrong mapping, it is an
// ABSENT one: a token field gets added to executor.Result and parsed correctly by
// an executor, but nobody copies it across this boundary, so it banks as 0
// forever and is indistinguishable from "the provider didn't report it". That is
// exactly how the entire v0.30.0 baseline came to understate cost
// (eval_results/baselines/v0.30.0/CAVEATS.md).
//
// Testing tokenUsageFromResult's current fields would not catch the NEXT one, so
// this asserts the structural invariant instead: every *Tokens field on
// executor.Result has a counterpart on TokenUsage. If you add a token field and
// this fails, map it in tokenUsageFromResult — do not just add it to the skip
// list.
func TestTokenUsageCoversEveryExecutorTokenField(t *testing.T) {
	resultType := reflect.TypeOf(executor.Result{})
	usageType := reflect.TypeOf(TokenUsage{})

	for i := 0; i < resultType.NumField(); i++ {
		name := resultType.Field(i).Name
		if !strings.HasSuffix(name, "Tokens") {
			continue
		}
		if _, ok := usageType.FieldByName(name); !ok {
			t.Errorf("executor.Result.%s has no counterpart on TokenUsage — it will bank as 0 "+
				"and be indistinguishable from 'not reported'. Map it in tokenUsageFromResult.", name)
		}
	}
}

// TestTokenUsage_ReasonTokensDisjointFromOutput pins the arithmetic contract the
// banking site in cmd/ailang/eval_benchmark.go relies on: reasoning is counted in
// the total but NOT already inside OutputTokens. If an executor ever folds
// thinking into its output count (managed_agents used to), the total
// double-counts it and agent-mode token figures inflate silently.
func TestTokenUsage_ReasonTokensDisjointFromOutput(t *testing.T) {
	u := tokenUsageFromResult(&executor.Result{
		InputTokens:  100,
		OutputTokens: 20,
		ReasonTokens: 500,
	})

	total := u.InputTokens + u.OutputTokens + u.ReasonTokens +
		u.CacheCreationInputTokens + u.CacheReadInputTokens
	if total != 620 {
		t.Errorf("total = %d, want 620 (100 input + 20 output + 500 reasoning)", total)
	}

	// A model that thought far more than it emitted is the normal case for
	// reasoning models, not a bug: in the v0.30.0 baseline the one correctly
	// measured model reasoned 8.45x its output.
	if u.OutputTokens >= u.ReasonTokens {
		t.Fatal("fixture no longer exercises reasoning-heavy case")
	}
}
