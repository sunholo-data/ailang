package coordinator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func workSpec() MissionWorkItemSpec {
	return MissionWorkItemSpec{MissionWorkItemKey: MissionWorkItemKey{"m", "w"}, SpecJSON: `{"version":1}`, SpecDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(`{"version":1}`))), StageIDs: []string{"s", "judge"}, TimeoutSeconds: 120}
}
func TestMissionWorkAdmissionRace(t *testing.T) {
	for _, different := range []bool{false, true} {
		t.Run(fmt.Sprint(different), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "db")
			stores := []*SQLiteStore{missionStore(t, path), missionStore(t, path)}
			var wg sync.WaitGroup
			results := make(chan error, 2)
			for i, s := range stores {
				wg.Add(1)
				go func(i int, s *SQLiteStore) {
					defer wg.Done()
					spec := workSpec()
					if different && i == 1 {
						spec.WorkItemID = "other"
					}
					_, err := s.ClaimMissionWorkItem(context.Background(), spec, 30)
					results <- err
				}(i, s)
			}
			wg.Wait()
			close(results)
			wins := 0
			for err := range results {
				if err == nil {
					wins++
				}
			}
			if wins != 1 {
				t.Fatalf("admissions=%d", wins)
			}
		})
	}
}
func TestMissionParentFenceAndPreparedResume(t *testing.T) {
	ctx := context.Background()
	s := missionStore(t, filepath.Join(t.TempDir(), "db"))
	spec := workSpec()
	parent, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, parent.OwnerToken, missionSpec(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ReleaseMissionPreparedChild(ctx, spec.MissionWorkItemKey, parent.OwnerToken, child.Key(), child.OwnerToken, "quota_wait", "resume"); err != nil {
		t.Fatal(err)
	}
	second, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, parent.OwnerToken, child.Key(), child.OwnerToken); err == nil {
		t.Fatal("old parent started child")
	}
	again, err := s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, second.OwnerToken, missionSpec(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if again.AttemptID != child.AttemptID || again.OwnerToken == child.OwnerToken || again.Version <= child.Version {
		t.Fatal("prepared reclaim lost immutable identity or generation")
	}
	if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, second.OwnerToken, again.Key(), again.OwnerToken); err != nil {
		t.Fatal(err)
	}
	current, _ := s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
	if err = s.CancelMissionWorkItem(ctx, spec.MissionWorkItemKey, current.Version); err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteMissionAttempt(ctx, again.Key(), again.OwnerToken, "execution_completed", `{}`); err == nil {
		t.Fatal("cancelled child completed")
	}
	current, _ = s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
	if current.State != "needs_reconciliation" {
		t.Fatalf("lost running process ambiguity: %+v", current)
	}
	other := spec
	other.WorkItemID = "other"
	if _, err = s.ClaimMissionWorkItem(ctx, other, 30); err == nil {
		t.Fatal("admission released while external process ambiguous")
	}
}
func TestMissionAcceptanceAtomicAndImmutable(t *testing.T) {
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
	if err = s.CompleteMissionAttempt(ctx, a.Key(), a.OwnerToken, "execution_completed", `{}`); err != nil {
		t.Fatal(err)
	}
	a, _ = s.GetMissionAttempt(ctx, a.Key())
	acceptance := MissionStageAcceptance{MissionAttemptKey: a.Key(), RequestDigest: a.RequestDigest, OutcomeDigest: a.OutcomeDigest, AcceptanceJSON: `{"accepted":true}`, AcceptanceDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(`{"accepted":true}`)))}
	for i := 0; i < 2; i++ {
		if err = s.AcceptMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, acceptance); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
	if got.NextStage != 1 {
		t.Fatalf("stage pointer=%d", got.NextStage)
	}
	acceptance.OutcomeDigest = "changed"
	if err = s.AcceptMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, acceptance); err == nil {
		t.Fatal("mutated acceptance admitted")
	}
	if err = s.FinishMissionWorkItem(ctx, spec.MissionWorkItemKey, p.OwnerToken, "completed", ""); err == nil {
		t.Fatal("unfinished stages completed")
	}
}

func TestMissionParentExpiryCannotStealPreparedChild(t *testing.T) {
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
	if _, err = s.db.Exec("UPDATE mission_work_items SET lease_until=0"); err != nil {
		t.Fatal(err)
	}
	next, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	if next.Deadline != p.Deadline {
		t.Fatal("resume reset deadline")
	}
	if err = s.StartMissionChild(ctx, spec.MissionWorkItemKey, next.OwnerToken, a.Key(), a.OwnerToken); err == nil {
		t.Fatal("new parent stole live prepared generation")
	}
	if _, err = s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, next.OwnerToken, missionSpec(), 30); err == nil {
		t.Fatal("new parent reclaimed live child")
	}
	if err = s.StartMissionAttempt(ctx, a.Key(), a.OwnerToken); err == nil {
		t.Fatal("standalone start bypassed parent")
	}
	if _, err = s.ClaimMissionAttempt(ctx, missionSpec(), 30); err == nil {
		t.Fatal("standalone claim bypassed parent")
	}
	expireMission(t, s)
	if _, err = s.ClaimMissionChild(ctx, spec.MissionWorkItemKey, next.OwnerToken, missionSpec(), 30); err != nil {
		t.Fatal(err)
	}
}
func TestMissionParentExpiryFencesAcceptanceAndChangedSnapshot(t *testing.T) {
	ctx := context.Background()
	s := missionStore(t, filepath.Join(t.TempDir(), "db"))
	spec := workSpec()
	p, err := s.ClaimMissionWorkItem(ctx, spec, 30)
	if err != nil {
		t.Fatal(err)
	}
	before := p.Version
	if err = s.RenewMissionWorkItem(ctx, spec.MissionWorkItemKey, p.OwnerToken, 30); err != nil {
		t.Fatal(err)
	}
	p, _ = s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
	if p.Version != before {
		t.Fatal("renew changed cancellation version")
	}
	if _, err = s.db.Exec("UPDATE mission_work_items SET lease_until=0"); err != nil {
		t.Fatal(err)
	}
	altered := spec
	altered.SpecJSON = `{"version":2}`
	altered.SpecDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(altered.SpecJSON)))
	if _, err = s.ClaimMissionWorkItem(ctx, altered, 30); err == nil {
		t.Fatal("changed snapshot resumed")
	}
	if err = s.MarkMissionWorkItemValidating(ctx, spec.MissionWorkItemKey, p.OwnerToken); err == nil {
		t.Fatal("expired parent mutated phase")
	}
	a := MissionStageAcceptance{MissionAttemptKey: missionSpec().Key(), AcceptanceJSON: `{}`, AcceptanceDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(`{}`)))}
	if err = s.AcceptMissionStage(ctx, spec.MissionWorkItemKey, p.OwnerToken, a); err == nil {
		t.Fatal("expired parent accepted")
	}
}

func TestMissionReadOnlyStoreDoesNotCreateOrWrite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime db.sqlite")
	if _, err := OpenMissionReadOnlyStore(path); err == nil {
		t.Fatal("opened missing database")
	}
	s := missionStore(t, path)
	spec := workSpec()
	if _, err := s.ClaimMissionWorkItem(ctx, spec, 30); err != nil {
		t.Fatal(err)
	}
	ro, err := OpenMissionReadOnlyStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if _, err = ro.GetMissionWorkItem(ctx, spec.MissionWorkItemKey); err != nil {
		t.Fatal(err)
	}
	if _, err = ro.db.Exec("UPDATE mission_work_items SET lease_until=0"); err == nil {
		t.Fatal("readonly connection wrote")
	}
}
