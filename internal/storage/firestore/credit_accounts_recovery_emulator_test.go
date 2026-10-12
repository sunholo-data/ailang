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

func TestCreditRecoveryEmulatorConcurrentClientsAndBoundedRequests(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("requires local FIRESTORE_EMULATOR_HOST; never runs against live Firestore")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
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
	accountID := fmt.Sprintf("recovery-%d", now.UnixNano())
	one := creditbudget.New(NewCreditStore(first), func() time.Time { return now })
	two := creditbudget.New(NewCreditStore(second), func() time.Time { return now })
	if err := one.Configure(ctx, creditbudget.Policy{AccountID: accountID, Organization: "org", Workspace: "workspace", OperatingCeiling: 190 * creditbudget.USD, DailyCeiling: 6 * creditbudget.USD, TaskCeiling: 2 * creditbudget.USD, MaxTasks: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := one.Confirm(ctx, creditbudget.Confirmation{AccountID: accountID, GrantID: "grant", Amount: 200 * creditbudget.USD, Available: 200 * creditbudget.USD, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), Operator: "test", Evidence: "emulator"}); err != nil {
		t.Fatal(err)
	}
	if err := one.SetEnabled(ctx, accountID, true, "test", "emulator-only"); err != nil {
		t.Fatal(err)
	}
	if err := one.AdmitTask(ctx, creditbudget.Task{AccountID: accountID, ID: "t", AttemptID: "attempt", JobIdentity: "job", Models: []string{"claude-haiku-5-5"}, Ceiling: 2 * creditbudget.USD, LeaseUntil: now.Add(30 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err := one.Reserve(ctx, creditbudget.Reservation{AccountID: accountID, TaskID: "t", AttemptID: "attempt", JobIdentity: "job", Model: "claude-haiku-5-5", RequestID: "r", Amount: creditbudget.USD, PricingRevision: "verified", Deadline: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := one.MarkForwarded(ctx, accountID, "r"); err != nil {
		t.Fatal(err)
	}
	if err := one.MarkUnresolved(ctx, accountID, "r"); err != nil {
		t.Fatal(err)
	}
	if err := one.ReleaseTask(ctx, accountID, "t"); err != nil {
		t.Fatal(err)
	}
	if err := one.SetEnabled(ctx, accountID, false, "test", "stop"); err != nil {
		t.Fatal(err)
	}
	rows, err := two.Requests(ctx, accountID, "t")
	if err != nil || len(rows) != 1 || rows[0].RequestID != "r" || rows[0].State != "unresolved" {
		t.Fatalf("bounded metadata query=%+v err=%v", rows, err)
	}
	before, err := one.Status(ctx, accountID)
	if err != nil {
		t.Fatal(err)
	}
	d := creditbudget.ConservativeDebit{AccountID: accountID, GrantID: "grant", RequestID: "r", ExpectedAmount: creditbudget.USD, Operator: "test", Evidence: "no receipt"}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, engine := range []*creditbudget.Engine{one, two} {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- engine.ConservativeDebit(ctx, d) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	s, err := two.Status(ctx, accountID)
	if err != nil || s.Enabled || s.Settled != creditbudget.USD || s.ConservativeDebited != creditbudget.USD || s.Reserved != 0 || s.Unresolved != 0 || s.ReconciliationRequired || s.Available != before.Available || s.CanaryGatewaySettled != 0 {
		t.Fatalf("unsafe concurrent recovery=%+v err=%v", s, err)
	}
	rows, err = two.Requests(ctx, accountID, "t")
	if err != nil || len(rows) != 1 || rows[0].State != "conservative-debit" || rows[0].Actual != 0 || rows[0].UpstreamID != "" {
		t.Fatalf("fabricated receipt=%+v err=%v", rows, err)
	}
	d.Evidence = "different"
	if err := one.ConservativeDebit(ctx, d); !errors.Is(err, creditbudget.ErrConflict) {
		t.Fatalf("mutable audit accepted: %v", err)
	}
	ext := creditbudget.ExternalDebit{AccountID: accountID, GrantID: "grant", DebitID: "outside", Amount: 2 * creditbudget.USD, Kind: "conservative", Operator: "test", Evidence: "diagnostic"}
	for _, engine := range []*creditbudget.Engine{one, two} {
		if err := engine.ExternalDebit(ctx, ext); err != nil {
			t.Fatal(err)
		}
	}
	verified := ext
	verified.DebitID = "receipt"
	verified.Amount = 4
	verified.Kind = "provider-verified"
	verified.Evidence = "retained complete provider receipt"
	if err := two.ExternalDebit(ctx, verified); err != nil {
		t.Fatal(err)
	}
	s, err = one.Status(ctx, accountID)
	if err != nil || s.Settled != 3*creditbudget.USD+4 || s.ConservativeDebited != 3*creditbudget.USD || s.Day.Settled != 3*creditbudget.USD+4 || s.CanaryGatewaySettled != 0 {
		t.Fatalf("external debit duplicate/status=%+v err=%v", s, err)
	}
	if err := one.CompleteCanary(ctx, accountID, "test", "external-only"); err == nil {
		t.Fatal("external receipt qualified as gateway canary")
	}
	ext.DebitID = "beyond-smoke"
	ext.Amount = 3 * creditbudget.USD
	if err := two.ExternalDebit(ctx, ext); !errors.Is(err, creditbudget.ErrBlocked) {
		t.Fatalf("external canary cap exceeded: %v", err)
	}
	// Stored metadata queries fail loudly at 101, including the recovered original.
	err = NewCreditStore(first).Run(ctx, accountID, func(tx creditbudget.Transaction) error {
		for i := range 100 {
			id := fmt.Sprintf("metadata-%03d", i)
			if err := tx.Put("requests", id, creditbudget.Request{Reservation: creditbudget.Reservation{AccountID: accountID, TaskID: "t", RequestID: id}}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := two.Requests(ctx, accountID, "t"); err == nil {
		t.Fatal("truncated Firestore request query succeeded")
	}
}
