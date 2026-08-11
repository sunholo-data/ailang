package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const jsonOutputTestSource = `module json_output_test

test "passes" { true }
`

// The vacuous vehicle must be a type the runner cannot derive a generator
// for, and that target keeps moving as #517 lands: records became derivable at
// Lane B1 M3, same-file ADTs at M4. The vehicle is now an UNRESOLVED type name,
// which is the honest end of the line — B1 deliberately leaves imported and
// refined types underivable (deferred to B2 on the evaluator fuel budget), so
// unlike `Point` this one is not scheduled to become derivable underneath us.
// If a future milestone does derive it, this test fails loudly rather than
// silently ceasing to exercise the vacuous path — which is the whole point:
// the assertion below is the ONLY end-to-end pin on `ailang test` exiting 1 for
// a suite whose properties never ran.
const mixedVacuousTestSource = `module mixed_vacuous_test

export func anchor(x: int) -> int ! {}
ensures { result == x }
{
  x
}

export func shiftX(p: ImportedPoint, dx: int) -> int ! {}
ensures { result == dx }
{
  dx
}

export func headOr(xs: list[int], d: int) -> int ! {}
ensures { result == result }
{
  match xs { [] => d, ::(x, _) => x }
}

export func main() -> int ! {} { 0 }
`

func TestTestCommand_MixedVacuousSuiteExitSemantics(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "mixed_vacuous_test.ail")
	if err := os.WriteFile(file, []byte(mixedVacuousTestSource), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, exit := runAilangBin(t, bin, "test", "--format", "json", file)
	if exit != 1 {
		t.Fatalf("expected vacuous mixed suite to exit 1, got %d\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	var output struct {
		Success      bool `json:"success"`
		PassedTests  int  `json:"passed_tests"`
		VacuousSkips int  `json:"vacuous_skips"`
		Properties   []struct {
			Name     string `json:"name"`
			Status   string `json:"status"`
			TestsRun int    `json:"tests_run"`
			SkipKind string `json:"skip_kind"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("stdout is not pure JSON: %v\n%s", err, stdout)
	}
	if output.Success || output.PassedTests != 2 || output.VacuousSkips != 1 {
		t.Fatalf("unexpected mixed result: %+v", output)
	}
	if len(output.Properties) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(output.Properties))
	}
	if output.Properties[1].SkipKind != "no_generator" {
		t.Fatalf("shiftX skip kind = %q", output.Properties[1].SkipKind)
	}
	if output.Properties[2].Status != "pass" || output.Properties[2].TestsRun != 100 {
		t.Fatalf("headOr did not run 100 passing cases: %+v", output.Properties[2])
	}

	_, _, allowExit := runAilangBin(t, bin, "test", "--allow-skips", "--format", "json", file)
	if allowExit != 0 {
		t.Fatalf("--allow-skips expected exit 0, got %d", allowExit)
	}

	human, _, humanExit := runAilangBin(t, bin, "test", file)
	if humanExit != 1 {
		t.Fatalf("human mode expected exit 1, got %d", humanExit)
	}
	if !strings.Contains(human, "no generator for parameter p: ImportedPoint") {
		t.Fatalf("human output missing exact skip reason:\n%s", human)
	}
	if strings.Contains(human, "All tests passed!") {
		t.Fatalf("human output retained false-green headline:\n%s", human)
	}
}

func TestTestCommand_JSONStdoutSingleFile(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "json_output_test.ail")
	if err := os.WriteFile(file, []byte(jsonOutputTestSource), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, exit := runAilangBin(t, bin, "test", "--format", "json", file)
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exit, stderr)
	}
	var output map[string]any
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("stdout is not pure JSON: %v\nstdout:\n%s", err, stdout)
	}
	if !strings.Contains(stderr, "Running tests in") {
		t.Fatalf("expected preamble on stderr, got %q", stderr)
	}

	humanStdout, _, humanExit := runAilangBin(t, bin, "test", file)
	if humanExit != 0 {
		t.Fatalf("expected human mode exit 0, got %d", humanExit)
	}
	if !strings.Contains(humanStdout, "Running tests in") {
		t.Fatalf("human preamble moved off stdout: %q", humanStdout)
	}
}

func TestTestCommand_JSONStdoutPackage(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	manifest := `[package]
name = "test/json_output"
version = "0.1.0"
edition = "1"
`
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "json_output_test.ail"), []byte(jsonOutputTestSource), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, exit := runAilangBin(t, bin, "test", "--package", "--format", "json", dir)
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exit, stderr)
	}
	var output map[string]any
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("package stdout is not pure JSON: %v\nstdout:\n%s", err, stdout)
	}
	if !strings.Contains(stderr, "Package test/json_output") {
		t.Fatalf("expected package preamble on stderr, got %q", stderr)
	}
}
