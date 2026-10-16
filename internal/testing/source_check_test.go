package testing

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSourceCheckReport(t *testing.T) {
	for _, checked := range []bool{false, true} {
		suite := NewSuiteResult("probe/p")
		suite.AddTestResult(TestResult{Name: "passing", Status: StatusPass})
		if checked {
			suite.SourceCheck = &SourceCheckResult{Passed: 2, Failed: 1, Errors: []string{"bad.ail: type error"}, Warnings: []string{}}
		}
		var out bytes.Buffer
		if err := NewReporter(FormatJSON, &out, false).Report(suite); err != nil {
			t.Fatal(err)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		_, present := doc["source_check"]
		if present != checked {
			t.Fatalf("source_check present=%v, checked=%v", present, checked)
		}
		for _, key := range []string{"module_path", "total_tests", "passed_tests", "failed_tests", "success", "tests", "properties", "seed_mode", "seed", "seed_derivation"} {
			if _, ok := doc[key]; !ok {
				t.Errorf("missing existing key %s", key)
			}
		}
		if checked {
			if suite.Success() || suite.SuccessAllowingSkips() {
				t.Fatal("source failure waived by success predicate")
			}
			var check SourceCheckResult
			if err := json.Unmarshal(doc["source_check"], &check); err != nil {
				t.Fatal(err)
			}
			if check.Passed != 2 || check.Failed != 1 || len(check.Errors) != 1 {
				t.Fatalf("round trip: %+v", check)
			}
			out.Reset()
			if err := NewReporter(FormatHuman, &out, false).Report(suite); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "Package source check failed") || strings.Contains(out.String(), "All tests passed") {
				t.Fatal(out.String())
			}
		}
	}
}

func TestSourceCheckEmptyAndSkippedSuites(t *testing.T) {
	for _, empty := range []bool{false, true} {
		suite := NewSuiteResult("probe/p")
		suite.SourceCheck = &SourceCheckResult{Errors: []string{}, Warnings: []string{}}
		if !empty {
			suite.AddTestResult(TestResult{Status: StatusSkip})
		}
		if suite.Success() || !suite.SuccessAllowingSkips() {
			t.Fatal("empty/skipped suite semantics changed")
		}
		suite.SourceCheck.Failed = 1
		if suite.Success() || suite.SuccessAllowingSkips() {
			t.Fatal("source failure waived")
		}
	}
}
