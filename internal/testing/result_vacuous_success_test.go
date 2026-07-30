package testing

import "testing"

func TestSuiteResult_Success_MixedVacuousPropertyFails(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "sibling", Status: StatusPass})
	result.AddPropertyResult(PropertyResult{
		Name:     "vacuous",
		Status:   StatusSkip,
		SkipKind: SkipKindNoGenerator,
	})

	if result.Success() {
		t.Fatal("mixed suite with a vacuous property must not succeed")
	}
	if !result.SuccessAllowingSkips() {
		t.Fatal("--allow-skips semantics must forgive vacuous properties")
	}
}

func TestSuiteResult_Success_MixedOutOfContractPropertyPasses(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "sibling", Status: StatusPass})
	result.AddPropertyResult(PropertyResult{
		Name:     "discarded",
		Status:   StatusSkip,
		SkipKind: SkipKindOutOfContract,
		TestsRun: 1,
	})

	if !result.Success() {
		t.Fatal("out-of-contract discard must not make a mixed suite fail")
	}
}

func TestSuiteResult_Success_AllSkippedOutOfContractFails(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:     "discarded",
		Status:   StatusSkip,
		SkipKind: SkipKindOutOfContract,
		TestsRun: 1,
	})

	if result.Success() {
		t.Fatal("all-skipped suite must not succeed")
	}
	if !result.AllSkipped() {
		t.Fatal("all-skipped suite must retain NO TESTS RAN classification")
	}
}
