package coordinator

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
)

func TestMissionAcceptanceDeadlineRollback(t *testing.T) {
	for _, which := range []string{"parent", "stage"} {
		t.Run(which, func(t *testing.T) {
			ctx := context.Background()
			s := missionStore(t, filepath.Join(t.TempDir(), "db"))
			spec := workSpec()
			p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.BeginMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, "s", 30); err != nil {
				t.Fatal(err)
			}
			a, err := s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, missionSpec(), 30)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, a.Key(), a.OwnerToken); err != nil {
				t.Fatal(err)
			}
			if err = s.CompleteMissionAttempt(ctx, a.Key(), a.OwnerToken, "execution_completed", `{}`); err != nil {
				t.Fatal(err)
			}
			a, err = s.GetMissionAttempt(ctx, a.Key())
			if err != nil {
				t.Fatal(err)
			}
			receipt := MissionStageAcceptance{MissionAttemptKey: a.Key(), RequestDigest: a.RequestDigest, OutcomeDigest: a.OutcomeDigest, AcceptanceJSON: `{}`, AcceptanceDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(`{}`)))}
			query := "UPDATE mission_work_items SET deadline=0"
			if which == "stage" {
				query = "UPDATE mission_stage_deadlines SET deadline=0"
			}
			if _, err = s.db.Exec(query); err != nil {
				t.Fatal(err)
			}
			if err = s.AcceptMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, receipt); err == nil {
				t.Fatal("accepted expired deadline")
			}
			if _, err = s.GetMissionStageAcceptance(ctx, a.Key()); err != sql.ErrNoRows {
				t.Fatalf("acceptance insert not rolled back: %v", err)
			}
			p, err = s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
			if err != nil {
				t.Fatal(err)
			}
			if p.NextStage != 0 {
				t.Fatal("expired acceptance advanced pointer")
			}
		})
	}
}

func TestMissionStageExpiryPreparedAndRunning(t *testing.T) {
	for _, running := range []bool{false, true} {
		t.Run(fmt.Sprint(running), func(t *testing.T) {
			ctx := context.Background()
			s := missionStore(t, filepath.Join(t.TempDir(), "db"))
			spec := workSpec()
			p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.BeginMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, "s", 30); err != nil {
				t.Fatal(err)
			}
			a, err := s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, missionSpec(), 30)
			if err != nil {
				t.Fatal(err)
			}
			if running {
				if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, a.Key(), a.OwnerToken); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.ExpireMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, "s"); err == nil {
				t.Fatal("stage expired before deadline")
			}
			if _, err = s.db.Exec("UPDATE mission_stage_deadlines SET deadline=0"); err != nil {
				t.Fatal(err)
			}
			if err = s.ExpireMissionStage(ctx, spec.MissionWorkItemKey, "stale", "s"); err == nil {
				t.Fatal("stale owner expired stage")
			}
			if err = s.ExpireMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, "judge"); err == nil {
				t.Fatal("wrong stage expired")
			}
			if err = s.ExpireMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, "s"); err != nil {
				t.Fatal(err)
			}
			got, err := s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
			if err != nil {
				t.Fatal(err)
			}
			expected := "failed"
			if running {
				expected = "needs_reconciliation"
			}
			if got.State != expected || got.ReasonCode != "deadline_exceeded" {
				t.Fatalf("bad expiry %+v", got)
			}
			if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, a.Key(), a.OwnerToken); err == nil {
				t.Fatal("expired child restarted")
			}
			next := spec
			next.WorkItemID = "next"
			_, err = s.ClaimMissionWorkItem(ctx, next, 30)
			if running == (err == nil) {
				t.Fatalf("admission wrong: %v", err)
			}
		})
	}
}
