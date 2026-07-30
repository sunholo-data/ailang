package testing

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestReporter_VacuousPropertyDetails(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "anchor", Status: StatusPass})
	result.AddPropertyResult(PropertyResult{
		Name:     "shiftX",
		Status:   StatusSkip,
		SkipKind: SkipKindNoGenerator,
		Error:    "no generator for parameter p: Point",
	})

	var jsonBuf bytes.Buffer
	if err := NewReporter(FormatJSON, &jsonBuf, false).Report(result); err != nil {
		t.Fatal(err)
	}
	var output struct {
		Success      bool `json:"success"`
		VacuousSkips int  `json:"vacuous_skips"`
		Properties   []struct {
			SkipKind string `json:"skip_kind"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(jsonBuf.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Success || output.VacuousSkips != 1 {
		t.Fatalf("expected success=false and vacuous_skips=1, got %+v", output)
	}
	if got := output.Properties[0].SkipKind; got != SkipKindNoGenerator {
		t.Fatalf("expected skip_kind %q, got %q", SkipKindNoGenerator, got)
	}

	var humanBuf bytes.Buffer
	if err := NewReporter(FormatHuman, &humanBuf, false).Report(result); err != nil {
		t.Fatal(err)
	}
	human := humanBuf.String()
	if !strings.Contains(human, "no generator for parameter p: Point") {
		t.Fatalf("missing skip reason in human report:\n%s", human)
	}
	if !strings.Contains(human, "1 properties never ran") {
		t.Fatalf("missing vacuous summary in human report:\n%s", human)
	}
	if strings.Contains(human, "All tests passed!") {
		t.Fatalf("vacuous suite reported success:\n%s", human)
	}
}

func TestReporter_OutOfContractShowsReasonAndCases(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "anchor", Status: StatusPass})
	result.AddPropertyResult(PropertyResult{
		Name:     "bounded",
		Status:   StatusSkip,
		SkipKind: SkipKindOutOfContract,
		TestsRun: 2,
		Error:    "requires not satisfied by random input",
	})

	var buf bytes.Buffer
	if err := NewReporter(FormatHuman, &buf, false).Report(result); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "(2 cases,") {
		t.Fatalf("missing executed case count:\n%s", output)
	}
	if !strings.Contains(output, "requires not satisfied by random input") {
		t.Fatalf("missing out-of-contract reason:\n%s", output)
	}
	if !strings.Contains(output, "All tests passed!") {
		t.Fatalf("forgiven out-of-contract skip should stay green:\n%s", output)
	}
}
