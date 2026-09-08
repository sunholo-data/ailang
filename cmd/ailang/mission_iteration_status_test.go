package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

func TestMissionIterationStatusReportsActualUsageOnly(t *testing.T) {
	ctx := context.Background()
	store, err := coordinator.NewSQLiteStore(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	snapshot := iteration.Snapshot{WorkspaceRoot: filepath.Join(t.TempDir(), "work"), Spec: iteration.Spec{Brief: "PRIVATE PROMPT"}}
	b, _ := json.Marshal(snapshot)
	key := coordinator.MissionWorkItemKey{MissionID: "docs", WorkItemID: "status"}
	item, err := store.ClaimMissionWorkItem(ctx, coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: string(b), SpecDigest: fmt.Sprintf("%x", sha256.Sum256(b)), StageIDs: []string{"write"}, TimeoutSeconds: 60}, 30)
	if err != nil {
		t.Fatal(err)
	}
	request := `{"fixture":"PRIVATE REQUEST"}`
	child, err := store.ClaimMissionChild(ctx, key, item.OwnerToken, coordinator.MissionAttemptSpec{MissionID: key.MissionID, WorkItemID: key.WorkItemID, StageID: "write", AttemptID: "a1", RequestJSON: request, RequestDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(request)))}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.StartMissionChild(ctx, key, item.OwnerToken, child.Key(), child.OwnerToken); err != nil {
		t.Fatal(err)
	}
	report := dispatch.Report{Attempts: []dispatch.Attempt{{Candidate: dispatch.Candidate{Model: "skipped"}, Status: "skipped"}, {Candidate: dispatch.Candidate{Model: "actual"}, Status: "completed"}}, Result: &executor.Result{InputTokens: 12, OutputTokens: 7, CostUSD: 0.02, CostProvenance: executor.CostMetered, Output: "PRIVATE OUTPUT"}}
	b, _ = json.Marshal(report)
	if err = store.CompleteMissionAttempt(ctx, child.Key(), child.OwnerToken, "execution_completed", string(b)); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = writeIterationStatus(ctx, &out, store, item, true); err != nil {
		t.Fatal(err)
	}
	var status iterationStatus
	if err = json.Unmarshal(out.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if len(status.Routes) != 1 || status.Routes[0].Model != "actual" || len(status.Stages) != 1 || status.Stages[0].Usage == nil || status.Stages[0].Usage.InputTokens != 12 || status.Stages[0].Usage.CostProvenance != executor.CostMetered || status.Stages[0].SelectedRoute.Model != "actual" || status.EvidenceDirectory == "" || status.ReceiptDirectory == "" {
		t.Fatalf("lost status evidence: %s", out.String())
	}
	if strings.Contains(out.String(), "PRIVATE") || strings.Contains(out.String(), item.OwnerToken) || strings.Contains(out.String(), child.OwnerToken) {
		t.Fatal("private execution data exposed")
	}
}
