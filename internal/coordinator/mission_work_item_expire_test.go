package coordinator

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMissionWorkItemExpiry(t *testing.T) {
	for _, phase := range []string{"none", "prepared", "running", "execution_completed", "needs_reconciliation"} {
		t.Run(phase, func(t *testing.T) {
			ctx := context.Background()
			s := missionStore(t, filepath.Join(t.TempDir(), "db"))
			spec := workSpec()
			p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
			if err != nil {
				t.Fatal(err)
			}
			var child *MissionAttempt
			if phase != "none" {
				child, err = s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, missionSpec(), 30)
				if err != nil {
					t.Fatal(err)
				}
			}
			if phase != "none" && phase != "prepared" {
				if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, child.Key(), child.OwnerToken); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "execution_completed" {
				if err = s.CompleteMissionAttempt(ctx, child.Key(), child.OwnerToken, phase, `{}`); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "needs_reconciliation" {
				expireMission(t, s)
				if err = s.ReconcileMissionWorkItem(ctx, spec.MissionWorkItemKey); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.ExpireMissionWorkItem(ctx, spec.MissionWorkItemKey, p.OwnerToken); err == nil {
				t.Fatal("expired before deadline")
			}
			if _, err = s.db.Exec("UPDATE mission_work_items SET deadline=0,lease_until=0"); err != nil {
				t.Fatal(err)
			}
			if err = s.ExpireMissionWorkItem(ctx, spec.MissionWorkItemKey, "stale"); err == nil {
				t.Fatal("stale owner expired item")
			}
			if err = s.ExpireMissionWorkItem(ctx, spec.MissionWorkItemKey, p.OwnerToken); err != nil {
				t.Fatal(err)
			}
			got, err := s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
			if err != nil {
				t.Fatal(err)
			}
			ambiguous := phase == "running" || phase == "needs_reconciliation"
			expected := "failed"
			if ambiguous {
				expected = "needs_reconciliation"
			}
			if got.State != expected || got.ReasonCode != "deadline_exceeded" || got.OwnerToken != "" || got.LeaseUntil != 0 {
				t.Fatalf("expiry result %+v", got)
			}
			if child != nil {
				now, err := s.GetMissionAttempt(ctx, child.Key())
				if err != nil {
					t.Fatal(err)
				}
				if phase == "prepared" && now.State != "cancelled" {
					t.Fatal("prepared child remained retryable")
				}
				if ambiguous && now.State != "needs_reconciliation" {
					t.Fatal("running ambiguity lost")
				}
				if phase == "execution_completed" && now.State != phase {
					t.Fatal("terminal child outcome rewritten")
				}
			}
			if _, err = s.ClaimMissionWorkItem(ctx, spec, 30); err == nil && !ambiguous {
				t.Fatal("expired failed item restarted")
			}
			next := spec
			next.WorkItemID = "next"
			_, err = s.ClaimMissionWorkItem(ctx, next, 30)
			if ambiguous == (err == nil) {
				t.Fatalf("admission wrong for %s: %v", phase, err)
			}
		})
	}
}
