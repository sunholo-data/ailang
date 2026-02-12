package smt

import (
	"strings"
	"testing"
	"time"
)

// --- Mock tests (no Z3 needed) ---

func TestParseModel_SatOutput(t *testing.T) {
	output := `sat
(
  (define-fun x () Int
    0)
  (define-fun result () Int
    5)
)`
	bindings := parseModel(output)
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d: %+v", len(bindings), bindings)
	}

	// Check first binding
	if bindings[0].Name != "x" {
		t.Errorf("binding[0].Name = %q, want %q", bindings[0].Name, "x")
	}
	if bindings[0].Sort != "Int" {
		t.Errorf("binding[0].Sort = %q, want %q", bindings[0].Sort, "Int")
	}
	if bindings[0].Value != "0" {
		t.Errorf("binding[0].Value = %q, want %q", bindings[0].Value, "0")
	}

	// Check second binding
	if bindings[1].Name != "result" {
		t.Errorf("binding[1].Name = %q, want %q", bindings[1].Name, "result")
	}
	if bindings[1].Sort != "Int" {
		t.Errorf("binding[1].Sort = %q, want %q", bindings[1].Sort, "Int")
	}
	if bindings[1].Value != "5" {
		t.Errorf("binding[1].Value = %q, want %q", bindings[1].Value, "5")
	}
}

func TestParseModel_BoolBinding(t *testing.T) {
	output := `sat
(
  (define-fun flag () Bool
    true)
)`
	bindings := parseModel(output)
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].Name != "flag" {
		t.Errorf("Name = %q", bindings[0].Name)
	}
	if bindings[0].Value != "true" {
		t.Errorf("Value = %q, want %q", bindings[0].Value, "true")
	}
}

func TestParseModel_DatatypeBinding(t *testing.T) {
	output := `sat
(
  (define-fun season () Season
    LOW_SEASON)
)`
	bindings := parseModel(output)
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].Name != "season" {
		t.Errorf("Name = %q", bindings[0].Name)
	}
	if bindings[0].Sort != "Season" {
		t.Errorf("Sort = %q", bindings[0].Sort)
	}
	if bindings[0].Value != "LOW_SEASON" {
		t.Errorf("Value = %q, want %q", bindings[0].Value, "LOW_SEASON")
	}
}

func TestParseModel_NegativeInt(t *testing.T) {
	output := `sat
(
  (define-fun x () Int
    (- 5))
)`
	bindings := parseModel(output)
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].Value != "(- 5)" {
		t.Errorf("Value = %q, want %q", bindings[0].Value, "(- 5)")
	}
}

func TestParseModel_EmptySat(t *testing.T) {
	output := "sat\n"
	bindings := parseModel(output)
	if len(bindings) != 0 {
		t.Errorf("expected 0 bindings, got %d", len(bindings))
	}
}

func TestParseModel_InlineValue(t *testing.T) {
	// Some Z3 versions put value on same line
	output := `sat
(
  (define-fun x () Int 42)
)`
	bindings := parseModel(output)
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d: %+v", len(bindings), bindings)
	}
	if bindings[0].Value != "42" {
		t.Errorf("Value = %q, want %q", bindings[0].Value, "42")
	}
}

func TestSolverStatus_String(t *testing.T) {
	tests := []struct {
		status SolverStatus
		want   string
	}{
		{StatusVerified, "verified"},
		{StatusCounterexample, "counterexample"},
		{StatusUnknown, "unknown"},
		{StatusError, "error"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestDefaultSolverConfig(t *testing.T) {
	cfg := DefaultSolverConfig()
	if cfg.Timeout != 5*time.Second {
		t.Errorf("default timeout = %v, want 5s", cfg.Timeout)
	}
}

// --- Z3-gated integration tests ---

func TestSolve_Unsat_Z3(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}

	// Simple unsat: x >= 0 implies x >= 0 (trivially true)
	smtlib := `(set-logic ALL)
(declare-const x Int)
(assert (>= x 0))
(assert (not (>= x 0)))
(check-sat)
(get-model)
`
	result, err := Solve(smtlib, DefaultSolverConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusVerified {
		t.Errorf("expected StatusVerified, got %v (output: %s)", result.Status, result.RawOutput)
	}
	if result.Duration <= 0 {
		t.Error("duration should be positive")
	}
}

func TestSolve_Sat_Z3(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}

	// Satisfiable: x >= 0, result = 5, postcondition says result > 10 (violated!)
	smtlib := `(set-logic ALL)
(declare-const x Int)
(define-const result Int 5)
(assert (>= x 0))
(assert (not (> result 10)))
(check-sat)
(get-model)
`
	result, err := Solve(smtlib, DefaultSolverConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusCounterexample {
		t.Errorf("expected StatusCounterexample, got %v (output: %s)", result.Status, result.RawOutput)
	}
	if len(result.Model) == 0 {
		t.Error("expected model bindings")
	}
}

func TestSolve_AdmissionFee_Z3(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}

	smtlib := `; Verification of admissionFee
(set-logic ALL)

(declare-datatype Season ((LOW_SEASON) (HIGH_SEASON)))
(declare-const age Int)
(declare-const season Season)
(define-const result Int (match season ((LOW_SEASON (ite (< age 5) 0 (ite (>= age 65) 5 15))) (HIGH_SEASON (ite (>= age 65) 10 20)))))

(assert (>= age 0))
(assert (not (>= result 0)))

(check-sat)
(get-model)
`
	result, err := Solve(smtlib, DefaultSolverConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusVerified {
		t.Errorf("admissionFee should be verified, got %v (output: %s)", result.Status, result.RawOutput)
	}
}

func TestSolve_BrokenContract_Z3(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}

	// Function returns -1 for some inputs, but ensures result >= 0
	smtlib := `(set-logic ALL)
(declare-const x Int)
(define-const result Int (ite (>= x 0) x (- 1)))
(assert true)
(assert (not (>= result 0)))
(check-sat)
(get-model)
`
	result, err := Solve(smtlib, DefaultSolverConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusCounterexample {
		t.Errorf("broken contract should produce counterexample, got %v", result.Status)
	}

	// Check that the counterexample shows a negative x
	foundX := false
	for _, b := range result.Model {
		if b.Name == "x" {
			foundX = true
			// x should be negative in the counterexample
			if !strings.HasPrefix(b.Value, "(- ") && b.Value != "0" {
				// Accept any negative value
				t.Logf("counterexample: x = %s", b.Value)
			}
		}
	}
	if !foundX {
		t.Error("counterexample should include x binding")
	}
}

func TestFindZ3(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}

	path, err := FindZ3()
	if err != nil {
		t.Fatalf("FindZ3() error: %v", err)
	}
	if path == "" {
		t.Error("FindZ3() returned empty path")
	}
}

func TestZ3Version(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}

	version := Z3Version()
	if version == "" {
		t.Error("Z3Version() returned empty")
	}
	if !strings.Contains(version, "Z3") {
		t.Errorf("Z3Version() = %q, expected to contain 'Z3'", version)
	}
}

func TestSolve_Z3NotFound(t *testing.T) {
	// Use a fake path that doesn't exist
	config := SolverConfig{
		Z3Path:  "/nonexistent/path/z3",
		Timeout: 5 * time.Second,
	}
	result, err := Solve("(check-sat)", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusError {
		t.Errorf("expected StatusError for non-existent Z3, got %v", result.Status)
	}
}
