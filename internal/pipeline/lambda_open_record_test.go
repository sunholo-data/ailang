package pipeline_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// M-LAMBDA-OPEN-RECORD-PATTERN regression matrix.
//
// The bug: `\obj. match obj { {name, ...} => name }` inferred a CLOSED record
// parameter `{name: t}` instead of the OPEN `{name: t | r}` the user asked for
// with `...`, so a caller passing a record with extra fields was rejected — even
// though top-level `let ... in match` already worked. The fix must NOT reverse
// the (correct) strictness of a CLOSED `{name}` pattern.
//
// These are end-to-end (elaborate -> typecheck -> solve -> eval) via pipeline.Run,
// mirroring poly_arithmetic_test.go.

// TestLambdaOpenRecord_ExtraField_Pass is the primary bug: an OPEN pattern in a
// let-bound lambda must accept a caller with extra fields.
func TestLambdaOpenRecord_ExtraField_Pass(t *testing.T) {
	src := pipeline.Source{
		Code:     `let getName = \obj. match obj { {name, ...} => name } in getName({name: "Grace", id: 123})`,
		Filename: "",
	}
	result, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err != nil {
		t.Fatalf("expected open pattern + extra-field caller to type-check, got error: %v", err)
	}
	if got := result.Value.String(); got != "Grace" {
		t.Errorf("expected \"Grace\", got %s", got)
	}
}

// TestLambdaOpenRecord_MatchingShape_Pass: OPEN pattern, caller matches exactly.
func TestLambdaOpenRecord_MatchingShape_Pass(t *testing.T) {
	src := pipeline.Source{
		Code:     `let getName = \obj. match obj { {name, ...} => name } in getName({name: "Grace"})`,
		Filename: "",
	}
	result, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err != nil {
		t.Fatalf("expected open pattern + matching caller to type-check, got error: %v", err)
	}
	if got := result.Value.String(); got != "Grace" {
		t.Errorf("expected \"Grace\", got %s", got)
	}
}

// TestLambdaClosedRecord_MatchingShape_Pass: CLOSED pattern, caller matches exactly.
func TestLambdaClosedRecord_MatchingShape_Pass(t *testing.T) {
	src := pipeline.Source{
		Code:     `let getName = \obj. match obj { {name} => name } in getName({name: "Grace"})`,
		Filename: "",
	}
	result, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err != nil {
		t.Fatalf("expected closed pattern + matching caller to type-check, got error: %v", err)
	}
	if got := result.Value.String(); got != "Grace" {
		t.Errorf("expected \"Grace\", got %s", got)
	}
}

// TestLambdaClosedRecord_ExtraField_Fail is the soundness red-line: a CLOSED
// `{name}` pattern must STILL reject a caller passing extra fields. The M1 fix
// must not silently reverse this strictness.
func TestLambdaClosedRecord_ExtraField_Fail(t *testing.T) {
	src := pipeline.Source{
		Code:     `let getName = \obj. match obj { {name} => name } in getName({name: "Grace", id: 123})`,
		Filename: "",
	}
	_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err == nil {
		t.Fatalf("expected closed pattern + extra-field caller to be REJECTED, but it type-checked (strictness reversed)")
	}
}

// TestLambdaOpenRecord_IIFE_ExtraField_Pass guards M1 independently of any
// generalization: the immediately-applied lambda (no let, no scheme) must also
// accept extra fields.
func TestLambdaOpenRecord_IIFE_ExtraField_Pass(t *testing.T) {
	src := pipeline.Source{
		Code:     `(\obj. match obj { {name, ...} => name })({name: "Grace", id: 123})`,
		Filename: "",
	}
	result, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err != nil {
		t.Fatalf("expected open-pattern IIFE + extra-field caller to type-check, got error: %v", err)
	}
	if got := result.Value.String(); got != "Grace" {
		t.Errorf("expected \"Grace\", got %s", got)
	}
}

// TestLambdaOpenRecord_MissingField_Fail guards against over-generalization: an
// OPEN pattern still REQUIRES the fields it names, so a caller missing `name`
// must be rejected (the row variable absorbs EXTRA fields, not MISSING ones).
func TestLambdaOpenRecord_MissingField_Fail(t *testing.T) {
	src := pipeline.Source{
		Code:     `let getName = \obj. match obj { {name, ...} => name } in getName({id: 123})`,
		Filename: "",
	}
	_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err == nil {
		t.Fatalf("expected open pattern + caller missing `name` to be REJECTED, but it type-checked (over-generalized)")
	}
}

// TestLambdaOpenRecord_HintNotMisleading: after the fix the open reproducer
// produces NO error, so the "Use open record syntax" hint (which would tell the
// user to write `{name, ...}` they already wrote) must not appear.
func TestLambdaOpenRecord_HintNotMisleading(t *testing.T) {
	src := pipeline.Source{
		Code:     `let getName = \obj. match obj { {name, ...} => name } in getName({name: "Grace", id: 123})`,
		Filename: "",
	}
	_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err != nil {
		t.Fatalf("open reproducer should type-check with no error, got: %v", err)
	}
	// Belt-and-braces: the closed+extra case IS still an error and the hint
	// there is now genuinely correct (telling the user to add `...`). Assert the
	// hint fires on that path so we know the DX message wasn't accidentally lost.
	closedSrc := pipeline.Source{
		Code:     `let getName = \obj. match obj { {name} => name } in getName({name: "Grace", id: 123})`,
		Filename: "",
	}
	_, closedErr := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, closedSrc)
	if closedErr == nil {
		t.Fatalf("closed+extra should still be an error")
	}
	if !strings.Contains(closedErr.Error(), "open record syntax") {
		t.Errorf("closed+extra error should still carry the open-record hint (now genuinely helpful), got: %v", closedErr)
	}
}
