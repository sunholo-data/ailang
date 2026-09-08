package iteration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
)

func TestIterationRejectsSubstitutedWorkspaceBeforeDispatch(t *testing.T) {
	for _, kind := range []string{"source_symlink", "ancestor_symlink", "foreign_clone", "borrowed_gitdir"} {
		t.Run(kind, func(t *testing.T) {
			s, spec, f := runtimeFixture(t)
			f.fail = true
			target := filepath.Join(s.WorkspaceRoot, spec.MissionID, spec.WorkItemID, "stages", "executor")
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "source_symlink":
				if err := os.Symlink(s.RepositoryPath, target); err != nil {
					t.Fatal(err)
				}
			case "ancestor_symlink":
				if err := os.Remove(filepath.Dir(target)); err != nil {
					t.Fatal(err)
				}
				other := t.TempDir()
				gitTest(t, s.RepositoryPath, "worktree", "add", "--detach", filepath.Join(other, "executor"), spec.BaseRevision)
				if err := os.Symlink(other, filepath.Dir(target)); err != nil {
					t.Fatal(err)
				}
			case "foreign_clone":
				gitTest(t, s.RepositoryPath, "clone", "--quiet", s.RepositoryPath, target)
			case "borrowed_gitdir":
				other := filepath.Join(t.TempDir(), "other")
				gitTest(t, s.RepositoryPath, "worktree", "add", "--detach", other, spec.BaseRevision)
				gitdir := gitTest(t, other, "rev-parse", "--absolute-git-dir")
				gitTest(t, s.RepositoryPath, "clone", "--quiet", s.RepositoryPath, target)
				if err := os.RemoveAll(filepath.Join(target, ".git")); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(target, ".git"), []byte("gitdir: "+gitdir+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			item, err := s.Run(context.Background(), spec)
			if err == nil {
				t.Fatalf("substituted workspace accepted: %+v", item)
			}
			if f.calls != 0 {
				t.Fatalf("dispatched %d times into substituted workspace", f.calls)
			}
		})
	}
}

type stageHealthExecutor struct {
	*iterationExecutor
	health int
}

func (f *stageHealthExecutor) GetExecutor(string) (executor.Executor, error) { return f, nil }
func (f *stageHealthExecutor) HealthCheck(context.Context) error {
	f.health++
	if f.health > 1 {
		return fmt.Errorf("fixture unavailable after preflight")
	}
	return nil
}

func TestIterationHealthChangeIsAvailabilityWait(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	s.Factory = &stageHealthExecutor{iterationExecutor: f}
	item, err := s.Run(context.Background(), spec)
	if err != nil || item.State != "waiting" || item.ReasonCode != "availability" || f.calls != 0 {
		t.Fatalf("health failure mislabeled: %+v err=%v calls=%d", item, err, f.calls)
	}
}

func TestIterationStageTimeoutIncludesInitialAdmission(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	spec.Stages[0].Limits.TimeoutSeconds = 2
	s.Admit = func(ctx context.Context, _ dispatch.Candidate) (dispatch.Admission, error) {
		<-ctx.Done()
		return dispatch.Admission{}, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	start := time.Now()
	_, err := s.Run(ctx, spec)
	if err == nil {
		t.Fatal("expired stage accepted")
	}
	if elapsed := time.Since(start); elapsed > 3500*time.Millisecond {
		t.Fatalf("initial admission escaped stage deadline: %v", elapsed)
	}
	if f.calls != 0 {
		t.Fatal("expired admission dispatched")
	}
}

func TestIterationRechecksWorkspaceAfterAdmission(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	f.fail = true
	target := filepath.Join(s.WorkspaceRoot, spec.MissionID, spec.WorkItemID, "stages", "executor")
	allow := s.Admit
	changed := false
	s.Admit = func(ctx context.Context, c dispatch.Candidate) (dispatch.Admission, error) {
		if !changed {
			changed = true
			if err := os.Rename(target, target+".parked"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(s.RepositoryPath, target); err != nil {
				t.Fatal(err)
			}
		}
		return allow(ctx, c)
	}
	_, err := s.Run(context.Background(), spec)
	if err == nil || f.calls != 0 {
		t.Fatalf("workspace replaced during admission dispatched: calls=%d err=%v", f.calls, err)
	}
}

func TestIterationStageDeadlineSurvivesQuotaResume(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	spec.Stages[0].Limits.TimeoutSeconds = 2
	allowed := s.Admit
	s.Admit = func(context.Context, dispatch.Candidate) (dispatch.Admission, error) {
		return dispatch.Admission{Policy: "fixture", ObservedAt: time.Now(), Reason: "quota blocked"}, nil
	}
	item, err := s.Run(context.Background(), spec)
	if err != nil || item.State != "waiting" {
		t.Fatalf("initial wait: %+v %v", item, err)
	}
	time.Sleep(2200 * time.Millisecond)
	s.Admit = allowed
	item, err = s.Run(context.Background(), spec)
	if err == nil {
		t.Fatalf("expired stage resumed: %+v", item)
	}
	if f.calls != 0 {
		t.Fatalf("expired stage dispatched %d times", f.calls)
	}
}

func TestIterationPreparedStageExpiryReleasesAdmission(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	spec.Stages[0].Limits.TimeoutSeconds = 2
	allowed := s.Admit
	checks := 0
	s.Admit = func(ctx context.Context, c dispatch.Candidate) (dispatch.Admission, error) {
		checks++
		if checks == 1 {
			return allowed(ctx, c)
		}
		return dispatch.Admission{Policy: "fixture", ObservedAt: time.Now(), Reason: "quota changed"}, nil
	}
	item, err := s.Run(context.Background(), spec)
	if err != nil || item.State != "waiting" {
		t.Fatalf("prepared wait: %+v %v", item, err)
	}
	time.Sleep(2200 * time.Millisecond)
	s.Admit = allowed
	item, err = s.Run(context.Background(), spec)
	if err == nil || item == nil || item.State != "failed" || f.calls != 0 {
		t.Fatalf("expired prepared stage not finalized: state=%+v err=%v calls=%d", item, err, f.calls)
	}
	// Terminal expiration must release admission for a reviewed successor.
	spec.WorkItemID = "successor"
	spec.Stages[0].Limits.TimeoutSeconds = 30
	item, err = s.Run(context.Background(), spec)
	if err != nil || item.State != "completed" {
		t.Fatalf("expired prepared child retained admission: %+v %v", item, err)
	}
}
