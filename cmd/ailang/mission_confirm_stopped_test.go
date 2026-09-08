package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
)

func TestMissionConfirmStoppedMissingNeverCreates(t *testing.T) {
	deps, db := iterationTestDeps(t)
	err := runMissionConfirmStopped(context.Background(), []string{"docs", "--work-item", "w", "--version", "1", "--attestation", "verified stopped"}, &bytes.Buffer{}, deps)
	if missionErrorExitCode(err) != 2 {
		t.Fatalf("wrong failure: %v", err)
	}
	if _, err = os.Stat(db); !os.IsNotExist(err) {
		t.Fatalf("created DB: %v", err)
	}
}
func TestIterationDiagnosticTypedOutcomes(t *testing.T) {
	item := &coordinator.MissionWorkItem{State: "failed", ReasonCode: "secret stderr says timeout"}
	for _, tc := range []struct {
		r    *executor.Result
		want string
	}{
		{&executor.Result{Error: "timeout cost_exhausted secret"}, "unknown"},
		{&executor.Result{ThrashKilledAt: 100216}, "budget"},
		{&executor.Result{CostKilledAt: 2.1}, "budget"},
		{&executor.Result{FinishReason: executor.FinishTimeout}, "deadline"},
		{&executor.Result{FinishReason: executor.FinishError}, "provider_failure"},
	} {
		d := diagnoseIteration(item, tc.r, false)
		if d.Category != tc.want || d.NextAction == "" {
			t.Fatalf("got %+v want %s", d, tc.want)
		}
	}
	for _, state := range []string{"ready", "running", "validating", "waiting", "failed", "cancelled", "needs_reconciliation", "completed", "unexpected"} {
		item.State = state
		d := diagnoseIteration(item, nil, false)
		if d.NextAction == "" {
			t.Fatalf("empty action: %s", state)
		}
	}
	item.State = "needs_reconciliation"
	item.ReasonCode = "outcome_unknown"
	if d := diagnoseIteration(item, nil, false); d.SuggestedCommand != "" {
		t.Fatalf("unsafe automatic action %+v", d)
	}
}

func TestMissionConfirmStoppedRetainsEvidence(t *testing.T) {
	ctx := context.Background()
	deps, db := iterationTestDeps(t)
	store, err := coordinator.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := coordinator.MissionWorkItemKey{MissionID: "docs", WorkItemID: "w"}
	b := []byte(`{}`)
	digest := fmt.Sprintf("%x", sha256.Sum256(b))
	p, err := store.ClaimMissionWorkItem(ctx, coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: string(b), SpecDigest: digest, StageIDs: []string{"review"}, TimeoutSeconds: 60}, 30)
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.ClaimMissionChild(ctx, key, p.OwnerToken, coordinator.MissionAttemptSpec{MissionID: "docs", WorkItemID: "w", StageID: "review", AttemptID: "review-1", RequestJSON: string(b), RequestDigest: digest}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.StartMissionChild(ctx, key, p.OwnerToken, a.Key(), a.OwnerToken); err != nil {
		t.Fatal(err)
	}
	p, err = store.GetMissionWorkItem(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.CancelMissionWorkItem(ctx, key, p.Version); err != nil {
		t.Fatal(err)
	}
	p, err = store.GetMissionWorkItem(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"docs", "--work-item", "w", "--version", fmt.Sprint(p.Version), "--attestation", "operator verified review-1 and descendants absent"}
	var out bytes.Buffer
	if err = runMissionConfirmStopped(ctx, args, &out, deps); err != nil {
		t.Fatal(err)
	}
	evidence, err := store.GetMissionStopAttestation(ctx, key, p.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Children) != 1 || evidence.Children[0].AttemptID != "review-1" {
		t.Fatalf("missing evidence: %+v", evidence)
	}
	if strings.Contains(out.String(), "operator verified") {
		t.Fatal("status prints freeform attestation")
	}
	if err = runMissionConfirmStopped(ctx, args, &bytes.Buffer{}, deps); missionErrorExitCode(err) != 3 {
		t.Fatalf("stale confirmation not rejected: %v", err)
	}
}

func TestIterationStatusBudgetProjectionFromPersistedReport(t *testing.T) {
	ctx := context.Background()
	deps, db := iterationTestDeps(t)
	store, err := coordinator.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := coordinator.MissionWorkItemKey{MissionID: "docs", WorkItemID: "budget"}
	b := []byte(`{}`)
	digest := fmt.Sprintf("%x", sha256.Sum256(b))
	p, err := store.ClaimMissionWorkItem(ctx, coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: string(b), SpecDigest: digest, StageIDs: []string{"review"}, TimeoutSeconds: 60}, 30)
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.ClaimMissionChild(ctx, key, p.OwnerToken, coordinator.MissionAttemptSpec{MissionID: "docs", WorkItemID: "budget", StageID: "review", AttemptID: "review-1", RequestJSON: string(b), RequestDigest: digest}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.StartMissionChild(ctx, key, p.OwnerToken, a.Key(), a.OwnerToken); err != nil {
		t.Fatal(err)
	}
	report := dispatch.Report{Result: &executor.Result{ThrashKilledAt: 100216, Error: "SECRET STDERR", Transcript: "SECRET TRANSCRIPT"}, Progress: &dispatch.Progress{ToolCalls: 34, RepeatedCalls: 12}}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.CompleteMissionAttempt(ctx, a.Key(), a.OwnerToken, "execution_failed", string(encoded)); err != nil {
		t.Fatal(err)
	}
	if err = store.FinishMissionWorkItem(ctx, key, p.OwnerToken, "failed", "stage execution failed; review retained artifacts"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = runMissionIteration(ctx, "status", []string{"docs", "--work-item", "budget", "--json"}, &out, deps); err != nil {
		t.Fatal(err)
	}
	var status iterationStatus
	if err = json.Unmarshal(out.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Diagnostic.Category != "budget" || status.NextAction == "" || len(status.Stages) != 1 || status.Stages[0].Progress.RepeatedCalls != 12 {
		t.Fatalf("missing projection: %s", out.String())
	}
	if strings.Contains(out.String(), "SECRET") {
		t.Fatal("status leaked provider text")
	}
}
