package testing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func runnerAndSuiteFromSource(t *testing.T, src string) (*Runner, *TestSuite) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "contract_domain.ail")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	p := parser.New(lexer.New(src, path))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	suite := NewCollector(path).Collect(file)
	runner := NewRunner(path)
	runner.executor.SetSourceFile(file)
	return runner, suite
}

func propertyByName(t *testing.T, result *SuiteResult, name string) PropertyResult {
	t.Helper()
	for _, property := range result.Properties {
		if property.Name == name {
			return property
		}
	}
	t.Fatalf("property %q not found in %+v", name, result.Properties)
	return PropertyResult{}
}

func TestFindAllLoweredContractPredicates_OrderAndAssociation(t *testing.T) {
	runner, suite := runnerAndSuiteFromSource(t, `module ordered
export func g(x: int, y: int) -> bool ! {}
requires { x > 0, y > 0 }
ensures { result == true } { x > 0 }
export func h(x: int) -> bool ! {}
ensures { result == true } { true }
`)
	runner.RunSuite(suite)

	var g PropertyCase
	for _, property := range suite.Properties {
		if property.FunctionCtx == "g" {
			g = property
			break
		}
	}
	predicates := runner.findAllLoweredContractPredicates(g, core.RequiresKind)
	if len(predicates) != 2 {
		t.Fatalf("g requires predicate count = %d, want 2", len(predicates))
	}
	if predicates[0] == predicates[1] {
		t.Fatal("source-ordered requires predicates unexpectedly alias")
	}
	params := []EnsuresParam{
		{Name: "x", Value: &core.Lit{Value: int64(1)}},
		{Name: "y", Value: &core.Lit{Value: int64(-1)}},
	}
	first, err := runner.executor.EvaluateRequiresHarnessFromCore(params, predicates[0])
	if err != nil {
		t.Fatalf("evaluate first predicate: %v", err)
	}
	second, err := runner.executor.EvaluateRequiresHarnessFromCore(params, predicates[1])
	if err != nil {
		t.Fatalf("evaluate second predicate: %v", err)
	}
	if first.String() != "true" || second.String() != "false" {
		t.Fatalf("predicate order = %v, %v; want x > 0 then y > 0", first, second)
	}
}

func TestFindAllLoweredContractPredicates_ZeroRequires(t *testing.T) {
	runner, suite := runnerAndSuiteFromSource(t, `module zero_requires
export func h(x: int) -> bool ! {}
ensures { result == true } { true }
`)
	runner.RunSuite(suite)
	predicates := runner.findAllLoweredContractPredicates(suite.Properties[0], core.RequiresKind)
	if len(predicates) != 0 {
		t.Fatalf("h requires predicate count = %d, want 0", len(predicates))
	}
}

func TestEnsuresFiltersOutOfContractInputs(t *testing.T) {
	result := runEnsuresFromSource(t, `module precond2
export func big(x: int) -> bool ! {}
requires { x > 100 }
ensures { result == true } { x > 100 }
`)
	property := propertyByName(t, result, "big_property_2")
	if property.Status != StatusPass || property.TestsRun != 100 {
		t.Fatalf("property = %+v, want 100-case pass", property)
	}
	if property.DiscardedInputs == 0 || property.GeneratedInputs != 100+property.DiscardedInputs {
		t.Fatalf("generated/discarded = %d/%d, want generated=100+discarded and discarded>0",
			property.GeneratedInputs, property.DiscardedInputs)
	}
}

func TestEnsuresGenuineViolationStillFails(t *testing.T) {
	result := runEnsuresFromSource(t, `module precond_negative
export func broken(x: int) -> bool ! {}
requires { x > 100 }
ensures { result == true } { false }
`)
	property := propertyByName(t, result, "broken_property_2")
	if property.Status != StatusFail || !strings.Contains(property.Error, "ensures violated") {
		t.Fatalf("property = %+v, want genuine ensures failure", property)
	}
	input := strings.TrimPrefix(property.Error, "ensures violated for input: x=")
	if input == property.Error || strings.HasPrefix(input, "-") || input == "0" {
		t.Fatalf("counterexample %q does not prove requires x > 100", property.Error)
	}
}

