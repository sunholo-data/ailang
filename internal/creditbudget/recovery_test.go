package creditbudget

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func unresolvedRecovery(t *testing.T) (*Engine, time.Time, ConservativeDebit) {
	t.Helper()
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "t")
	if _, err := e.Reserve(ctx, reservation(now, "r", "t", USD)); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkForwarded(ctx, "a", "r"); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkUnresolved(ctx, "a", "r"); err != nil {
		t.Fatal(err)
	}
	if err := e.ReleaseTask(ctx, "a", "t"); err != nil {
		t.Fatal(err)
	}
	if err := e.SetEnabled(ctx, "a", false, "operator", "stop"); err != nil {
		t.Fatal(err)
	}
	return e, now, ConservativeDebit{AccountID: "a", GrantID: "g1", RequestID: "r", ExpectedAmount: USD, Operator: "operator", Evidence: "no retained receipt"}
}

func TestRecoveryMovesFullExposureWithoutInventingProviderSpend(t *testing.T) {
	e, _, d := unresolvedRecovery(t)
	ctx := context.Background()
	before, _ := e.Status(ctx, "a")
	if err := e.ConservativeDebit(ctx, d); err != nil {
		t.Fatal(err)
	}
	s, _ := e.Status(ctx, "a")
	if s.Enabled || s.Reserved != 0 || s.Unresolved != 0 || s.Settled != USD || s.ConservativeDebited != USD || s.ReconciliationRequired || s.Available != before.Available || s.CanaryReserved != 0 || s.CanarySettled != USD || s.CanaryGatewaySettled != 0 {
		t.Fatalf("unsafe recovery: %+v", s)
	}
	rows, err := e.Requests(ctx, "a", "t")
	if err != nil || len(rows) != 1 {
		t.Fatalf("requests=%+v err=%v", rows, err)
	}
	if rows[0].State != "conservative-debit" || rows[0].Actual != 0 || rows[0].UpstreamID != "" {
		t.Fatalf("invented provider receipt: %+v", rows[0])
	}
	if err = e.CompleteCanary(ctx, "a", "operator", "conservative only"); err == nil {
		t.Fatal("conservative recovery qualified as verified gateway canary")
	}
	if err = e.ConservativeDebit(ctx, d); err != nil {
		t.Fatal(err)
	}
	d.Evidence = "changed"
	if err = e.ConservativeDebit(ctx, d); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed recovery accepted: %v", err)
	}
	if err = e.Settle(ctx, "a", "r", 0, "invented"); err == nil {
		t.Fatal("settlement refunded conservative debit")
	}
	if err = e.ReleaseUnsent(ctx, "a", "r"); err == nil {
		t.Fatal("release refunded conservative debit")
	}
}

