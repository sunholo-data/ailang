package coordinator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func missionSpec() MissionAttemptSpec {
	return MissionAttemptSpec{MissionID: "m", WorkItemID: "w", StageID: "s", AttemptID: "a", RequestDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(`{"version":1}`))), RequestJSON: `{"version":1}`}
}
func missionStore(t *testing.T, path string) *SQLiteStore {
	t.Helper()
	s, e := NewSQLiteStore(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
func expireMission(t *testing.T, s *SQLiteStore) {
	t.Helper()
	if _, e := s.db.Exec("UPDATE mission_attempts SET lease_until=0"); e != nil {
		t.Fatal(e)
	}
}

func TestMissionClaimAcrossConnections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coordinator.db")
	a, b := missionStore(t, path), missionStore(t, path)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, s := range []*SQLiteStore{a, b} {
		wg.Add(1)
		go func(s *SQLiteStore) {
			defer wg.Done()
			_, e := s.ClaimMissionAttempt(context.Background(), missionSpec(), 30)
			results <- e
		}(s)
	}
	wg.Wait()
	close(results)
	wins := 0
	for e := range results {
		if e == nil {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("got %d owners", wins)
	}
}

func TestMissionRestartAndFencing(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "coordinator.db")
	a := missionStore(t, path)
	spec := missionSpec()
	first, e := a.ClaimMissionAttempt(ctx, spec, 30)
	if e != nil {
		t.Fatal(e)
	}
	expireMission(t, a)
	b := missionStore(t, path)
	second, e := b.ClaimMissionAttempt(ctx, spec, 30)
	if e != nil {
		t.Fatal(e)
	}
	if first.OwnerToken == second.OwnerToken {
		t.Fatal("reused fencing token")
	}
	if e := a.StartMissionAttempt(ctx, spec.Key(), first.OwnerToken); e == nil {
		t.Fatal("stale worker started")
	}
	if e := b.StartMissionAttempt(ctx, spec.Key(), second.OwnerToken); e != nil {
		t.Fatal(e)
	}
	expireMission(t, b)
	if _, e := a.ClaimMissionAttempt(ctx, spec, 30); e == nil {
		t.Fatal("expired running attempt rerun")
	}
	n, e := b.ReconcileMissionAttempts(ctx)
	if e != nil || n != 1 {
		t.Fatalf("reconcile %d %v", n, e)
	}
	got, e := a.GetMissionAttempt(ctx, spec.Key())
	if e != nil || got.State != "needs_reconciliation" {
		t.Fatalf("lost ambiguity: %+v %v", got, e)
	}
	if e := a.CompleteMissionAttempt(ctx, spec.Key(), second.OwnerToken, "execution_completed", `{"ok":true}`); e == nil {
		t.Fatal("stale completion overrode reconciliation")
	}
}

func TestMissionCompletionAndCancellation(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "cancel"}[cancel], func(t *testing.T) {
			ctx := context.Background()
			s := missionStore(t, filepath.Join(t.TempDir(), "db"))
			spec := missionSpec()
			a, e := s.ClaimMissionAttempt(ctx, spec, 30)
			if e != nil {
				t.Fatal(e)
			}
			if e := s.StartMissionAttempt(ctx, spec.Key(), a.OwnerToken); e != nil {
				t.Fatal(e)
			}
			current, e := s.GetMissionAttempt(ctx, spec.Key())
			if e != nil {
				t.Fatal(e)
			}
			if cancel {
				if e := s.CancelMissionAttempt(ctx, spec.Key(), current.Version-1); e == nil {
					t.Fatal("stale cancel accepted")
				}
				if e := s.CancelMissionAttempt(ctx, spec.Key(), current.Version); e != nil {
					t.Fatal(e)
				}
				if e := s.CompleteMissionAttempt(ctx, spec.Key(), a.OwnerToken, "execution_completed", `{}`); e == nil {
					t.Fatal("completion undid cancellation")
				}
			} else {
				for i := 0; i < 2; i++ {
					if e := s.CompleteMissionAttempt(ctx, spec.Key(), a.OwnerToken, "execution_completed", `{}`); e != nil {
						t.Fatal(e)
					}
				}
				if e := s.CompleteMissionAttempt(ctx, spec.Key(), a.OwnerToken, "execution_completed", `{"changed":true}`); e == nil {
					t.Fatal("conflicting duplicate accepted")
				}
				if e := s.CompleteMissionAttempt(ctx, spec.Key(), "stale", "execution_completed", `{}`); e == nil {
					t.Fatal("foreign completion accepted")
				}
			}
			spec.AttemptID = "another-receipt"
			if _, e := s.ClaimMissionAttempt(ctx, spec, 30); e == nil {
				t.Fatal("same stage bypassed by new attempt")
			}
		})
	}
}

func TestMissionRejectsFalseCompletionAndDigest(t *testing.T) {
	ctx := context.Background()
	s := missionStore(t, filepath.Join(t.TempDir(), "db"))
	spec := missionSpec()
	wrong := spec
	wrong.RequestJSON = `{"modified":true}`
	if _, e := s.ClaimMissionAttempt(ctx, wrong, 30); e == nil {
		t.Fatal("unbound request admitted")
	}
	a, e := s.ClaimMissionAttempt(ctx, spec, 30)
	if e != nil {
		t.Fatal(e)
	}
	if e := s.CompleteMissionAttempt(ctx, spec.Key(), a.OwnerToken, "execution_completed", `{}`); e == nil {
		t.Fatal("unstarted work reported complete")
	}
	if e := s.RenewMissionAttempt(ctx, spec.Key(), "wrong", 30); e == nil {
		t.Fatal("wrong owner renewed")
	}
	expireMission(t, s)
	if e := s.RenewMissionAttempt(ctx, spec.Key(), a.OwnerToken, 30); e == nil {
		t.Fatal("dead lease revived")
	}
}

// Abrupt subprocess termination proves persistence independently of Close/defers.
func TestMissionAbruptProcessRecovery(t *testing.T) {
	if path := os.Getenv("AILANG_TEST_MISSION_CRASH_DB"); path != "" {
		s, e := NewSQLiteStore(path)
		if e != nil {
			os.Exit(24)
		}
		a, e := s.ClaimMissionAttempt(context.Background(), missionSpec(), 30)
		if e != nil {
			os.Exit(25)
		}
		if os.Getenv("AILANG_TEST_MISSION_CRASH_PHASE") == "running" {
			if s.StartMissionAttempt(context.Background(), missionSpec().Key(), a.OwnerToken) != nil {
				os.Exit(26)
			}
		}
		os.Exit(23)
	}
	for _, phase := range []string{"prepared", "running"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "db")
			ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
			defer c()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMissionAbruptProcessRecovery$")
			cmd.Env = append(os.Environ(), "AILANG_TEST_MISSION_CRASH_DB="+path, "AILANG_TEST_MISSION_CRASH_PHASE="+phase)
			err := cmd.Run()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 23 {
				t.Fatalf("child did not reach crash boundary: %v", err)
			}
			s := missionStore(t, path)
			a, e := s.GetMissionAttempt(context.Background(), missionSpec().Key())
			if e != nil || a.State != phase {
				t.Fatalf("lost durable state: %+v %v", a, e)
			}
			expireMission(t, s)
			_, e = s.ClaimMissionAttempt(context.Background(), missionSpec(), 30)
			if (phase == "prepared") != (e == nil) {
				t.Fatalf("unsafe restart in %s: %v", phase, e)
			}
		})
	}
}
