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

// M-LAMBDA-OPEN-RECORD-PATTERN hardening (evaluator follow-up #2):
// order-independent open/closed unification.
//
// Two match arms scrutinize the SAME lambda parameter: one OPEN (`{a, ...}`),
// one CLOSED (`{a}`). The closed arm pins the parameter to EXACTLY {a}, so a
// caller passing a wider record `{a, b}` must be REJECTED regardless of which
// arm is written first. Before the fix, unifyOpenRecords bound the open side's
// extension row to an OPEN row even when the other (closed) arm's constraint was
// solved against the same TVar — so the closed constraint was silently weakened
// and open-first ACCEPTED the wide caller while closed-first REJECTED it
// (order-dependent static acceptance). This mirrors the evaluator's probes i/i2.
//
// Note: this is a STATIC-acceptance bug only; the runtime matcher is subset-based
// and already sound. These tests assert both arm orders now AGREE (both FAIL).

// TestLambdaOpenClosed_OpenFirst_WideCaller_Fail: open arm first, closed arm
// second, wide caller. Must be rejected (was the accepting/buggy order).
func TestLambdaOpenClosed_OpenFirst_WideCaller_Fail(t *testing.T) {
	src := pipeline.Source{
		Code:     `let f = \obj. match obj { {a, ...} => a, {a} => a } in f({a: "x", b: "y"})`,
		Filename: "",
	}
	_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err == nil {
		t.Fatalf("open-first arm order + wide caller must be REJECTED (a closed arm pins the param to exactly {a}); it type-checked — order-dependent acceptance not fixed")
	}
}

// TestLambdaOpenClosed_ClosedFirst_WideCaller_Fail: closed arm first, open arm
// second, wide caller. Must be rejected (already failed pre-fix). Pairing this
// with the open-first test proves the two arm orders now AGREE.
func TestLambdaOpenClosed_ClosedFirst_WideCaller_Fail(t *testing.T) {
	src := pipeline.Source{
		Code:     `let f = \obj. match obj { {a} => a, {a, ...} => a } in f({a: "x", b: "y"})`,
		Filename: "",
	}
	_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, src)
	if err == nil {
		t.Fatalf("closed-first arm order + wide caller must be REJECTED; it type-checked")
	}
}
