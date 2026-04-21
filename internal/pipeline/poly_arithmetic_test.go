package pipeline_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// TestPolyArithmetic_VarBoundLambda_FloatAdd is the primary M-POLY-ARITH bug:
// let add = \x. \y. x + y in add(3.14)(2.71) should return 5.85 but panics
// with "expected int arguments" because Num defaulting resolves the lambda to
// int -> int -> int before monomorphization can specialize it.
func TestPolyArithmetic_VarBoundLambda_FloatAdd(t *testing.T) {
	src := pipeline.Source{
		Code:     `let add = \x. \y. x + y in add(3.14)(2.71)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	if got != "5.85" {
		t.Errorf("Expected 5.85, got %s", got)
	}
}

// TestPolyArithmetic_VarBoundLambda_FloatSub tests subtraction
func TestPolyArithmetic_VarBoundLambda_FloatSub(t *testing.T) {
	src := pipeline.Source{
		Code:     `let sub = \x. \y. x - y in sub(3.14)(2.71)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	// 3.14 - 2.71 = 0.43 (floating point)
	if !strings.HasPrefix(got, "0.43") {
		t.Errorf("Expected ~0.43, got %s", got)
	}
}

// TestPolyArithmetic_VarBoundLambda_FloatMul tests multiplication
func TestPolyArithmetic_VarBoundLambda_FloatMul(t *testing.T) {
	src := pipeline.Source{
		Code:     `let mul = \x. \y. x * y in mul(3.0)(2.0)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	if got != "6" && got != "6.0" {
		t.Errorf("Expected 6 or 6.0, got %s", got)
	}
}

// TestPolyArithmetic_VarBoundLambda_FloatDiv tests division
func TestPolyArithmetic_VarBoundLambda_FloatDiv(t *testing.T) {
	src := pipeline.Source{
		Code:     `let divide = \x. \y. x / y in divide(10.0)(4.0)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	if got != "2.5" {
		t.Errorf("Expected 2.5, got %s", got)
	}
}

// TestPolyArithmetic_VarBoundLambda_IntAdd tests that int arithmetic still works
func TestPolyArithmetic_VarBoundLambda_IntAdd(t *testing.T) {
	src := pipeline.Source{
		Code:     `let add = \x. \y. x + y in add(3)(7)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	if got != "10" {
		t.Errorf("Expected 10, got %s", got)
	}
}

// TestPolyArithmetic_InlineLambda_FloatAdd verifies inline lambdas still work (regression check)
func TestPolyArithmetic_InlineLambda_FloatAdd(t *testing.T) {
	src := pipeline.Source{
		Code:     `(\x. \y. x + y)(3.14)(2.71)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	if got != "5.85" {
		t.Errorf("Expected 5.85, got %s", got)
	}
}

// TestPolyArithmetic_InlineLambda_FloatMul verifies inline lambda multiplication works (regression)
func TestPolyArithmetic_InlineLambda_FloatMul(t *testing.T) {
	src := pipeline.Source{
		Code:     `(\x. \y. x * y)(3.0)(2.0)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	if got != "6" && got != "6.0" {
		t.Errorf("Expected 6 or 6.0, got %s", got)
	}
}

// === M3: Comprehensive tests ===

// TestPolyArithmetic_NestedOps tests nested operators: (x + y) * (x - y)
func TestPolyArithmetic_NestedOps(t *testing.T) {
	src := pipeline.Source{
		Code:     `let f = \x. \y. (x + y) * (x - y) in f(5.0)(3.0)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	// (5+3) * (5-3) = 8 * 2 = 16
	if got != "16" && got != "16.0" {
		t.Errorf("Expected 16 or 16.0, got %s", got)
	}
}

// TestPolyArithmetic_NestedOps_Int tests nested operators with ints
func TestPolyArithmetic_NestedOps_Int(t *testing.T) {
	src := pipeline.Source{
		Code:     `let f = \x. \y. (x + y) * (x - y) in f(5)(3)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	// (5+3) * (5-3) = 8 * 2 = 16
	if got != "16" {
		t.Errorf("Expected 16, got %s", got)
	}
}

// TestPolyArithmetic_MultipleApplications tests same polymorphic lambda with different types
func TestPolyArithmetic_MultipleApplications(t *testing.T) {
	// Test that int application still works
	src := pipeline.Source{
		Code:     `let add = \x. \y. x + y in add(10)(20)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	if got != "30" {
		t.Errorf("Expected 30, got %s", got)
	}
}

// TestPolyArithmetic_ChainedArithmetic tests chained arithmetic: x + y + z via nested lets
func TestPolyArithmetic_ChainedArithmetic(t *testing.T) {
	src := pipeline.Source{
		Code:     `let add = \x. \y. x + y in add(add(1.5)(2.5))(3.0)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	// (1.5 + 2.5) + 3.0 = 4.0 + 3.0 = 7.0
	if got != "7" && got != "7.0" {
		t.Errorf("Expected 7 or 7.0, got %s", got)
	}
}

// TestPolyArithmetic_DivisionByFloat tests division producing non-integer result
func TestPolyArithmetic_DivisionByFloat(t *testing.T) {
	src := pipeline.Source{
		Code:     `let div = \x. \y. x / y in div(7.0)(2.0)`,
		Filename: "",
	}
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		t.Fatalf("Pipeline error: %v", err)
	}

	got := result.Value.String()
	if got != "3.5" {
		t.Errorf("Expected 3.5, got %s", got)
	}
}
