package coordinator

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMissionStageDeadlineSurvivesQuotaResume(t *testing.T) {
	ctx := context.Background()
	s := missionStore(t, filepath.Join(t.TempDir(), "db"))
	spec := workSpec()
	p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	deadline, err := s.BeginMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, "s", 30)
	if err != nil {
		t.Fatal(err)
	}
	if deadline >= p.Deadline || deadline <= p.Deadline-120 {
		t.Fatal("stage deadline not bounded")
	}
	// Move the persisted clock boundary backward to model elapsed time without sleeps.
	deadline -= 5
	if _, err = s.db.Exec("UPDATE mission_stage_deadlines SET deadline=?", deadline); err != nil {
		t.Fatal(err)
	}
	if err = s.WaitMissionWorkItem(ctx, spec.MissionWorkItemKey, p.OwnerToken, "quota_wait", "resume"); err != nil {
		t.Fatal(err)
	}
	next, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.BeginMissionStage(ctx, spec.MissionWorkItemKey, next.OwnerToken, "s", 30)
	if err != nil || again != deadline {
		t.Fatalf("deadline changed: %d %d %v", deadline, again, err)
	}
	if _, err = s.BeginMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, "s", 30); err == nil {
		t.Fatal("stale parent accessed stage")
	}
	if _, err = s.BeginMissionStage(ctx, spec.MissionWorkItemKey, next.OwnerToken, "s", 60); err == nil {
		t.Fatal("changed timeout admitted")
	}
	if _, err = s.BeginMissionStage(ctx, spec.MissionWorkItemKey, next.OwnerToken, "judge", 30); err == nil {
		t.Fatal("future stage admitted")
	}
}
func TestMissionStageDeadlineBoundedByParent(t *testing.T) {
	ctx := context.Background()
	s := missionStore(t, filepath.Join(t.TempDir(), "db"))
	spec := workSpec()
	p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	deadline, err := s.BeginMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, "s", 1800)
	if err != nil || deadline != p.Deadline {
		t.Fatalf("parent deadline not respected: %d %v", deadline, err)
	}
}

func TestMissionExpiredStageCannotStartPreparedChild(t *testing.T) {
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
	if _, err = s.db.Exec("UPDATE mission_stage_deadlines SET deadline=0"); err != nil {
		t.Fatal(err)
	}
	if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, a.Key(), a.OwnerToken); err == nil {
		t.Fatal("expired stage started")
	}
	a, err = s.GetMissionAttempt(ctx, a.Key())
	if err != nil {
		t.Fatal(err)
	}
	if a.State != "prepared" {
		t.Fatal("failed start did not roll back child transition")
	}
}