func TestExternalDebitsAreImmutableAndCannotQualifyCanary(t *testing.T) {
	e, _ := fixture(t)
	ctx := context.Background()
	_ = e.SetEnabled(ctx, "a", false, "operator", "stop")
	d := ExternalDebit{AccountID: "a", GrantID: "g1", DebitID: "diagnostic", Amount: USD, Kind: "conservative", Operator: "operator", Evidence: "timeout"}
	for range 2 {
		if err := e.ExternalDebit(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	verified := d
	verified.DebitID = "receipt"
	verified.Amount = 4
	verified.Kind = "provider-verified"
	verified.Evidence = "retained receipt"
	if err := e.ExternalDebit(ctx, verified); err != nil {
		t.Fatal(err)
	}
	s, _ := e.Status(ctx, "a")
	if s.Settled != USD+4 || s.ConservativeDebited != USD || s.Day.Settled != USD+4 || s.CanarySettled != USD+4 || s.CanaryGatewaySettled != 0 || s.Enabled {
		t.Fatalf("unsafe external debit: %+v", s)
	}
	if err := e.CompleteCanary(ctx, "a", "operator", "verified external"); err == nil {
		t.Fatal("external receipt qualified as gateway canary")
	}
	d.Operator = "different"
	if err := e.ExternalDebit(ctx, d); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed actor replay: %v", err)
	}
}

func TestRecoveryGuardsAreAtomic(t *testing.T) {
	for _, kind := range []string{"enabled", "stale-grant", "wrong-amount", "active-lease", "forwarding", "unrelated-blocker", "over-reservation"} {
		t.Run(kind, func(t *testing.T) {
			e, now, d := unresolvedRecovery(t)
			ctx := context.Background()
			switch kind {
			case "enabled":
				_ = e.SetEnabled(ctx, "a", true, "operator", "enable")
			case "stale-grant":
				d.GrantID = "wrong"
			case "wrong-amount":
				d.ExpectedAmount = USD / 2
			case "active-lease", "forwarding", "unrelated-blocker", "over-reservation":
				_ = e.store.Run(ctx, "a", func(tx Transaction) error {
					a, err := getAccount(tx)
					if err != nil {
						return err
					}
					switch kind {
					case "active-lease":
						a.Leases["other"] = now.Add(time.Hour)
					case "forwarding":
						a.Forwarding = 1
					case "unrelated-blocker":
						a.ReconciliationReason = "billing mismatch"
					case "over-reservation":
						var r Request
						_, err := tx.Get("requests", "r", &r)
						if err != nil {
							return err
						}
						r.Actual = 2 * USD
						if err := tx.Put("requests", "r", r); err != nil {
							return err
						}
					}
					return tx.Put("account", "current", a)
				})
			}
			before, _ := e.Status(ctx, "a")
			if err := e.ConservativeDebit(ctx, d); err == nil {
				t.Fatal("unsafe recovery accepted")
			}
			after, _ := e.Status(ctx, "a")
			if before.Settled != after.Settled || before.Reserved != after.Reserved || before.Unresolved != after.Unresolved || after.ConservativeDebited != 0 {
				t.Fatalf("failed recovery mutated accounting: before=%+v after=%+v", before, after)
			}
		})
	}
}

func TestRequestsBoundFailsLoudly(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	_ = e.store.Run(ctx, "a", func(tx Transaction) error {
		for i := range 101 {
			id := string(rune('A' + i))
			if err := tx.Put("requests", id, Request{Reservation: Reservation{AccountID: "a", TaskID: "t", RequestID: id}, CreatedAt: now}); err != nil {
				return err
			}
		}
		return nil
	})
	if _, err := e.Requests(ctx, "a", "t"); err == nil {
		t.Fatal("truncated request inspection returned success")
	}
	if _, err := e.Requests(ctx, "a", "bad/task"); err == nil {
		t.Fatal("invalid task ID accepted")
	}
	if _, err := e.Requests(ctx, "missing", "t"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing account returned empty success: %v", err)
	}
}

func TestRecoveryConcurrentIdenticalDebitsAndGrantReset(t *testing.T) {
	e, now, d := unresolvedRecovery(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- e.ConservativeDebit(ctx, d) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	s, _ := e.Status(ctx, "a")
	if s.Settled != USD || s.ConservativeDebited != USD {
		t.Fatalf("duplicate debit charged: %+v", s)
	}
	c := Confirmation{AccountID: "a", GrantID: "g2", ExpectedGrantID: "g1", Amount: 200 * USD, Available: 200 * USD, StartsAt: now, ExpiresAt: now.Add(48 * time.Hour), Operator: "operator", Evidence: "fresh"}
	s, err := e.Confirm(ctx, c)
	if err != nil || s.Settled != 0 || s.ConservativeDebited != 0 || s.CanarySettled != USD || s.CanaryGatewaySettled != 0 {
		t.Fatalf("renewal subset mismatch: %+v %v", s, err)
	}
	if err := e.ConservativeDebit(ctx, d); err != nil {
		t.Fatalf("historical identical replay failed: %v", err)
	}
	s, _ = e.Status(ctx, "a")
	if s.Settled != 0 || s.ConservativeDebited != 0 {
		t.Fatal("historical replay charged fresh grant")
	}
}

func TestExternalDebitCeilingsAndBlockersArePreserved(t *testing.T) {
	for _, kind := range []string{"operating", "daily", "canary", "unrelated-blocker"} {
		t.Run(kind, func(t *testing.T) {
			e, _ := fixture(t)
			ctx := context.Background()
			_ = e.SetEnabled(ctx, "a", false, "operator", "stop")
			_ = e.store.Run(ctx, "a", func(tx Transaction) error {
				a, err := getAccount(tx)
				if err != nil {
					return err
				}
				day := Day{}
				switch kind {
				case "operating":
					a.Settled = 190 * USD
				case "daily":
					day.Settled = 6 * USD
				case "canary":
					a.CanarySettled = CanaryCeiling
				case "unrelated-blocker":
					a.ReconciliationRequired = true
					a.ReconciliationReason = "billing mismatch"
				}
				if err := tx.Put("days", e.now().UTC().Format("2006-01-02"), day); err != nil {
					return err
				}
				return tx.Put("account", "current", a)
			})
			d := ExternalDebit{AccountID: "a", GrantID: "g1", DebitID: "diagnostic", Amount: 1, Kind: "conservative", Operator: "operator", Evidence: "retained"}
			before, _ := e.Status(ctx, "a")
			err := e.ExternalDebit(ctx, d)
			after, _ := e.Status(ctx, "a")
			if kind == "unrelated-blocker" {
				if err != nil || !after.ReconciliationRequired || after.ReconciliationReason != "billing mismatch" {
					t.Fatalf("external debit cleared unrelated blocker: %+v %v", after, err)
				}
			} else if !errors.Is(err, ErrBlocked) || before.Settled != after.Settled || after.ConservativeDebited != 0 {
				t.Fatalf("unsafe external cap: before=%+v after=%+v err=%v", before, after, err)
			}
		})
	}
}

type failingWrites struct {
	Store
	failAt int
}

func (s failingWrites) Run(ctx context.Context, id string, fn func(Transaction) error) error {
	return s.Store.Run(ctx, id, func(tx Transaction) error { return fn(&failingTransaction{Transaction: tx, failAt: s.failAt}) })
}

type failingTransaction struct {
	Transaction
	writes, failAt int
}

func (t *failingTransaction) Put(collection, id string, value any) error {
	t.writes++
	if t.writes == t.failAt {
		return errors.New("injected storage write failure")
	}
	return t.Transaction.Put(collection, id, value)
}

func TestRecoveryWriteFailureCannotPartiallyCommit(t *testing.T) {
	for _, kind := range []string{"request", "external"} {
		t.Run(kind, func(t *testing.T) {
			e, _, d := unresolvedRecovery(t)
			ctx := context.Background()
			before, _ := e.Status(ctx, "a")
			failed := New(failingWrites{Store: e.store, failAt: 2}, e.now)
			var err error
			if kind == "request" {
				err = failed.ConservativeDebit(ctx, d)
			} else {
				err = failed.ExternalDebit(ctx, ExternalDebit{AccountID: "a", GrantID: "g1", DebitID: "outside", Amount: 1, Kind: "conservative", Operator: "operator", Evidence: "retained"})
			}
			if err == nil {
				t.Fatal("injected write error ignored")
			}
			after, _ := e.Status(ctx, "a")
			if before.Settled != after.Settled || before.Reserved != after.Reserved || before.Unresolved != after.Unresolved || before.CanaryReserved != after.CanaryReserved || after.ConservativeDebited != 0 {
				t.Fatalf("partial debit committed: before=%+v after=%+v", before, after)
			}
		})
	}
}

func TestRecoveryWhitespaceActorOrEvidenceCannotCreateAudit(t *testing.T) {
	for _, kind := range []string{"request-actor", "request-evidence", "external-actor", "external-evidence"} {
		t.Run(kind, func(t *testing.T) {
			e, _, d := unresolvedRecovery(t)
			ctx := context.Background()
			before, _ := e.Status(ctx, "a")
			external := ExternalDebit{AccountID: "a", GrantID: "g1", DebitID: "outside", Amount: 1, Kind: "conservative", Operator: "operator", Evidence: "retained"}
			var err error
			switch kind {
			case "request-actor":
				d.Operator = " \t\n"
				err = e.ConservativeDebit(ctx, d)
			case "request-evidence":
				d.Evidence = " \t\n"
				err = e.ConservativeDebit(ctx, d)
			case "external-actor":
				external.Operator = " \t\n"
				err = e.ExternalDebit(ctx, external)
			case "external-evidence":
				external.Evidence = " \t\n"
				err = e.ExternalDebit(ctx, external)
			}
			if err == nil {
				t.Fatal("blank audit field accepted")
			}
			after, _ := e.Status(ctx, "a")
			if before.Settled != after.Settled || before.Reserved != after.Reserved {
				t.Fatalf("blank evidence mutated account: before=%+v after=%+v", before, after)
			}
		})
	}
}
