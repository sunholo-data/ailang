package coordinator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMissionWorkItemAbruptRecovery(t *testing.T) {
	if path := os.Getenv("AILANG_TEST_WORK_CRASH_DB"); path != "" {
		s, err := NewSQLiteStore(path)
		if err != nil {
			os.Exit(24)
		}
		ctx := context.Background()
		spec := workSpec()
		p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
		if err != nil {
			os.Exit(25)
		}
		a, err := s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, missionSpec(), 30)
		if err != nil {
			os.Exit(26)
		}
		phase := os.Getenv("AILANG_TEST_WORK_CRASH_PHASE")
		if phase != "prepared" {
			if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, p.OwnerToken, a.Key(), a.OwnerToken); err != nil {
				os.Exit(27)
			}
		}
		if phase == "completed" || phase == "accepted" {
			if err = s.CompleteMissionAttempt(ctx, a.Key(), a.OwnerToken, "execution_completed", `{}`); err != nil {
				os.Exit(28)
			}
		}
		if phase == "accepted" {
			a, err = s.GetMissionAttempt(ctx, a.Key())
			if err != nil {
				os.Exit(29)
			}
			receipt := MissionStageAcceptance{MissionAttemptKey: a.Key(), RequestDigest: a.RequestDigest, OutcomeDigest: a.OutcomeDigest, AcceptanceJSON: `{}`, AcceptanceDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(`{}`)))}
			if err = s.AcceptMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, receipt); err != nil {
				os.Exit(30)
			}
		}
		os.Exit(23)
	}
	for _, phase := range []string{"prepared", "running", "completed", "accepted"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "db")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMissionWorkItemAbruptRecovery$")
			cmd.Env = append(os.Environ(), "AILANG_TEST_WORK_CRASH_DB="+path, "AILANG_TEST_WORK_CRASH_PHASE="+phase)
			err := cmd.Run()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 23 {
				t.Fatalf("missed crash boundary: %v", err)
			}
			s := missionStore(t, path)
			spec := workSpec()
			p, err := s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.db.Exec("UPDATE mission_work_items SET lease_until=0"); err != nil {
				t.Fatal(err)
			}
			expireMission(t, s)
			next, err := s.ClaimMissionWorkItem(ctx, spec, 30)
			if err != nil {
				t.Fatal(err)
			}
			if next.Deadline != p.Deadline {
				t.Fatal("deadline reset")
			}
			if phase == "accepted" {
				if next.NextStage != 1 {
					t.Fatal("atomic acceptance pointer lost")
				}
				if _, err = s.GetMissionStageAcceptance(ctx, missionSpec().Key()); err != nil {
					t.Fatal(err)
				}
			}
			_, err = s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, next.OwnerToken, missionSpec(), 30)
			if (phase == "prepared") != (err == nil) {
				t.Fatalf("unsafe recovery phase %s: %v", phase, err)
			}
			if phase == "running" {
				if err = s.ReconcileMissionWorkItem(ctx, spec.MissionWorkItemKey); err != nil {
					t.Fatal(err)
				}
				p, _ = s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
				if p.State != "needs_reconciliation" {
					t.Fatalf("ambiguity lost: %+v", p)
				}
			}
		})
	}
}

func TestMissionCancellationReleaseRequiresConfirmedStop(t *testing.T) {
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
	if err = s.ConfirmMissionWorkItemStopped(ctx, spec.MissionWorkItemKey, p.Version-1); err == nil {
		t.Fatal("stale stop confirmation accepted")
	}
	if err = s.ConfirmMissionWorkItemStopped(ctx, spec.MissionWorkItemKey, p.Version); err != nil {
		t.Fatal(err)
	}
	spec.WorkItemID = "next"
	if _, err = s.ClaimMissionWorkItem(ctx, spec, 30); err != nil {
		t.Fatal(err)
	}
}
