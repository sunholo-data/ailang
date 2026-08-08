package testing

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReporter_JSON_Empty(t *testing.T) {
	result := NewSuiteResult("test.ail")

	var buf bytes.Buffer
	reporter := NewReporter(FormatJSON, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify fields
	if output["module_path"] != "test.ail" {
		t.Errorf("expected module_path 'test.ail', got %v", output["module_path"])
	}

	if output["total_tests"].(float64) != 0 {
		t.Errorf("expected total_tests 0, got %v", output["total_tests"])
	}

	if output["success"].(bool) != false {
		t.Error("expected success false (no tests run)")
	}
}

func TestReporter_JSON_PassingTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:     "test1",
		Status:   StatusPass,
		Duration: 100 * time.Millisecond,
		Location: "test.ail:5:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatJSON, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify summary fields
	if output["passed_tests"].(float64) != 1 {
		t.Errorf("expected passed_tests 1, got %v", output["passed_tests"])
	}

	if output["failed_tests"].(float64) != 0 {
		t.Errorf("expected failed_tests 0, got %v", output["failed_tests"])
	}

	if output["success"].(bool) != true {
		t.Error("expected success true")
	}

	// Verify test details
	tests := output["tests"].([]interface{})
	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}

	test := tests[0].(map[string]interface{})
	if test["name"] != "test1" {
		t.Errorf("expected name 'test1', got %v", test["name"])
	}

	if test["status"] != "pass" {
		t.Errorf("expected status 'pass', got %v", test["status"])
	}

	if test["location"] != "test.ail:5:1" {
		t.Errorf("expected location 'test.ail:5:1', got %v", test["location"])
	}
}

func TestReporter_JSON_FailingTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:     "test1",
		Status:   StatusFail,
		Error:    "assertion failed: expected 4, got 5",
		Duration: 50 * time.Millisecond,
		Location: "test.ail:10:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatJSON, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify summary
	if output["failed_tests"].(float64) != 1 {
		t.Errorf("expected failed_tests 1, got %v", output["failed_tests"])
	}

	if output["success"].(bool) != false {
		t.Error("expected success false")
	}

	// Verify test error
	tests := output["tests"].([]interface{})
	test := tests[0].(map[string]interface{})

	if test["error"] != "assertion failed: expected 4, got 5" {
		t.Errorf("expected error message, got %v", test["error"])
	}
}

func TestReporter_JSON_PropertyResult(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:     "prop1",
		Status:   StatusPass,
		TestsRun: 100,
		Duration: 500 * time.Millisecond,
		Location: "test.ail:20:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatJSON, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify property details
	props := output["properties"].([]interface{})
	if len(props) != 1 {
		t.Fatalf("expected 1 property, got %d", len(props))
	}

	prop := props[0].(map[string]interface{})
	if prop["name"] != "prop1" {
		t.Errorf("expected name 'prop1', got %v", prop["name"])
	}

	if prop["tests_run"].(float64) != 100 {
		t.Errorf("expected tests_run 100, got %v", prop["tests_run"])
	}

	if prop["status"] != "pass" {
		t.Errorf("expected status 'pass', got %v", prop["status"])
	}
}

