// UpdateStageEvalAssessment has five error branches, but only four are reachable.
// EvalAssessment currently contains only string, bool, int, and int64 fields, so
// json.Marshal cannot fail. TestEvalAssessment_IsAlwaysMarshalable is a type guard
// for that construction invariant, not coverage of the marshal-error branch.
package firestore

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	obs "github.com/sunholo-data/ailang/internal/observatory"
)

func callUpdateStageEvalAssessment(store *ObservatoryStore, stageID string, assessment *obs.EvalAssessment) (err error, panicValue any) {
	defer func() {
		panicValue = recover()
	}()
	err = store.UpdateStageEvalAssessment(context.Background(), stageID, assessment)
	return err, nil
}

func TestUpdateStageEvalAssessment_RequiresStageID(t *testing.T) {
	err, panicValue := callUpdateStageEvalAssessment(&ObservatoryStore{}, "", &obs.EvalAssessment{})
	if panicValue != nil {
		t.Fatalf("missing stage ID reached Firestore client and panicked: %v", panicValue)
	}
	if err == nil || !strings.Contains(err.Error(), "stage_id is required") {
		t.Fatalf("missing stage ID error = %v, want message containing %q", err, "stage_id is required")
	}
}

func TestUpdateStageEvalAssessment_RequiresAssessment(t *testing.T) {
	err, panicValue := callUpdateStageEvalAssessment(&ObservatoryStore{}, "stage-1", nil)
	if panicValue != nil {
		t.Fatalf("missing assessment reached Firestore client and panicked: %v", panicValue)
	}
	if err == nil || !strings.Contains(err.Error(), "assessment is required") {
		t.Fatalf("missing assessment error = %v, want message containing %q", err, "assessment is required")
	}
}

func TestUpdateStageEvalAssessment_PositiveControlPassesBothGuards(t *testing.T) {
	err, panicValue := callUpdateStageEvalAssessment(&ObservatoryStore{}, "stage-1", &obs.EvalAssessment{})
	if err != nil && (strings.Contains(err.Error(), "stage_id is required") || strings.Contains(err.Error(), "assessment is required")) {
		t.Fatalf("valid arguments stopped at a validation guard: %v", err)
	}
	if err == nil && panicValue == nil {
		t.Fatal("valid arguments did not reach the nil Firestore client")
	}
}

func TestEvalAssessment_IsAlwaysMarshalable(t *testing.T) {
	assessment := obs.EvalAssessment{
		BenchmarkID: "bench", Model: "model", Language: "ailang", Condition: "control",
		EvalMode: "agent", Executor: "codex", Seed: 42,
		CompileOk: true, RuntimeOk: true, StdoutOk: true, ErrorCategory: "none",
		FirstAttemptOk: true, RepairUsed: true, RepairOk: true, ErrCode: "E_TEST",
		VerifyOk: true, VerifyVerified: 3, VerifyCounterex: 1, VerifySkipped: 2, VerifyErrors: 0,
		PromptVersion: "v1", CodeHash: "sha256", Code: "code", Stdout: "out",
		ExpectedStdout: "out", Stderr: "stderr",
	}
	if _, err := json.Marshal(assessment); err != nil {
		t.Fatalf("EvalAssessment ceased to be marshalable: %v", err)
	}
}
