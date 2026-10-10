package firestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/creditbudget"
)

func TestCreditAuthorityEmulatorConcurrentClients(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("requires local FIRESTORE_EMULATOR_HOST; never runs against live Firestore")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	first, err := NewClientForProject(ctx, "ailang-credit-emulator")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := NewClientForProject(ctx, "ailang-credit-emulator")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	now := time.Now().UTC()
	accountID := fmt.Sprintf("test-%d", now.UnixNano())
	one := creditbudget.New(NewCreditStore(first), func() time.Time { return now })
	two := creditbudget.New(NewCreditStore(second), func() time.Time { return now })
	if err := one.Configure(ctx, creditbudget.Policy{AccountID: accountID, Organization: "org", Workspace: "workspace", Enabled: false, OperatingCeiling: creditbudget.USD, DailyCeiling: creditbudget.USD, TaskCeiling: creditbudget.USD, MaxTasks: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := one.Confirm(ctx, creditbudget.Confirmation{AccountID: accountID, GrantID: "g", Amount: creditbudget.USD, Available: creditbudget.USD, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), Operator: "test", Evidence: "emulator"}); err != nil {
		t.Fatal(err)
	}
	if err := one.SetEnabled(ctx, accountID, true, "test", "emulator-only"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	admissionErrors := make(chan error, 2)
	for i, engine := range []*creditbudget.Engine{one, two} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			admissionErrors <- engine.AdmitTask(ctx, creditbudget.Task{AccountID: accountID, ID: []string{"one", "two"}[i], AttemptID: "attempt", JobIdentity: "job", Models: []string{"claude-haiku-5-5"}, Ceiling: creditbudget.USD, LeaseUntil: now.Add(30 * time.Minute)})
		}()
	}
	wg.Wait()
	close(admissionErrors)
	for err := range admissionErrors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := two.AdmitTask(ctx, creditbudget.Task{AccountID: accountID, ID: "third", AttemptID: "attempt", JobIdentity: "job", Models: []string{"claude-haiku-5-5"}, Ceiling: creditbudget.USD, LeaseUntil: now.Add(30 * time.Minute)}); !errors.Is(err, creditbudget.ErrBlocked) {
		t.Fatalf("third concurrent task admitted: %v", err)
	}
	wins := make(chan string, 2)
	errorsSeen := make(chan error, 2)
	for i, engine := range []*creditbudget.Engine{one, two} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := []string{"one", "two"}[i]
			_, err := engine.Reserve(ctx, creditbudget.Reservation{AccountID: accountID, TaskID: id, AttemptID: "attempt", JobIdentity: "job", Model: "claude-haiku-5-5", RequestID: id, Amount: creditbudget.USD, PricingRevision: "verified", Deadline: now.Add(time.Minute)})
			if err == nil {
				wins <- id
			} else {
				errorsSeen <- err
			}
		}()
	}
	wg.Wait()
	close(wins)
	close(errorsSeen)
	for err := range errorsSeen {
		if !errors.Is(err, creditbudget.ErrBlocked) {
			t.Fatalf("unexpected transaction refusal: %v", err)
		}
	}
	if len(wins) != 1 {
		t.Fatalf("competing clients admitted %d, want 1", len(wins))
	}
	s, err := two.Status(ctx, accountID)
	if err != nil || s.Reserved != creditbudget.USD || s.Eligible {
		t.Fatalf("status=%+v err=%v", s, err)
	}
	winner := <-wins
	if err := one.MarkForwarded(ctx, accountID, winner); err != nil {
		t.Fatal(err)
	}
	fresh := creditbudget.Confirmation{AccountID: accountID, GrantID: "fresh", ExpectedGrantID: "g", Amount: creditbudget.USD, Available: creditbudget.USD, StartsAt: now, ExpiresAt: now.Add(2 * time.Hour), Operator: "test", Evidence: "fresh-emulator-only"}
	if renewed, err := two.Confirm(ctx, fresh); err != nil || renewed.Reserved != creditbudget.USD {
		t.Fatalf("renewal lost live exposure: %+v %v", renewed, err)
	}
	if err := two.Settle(ctx, accountID, winner, creditbudget.USD/2, "provider-request"); err != nil {
		t.Fatal(err)
	}
	if err := one.Settle(ctx, accountID, winner, creditbudget.USD/2, "provider-request"); err != nil {
		t.Fatal(err)
	}
	s, err = one.Status(ctx, accountID)
	if err != nil || s.Settled != creditbudget.USD/2 || s.Reserved != 0 {
		t.Fatalf("duplicate settlement status=%+v err=%v", s, err)
	}
	if renewed, err := one.Confirm(ctx, fresh); err != nil || renewed.Settled != creditbudget.USD/2 {
		t.Fatalf("reconfirm reset settled spend: %+v %v", renewed, err)
	}
}