func TestReporter_JSONIncludesDiscardCounters(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:            "pass",
		Status:          StatusPass,
		TestsRun:        100,
		GeneratedInputs: 120,
		DiscardedInputs: 20,
	})
	result.AddPropertyResult(PropertyResult{
		Name:            "fail",
		Status:          StatusFail,
		TestsRun:        1,
		GeneratedInputs: 2,
		DiscardedInputs: 1,
	})
	result.AddPropertyResult(PropertyResult{
		Name:            "skip",
		Status:          StatusSkip,
		SkipKind:        SkipKindOutOfContract,
		GeneratedInputs: 1000,
		DiscardedInputs: 1000,
	})

	var buf bytes.Buffer
	if err := NewReporter(FormatJSON, &buf, false).Report(result); err != nil {
		t.Fatalf("Report() error: %v", err)
	}
	var output struct {
		Properties []struct {
			GeneratedInputs int `json:"generated_inputs"`
			DiscardedInputs int `json:"discarded_inputs"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}
	want := [][2]int{{120, 20}, {2, 1}, {1000, 1000}}
	for i, property := range output.Properties {
		if got := [2]int{property.GeneratedInputs, property.DiscardedInputs}; got != want[i] {
			t.Errorf("property %d counters = %v, want %v", i, got, want[i])
		}
	}
}

func TestReporter_HumanIncludesDiscardCounters(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:            "filtered",
		Status:          StatusPass,
		TestsRun:        100,
		GeneratedInputs: 125,
		DiscardedInputs: 25,
	})
	var buf bytes.Buffer
	if err := NewReporter(FormatHuman, &buf, false).Report(result); err != nil {
		t.Fatalf("Report() error: %v", err)
	}
	if !strings.Contains(buf.String(), "accepted 100, discarded 25, generated 125") {
		t.Fatalf("human output missing discard counters:\n%s", buf.String())
	}
}

func TestReporter_Human_Empty(t *testing.T) {
	result := NewSuiteResult("test.ail")

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false) // No colors for easier testing

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for key sections
	if !strings.Contains(output, "Test Results") {
		t.Error("expected 'Test Results' header")
	}

	if !strings.Contains(output, "Module: test.ail") {
		t.Error("expected module path")
	}

	if !strings.Contains(output, "No tests found") {
		t.Error("expected 'No tests found' message")
	}
}

func TestReporter_Human_PassingTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:     "simple test",
		Status:   StatusPass,
		Duration: 10 * time.Millisecond,
		Location: "test.ail:5:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for test details
	if !strings.Contains(output, "simple test") {
		t.Error("expected test name in output")
	}

	if !strings.Contains(output, "✓") {
		t.Error("expected pass icon")
	}

	if !strings.Contains(output, "All tests passed!") {
		t.Error("expected success message")
	}

	if !strings.Contains(output, "1 passed") {
		t.Error("expected '1 passed' in summary")
	}
}

func TestReporter_Human_FailingTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:     "failing test",
		Status:   StatusFail,
		Error:    "assertion failed",
		Location: "test.ail:10:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for test details
	if !strings.Contains(output, "failing test") {
		t.Error("expected test name in output")
	}

	if !strings.Contains(output, "✗") {
		t.Error("expected fail icon")
	}

	if !strings.Contains(output, "assertion failed") {
		t.Error("expected error message in output")
	}

	if !strings.Contains(output, "Some tests failed") {
		t.Error("expected failure message")
	}

	if !strings.Contains(output, "1 failed") {
		t.Error("expected '1 failed' in summary")
	}
}

func TestReporter_Human_SkippedTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:   "skipped test",
		Status: StatusSkip,
		Error:  "Property testing not yet implemented",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for skip icon
	if !strings.Contains(output, "⊘") {
		t.Error("expected skip icon")
	}

	if !strings.Contains(output, "1 skipped") {
		t.Error("expected '1 skipped' in summary")
	}
}

func TestReporter_Human_MixedResults(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "test1", Status: StatusPass})
	result.AddTestResult(TestResult{Name: "test2", Status: StatusPass})
	result.AddTestResult(TestResult{Name: "test3", Status: StatusFail, Error: "failed"})
	result.AddTestResult(TestResult{Name: "test4", Status: StatusSkip})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check summary counts
	if !strings.Contains(output, "4 tests") {
		t.Error("expected '4 tests' in summary")
	}

	if !strings.Contains(output, "2 passed") {
		t.Error("expected '2 passed' in summary")
	}

	if !strings.Contains(output, "1 failed") {
		t.Error("expected '1 failed' in summary")
	}

	if !strings.Contains(output, "1 skipped") {
		t.Error("expected '1 skipped' in summary")
	}
}

func TestReporter_Human_PropertyResult(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:     "always positive",
		Status:   StatusPass,
		TestsRun: 100,
		Duration: 200 * time.Millisecond,
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for property details
	if !strings.Contains(output, "Properties:") {
		t.Error("expected 'Properties:' section")
	}

	if !strings.Contains(output, "always positive") {
		t.Error("expected property name in output")
	}

	if !strings.Contains(output, "100 cases") {
		t.Error("expected '100 cases' in output")
	}
}

func TestReporter_Human_FailingProperty(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:         "fails sometimes",
		Status:       StatusFail,
		TestsRun:     50,
		FailingInput: "x=42, y=-1",
		Error:        "property violated",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for failure details
	if !strings.Contains(output, "Failing input") {
		t.Error("expected 'Failing input' label")
	}

	if !strings.Contains(output, "x=42, y=-1") {
		t.Error("expected failing input values")
	}

	if !strings.Contains(output, "property violated") {
		t.Error("expected error message")
	}
}

func TestReporter_Colors_Disabled(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "test1", Status: StatusPass})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false) // Colors disabled

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check that no ANSI escape codes are present
	if strings.Contains(output, "\033[") {
		t.Error("expected no ANSI color codes when colors disabled")
	}
}

func TestReporter_Colors_Enabled(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "test1", Status: StatusPass})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, true) // Colors enabled

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check that ANSI escape codes ARE present
	if !strings.Contains(output, "\033[") {
		t.Error("expected ANSI color codes when colors enabled")
	}
}

func TestReporter_UnknownFormat(t *testing.T) {
	result := NewSuiteResult("test.ail")

	var buf bytes.Buffer
	reporter := NewReporter("unknown", &buf, false)

	err := reporter.Report(result)
	if err == nil {
		t.Error("expected error for unknown format")
	}

	if !strings.Contains(err.Error(), "unknown output format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// S15 — the top-level and per-property seed fields must decode as decimal
// STRINGS, never JSON numbers. A JSON number is float64 on decode and silently
// loses precision above 2^53; both cases here exercise values beyond that bound
// (positive and negative) so a JSON-number mutant genuinely discriminates.
func TestReporter_JSONSeedFieldsAreDecimalStrings(t *testing.T) {
	seedRe := regexp.MustCompile(`^-?[0-9]+$`)
	cases := []struct {
		master   int64
		propSeed int64
	}{
		{master: 9007199254740993, propSeed: 9007199254740999},   // 2^53+1 / 2^53+7
		{master: -9007199254740993, propSeed: -9007199254747319}, // negative round-trip
	}
	for _, c := range cases {
		result := NewSuiteResult("case/stable.ail")
		result.SetSeedMetadata(TestConfig{WorkspaceRoot: "/work", SeedMode: SeedModeMaster, MasterSeed: c.master})
		result.AddPropertyResult(PropertyResult{Name: "p", Status: StatusPass, Seed: c.propSeed})

		var buf bytes.Buffer
		if err := NewReporter(FormatJSON, &buf, false).Report(result); err != nil {
			t.Fatalf("Report() error: %v", err)
		}
		var output map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		// Top-level seed: a decimal string matching the master exactly.
		topSeed, ok := output["seed"].(string)
		if !ok {
			t.Fatalf("master %d: top-level seed is %T (%v), want string", c.master, output["seed"], output["seed"])
		}
		if !seedRe.MatchString(topSeed) {
			t.Errorf("master %d: top-level seed %q does not match decimal regex", c.master, topSeed)
		}
		if topSeed != strconv.FormatInt(c.master, 10) {
			t.Errorf("master %d: round-trip got %q, want %q", c.master, topSeed, strconv.FormatInt(c.master, 10))
		}

		// Derivation tag.
		if output["seed_derivation"] != SeedDerivationV1 {
			t.Errorf("master %d: seed_derivation = %v, want %q", c.master, output["seed_derivation"], SeedDerivationV1)
		}

		// Per-property seed: also a decimal string.
		prop := output["properties"].([]interface{})[0].(map[string]interface{})
		propSeed, ok := prop["seed"].(string)
		if !ok {
			t.Fatalf("master %d: property seed is %T (%v), want string", c.master, prop["seed"], prop["seed"])
		}
		if !seedRe.MatchString(propSeed) {
			t.Errorf("master %d: property seed %q does not match decimal regex", c.master, propSeed)
		}
		if propSeed != strconv.FormatInt(c.propSeed, 10) {
			t.Errorf("master %d: property seed round-trip got %q, want %q", c.master, propSeed, strconv.FormatInt(c.propSeed, 10))
		}
	}
}

// S16 — replay appears on a FAIL property and is absent on pass and skip, and
// equals the exact copy/paste command text (D9).
func TestReporter_ReplayOnlyOnFailure(t *testing.T) {
	result := NewSuiteResult("mods/example.ail")
	result.SetSeedMetadata(TestConfig{SeedMode: SeedModeMaster, MasterSeed: 42})
	result.AddPropertyResult(PropertyResult{Name: "fail_prop", Status: StatusFail})
	result.AddPropertyResult(PropertyResult{Name: "pass_prop", Status: StatusPass})
	result.AddPropertyResult(PropertyResult{Name: "skip_prop", Status: StatusSkip, SkipKind: SkipKindNoGenerator})

	var buf bytes.Buffer
	if err := NewReporter(FormatJSON, &buf, false).Report(result); err != nil {
		t.Fatalf("Report() error: %v", err)
	}
	var output struct {
		Properties []map[string]interface{} `json:"properties"`
	}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}
	props := map[string]map[string]interface{}{}
	for _, p := range output.Properties {
		props[p["name"].(string)] = p
	}

	wantReplay := "ailang test --seed 42 mods/example.ail"
	for name, p := range props {
		got, present := p["replay"]
		if name == "fail_prop" {
			if !present {
				t.Errorf("fail property missing replay key")
			} else if got != wantReplay {
				t.Errorf("fail replay = %v, want %q", got, wantReplay)
			}
		} else if present {
			t.Errorf("%s property should NOT have replay, got %v", name, got)
		}
	}
	for _, name := range []string{"fail_prop", "pass_prop", "skip_prop"} {
		if _, ok := props[name]; !ok {
			t.Errorf("missing property %q in output", name)
		}
	}
}

// T2 — the replay command prefers ReplayTarget (the shell-safe CLI argument
// tail recorded at invocation time) and falls back to ModulePath when
// ReplayTarget is empty, preserving the historical behaviour for library
// callers and tests that build a SuiteResult directly. One arm carries a space
// (and one an embedded single quote) to assert the shell quoting rules.
func TestReplayCommand_FallsBackToModulePath(t *testing.T) {
	// The three non-empty-target arms pass the target the way the CLI records it:
	// ALREADY shell-safe (the quoting lives in replayTargetArg in cmd/ailang — the
	// reporter must emit it verbatim and never fall back to ModulePath). The 'want'
	// strings therefore carry the quotes as produced upstream.
	cases := []struct {
		name         string
		replayTarget string
		modulePath   string
		want         string
	}{
		{
			name:         "empty target falls back to module path",
			replayTarget: "",
			modulePath:   "mods/a.ail",
			want:         "ailang test --seed 7 mods/a.ail",
		},
		{
			name:         "explicit target is preferred over module path",
			replayTarget: "path/to/f.ail",
			modulePath:   "mods/a.ail (ignored)",
			want:         "ailang test --seed 7 path/to/f.ail",
		},
		{
			name:         "target with a space is single-quoted",
			replayTarget: "'dir with space/ f.ail'",
			modulePath:   "mods/a.ail",
			want:         "ailang test --seed 7 'dir with space/ f.ail'",
		},
		{
			name:         "target with an embedded single quote is escaped",
			replayTarget: "'it'\\''s here/ f.ail'",
			modulePath:   "mods/a.ail",
			want:         "ailang test --seed 7 'it'\\''s here/ f.ail'",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := NewSuiteResult(c.modulePath)
			result.SetSeedMetadata(TestConfig{SeedMode: SeedModeMaster, MasterSeed: 7, ReplayTarget: c.replayTarget})
			result.AddPropertyResult(PropertyResult{Name: "fail_prop", Status: StatusFail})

			var buf bytes.Buffer
			if err := NewReporter(FormatJSON, &buf, false).Report(result); err != nil {
				t.Fatalf("Report() error: %v", err)
			}
			var output struct {
				Properties []map[string]interface{} `json:"properties"`
			}
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				t.Fatalf("Invalid JSON output: %v", err)
			}
			if len(output.Properties) != 1 {
				t.Fatalf("expected 1 property, got %d", len(output.Properties))
			}
			if got, ok := output.Properties[0]["replay"].(string); !ok || got != c.want {
				t.Errorf("replay = %q, want %q", got, c.want)
			}
		})
	}
}
