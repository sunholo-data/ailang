package coordinator

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMissionStopAttestationAtomicFence(t *testing.T) {
	ctx := context.Background()
	s := missionStore(t, filepath.Join(t.TempDir(), "db"))
	spec := workSpec()
	p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, missionSpec(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, a.Key(), a.OwnerToken); err != nil {
		t.Fatal(err)
	}
	p, _ = s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
	if err = s.CancelMissionWorkItem(ctx, spec.MissionWorkItemKey, p.Version); err != nil {
		t.Fatal(err)
	}
	p, _ = s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
	for _, tc := range []struct {
		version int64
		text    string
	}{{p.Version - 1, "verified descendants stopped"}, {p.Version, " "}} {
		if err = s.ConfirmMissionWorkItemStoppedAttested(ctx, spec.MissionWorkItemKey, tc.version, tc.text); err == nil {
			t.Fatal("invalid attestation accepted")
		}
	}
	if err = s.ConfirmMissionWorkItemStoppedAttested(ctx, spec.MissionWorkItemKey, p.Version, "verified descendants stopped"); err != nil {
		t.Fatal(err)
	}
	evidence, err := s.GetMissionStopAttestation(ctx, spec.MissionWorkItemKey, p.Version)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Attestation != "verified descendants stopped" || len(evidence.Children) != 1 || evidence.Children[0].AttemptID != a.AttemptID || evidence.Children[0].Version <= a.Version {
		t.Fatalf("missing exact evidence: %+v", evidence)
	}
	if err = s.ConfirmMissionWorkItemStoppedAttested(ctx, spec.MissionWorkItemKey, p.Version, "duplicate"); err == nil {
		t.Fatal("repeat accepted")
	}
}

func TestMissionStopAttestationRejectsUnknownOutcome(t *testing.T) {
	ctx := context.Background()
	s := missionStore(t, filepath.Join(t.TempDir(), "db"))
	spec := workSpec()
	p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("UPDATE mission_work_items SET state='needs_reconciliation',reason_code='outcome_unknown'"); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfirmMissionWorkItemStoppedAttested(ctx, spec.MissionWorkItemKey, p.Version, "process stopped"); err == nil {
		t.Fatal("generic ambiguity cleared")
	}
	p, _ = s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
	if p.State != "needs_reconciliation" {
		t.Fatal("state mutated")
	}
}