func TestEnsuresUnreachableDomainIsUnverifiedSkip(t *testing.T) {
	result := runEnsuresFromSource(t, `module unreachable
export func impossible(x: int) -> bool ! {}
requires { x > 1000 && x < -1000 }
ensures { result == true } { false }
`)
	property := propertyByName(t, result, "impossible_property_2")
	if property.Status != StatusSkip || property.SkipKind != SkipKindOutOfContract ||
		property.TestsRun != 0 || property.GeneratedInputs != 1000 || property.DiscardedInputs != 1000 ||
		!strings.HasPrefix(property.Error, "unverified:") {
		t.Fatalf("property = %+v, want exact unreachable-domain skip", property)
	}
}

// TestEnsuresSparseDomainIsSkipNotPass pins that a domain the generators hit only
// rarely exhausts the attempt cap and reports skip rather than pass.
//
// NOTE ON SCOPE (renamed 2026-07-31, iteration 126): this test was originally named
// TestEnsuresNinetyNineAcceptedIsNotAPass, which claimed more than it proves. With
// `x > 900` the int generator accepts roughly 5% of inputs, so TestsRun settles near
// 50 — never 99. A controller mutation that relaxed the guard to
// `TestsRun < requiredAccepted-1` (i.e. accept exactly 99 as a pass) SURVIVED this
// test, because the scenario never reaches 99.
//
// Pinning the exact requiredAccepted-1 boundary needs a run that lands on exactly 99
// accepted, which is not constructible while generation is wall-clock seeded. That
// makes it blocked on #535 (deterministic seeding, milestone M2). When --seed lands,
// add a seeded case that hits 99 exactly and assert skip.
func TestEnsuresSparseDomainIsSkipNotPass(t *testing.T) {
	result := runEnsuresFromSource(t, `module sparse
export func sparse(x: int) -> bool ! {}
requires { x > 900 }
ensures { result == true } { true }
`)
	property := propertyByName(t, result, "sparse_property_2")
	if property.Status != StatusSkip || property.TestsRun >= 100 {
		t.Fatalf("property = %+v, want fewer than 100 accepted and skip", property)
	}
}

func TestEnsuresNoRequiresFastPath(t *testing.T) {
	result := runEnsuresFromSource(t, `module no_requires
export func ok(x: int) -> bool ! {}
ensures { result == true } { true }
`)
	property := propertyByName(t, result, "ok_property_1")
	if property.Status != StatusPass || property.TestsRun != 100 ||
		property.GeneratedInputs != 100 || property.DiscardedInputs != 0 {
		t.Fatalf("property = %+v, want unchanged 100-case fast path", property)
	}
}

func TestEnsuresRequiresEvaluationErrorFailsLoudly(t *testing.T) {
	runner := NewRunner("requires_error.ail")
	_, err := runner.allRequiresHold(nil, []core.CoreExpr{&core.Lit{Value: int64(1)}})
	if err == nil || !strings.Contains(err.Error(), "requires predicate must return bool") {
		t.Fatalf("error = %v, want loud non-boolean requires failure", err)
	}
}

func TestEnsuresDiscardedTupleNeverCallsFunction(t *testing.T) {
	result := runEnsuresFromSource(t, `module never_call
export func dangerous(x: int) -> int ! {}
requires { x > 1000 && x < -1000 }
ensures { result == 0 } { 1 / 0 }
`)
	property := propertyByName(t, result, "dangerous_property_2")
	if property.Status != StatusSkip || property.DiscardedInputs != 1000 ||
		strings.Contains(property.Error, "division") {
		t.Fatalf("property = %+v, discarded input called function", property)
	}
}
