package creditbudget

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T, daily ...MicroUSD) (*Engine, time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	e := New(NewMemoryStore(), func() time.Time { return now })
	ctx := context.Background()
	dayLimit := 6 * USD
	if len(daily) > 0 {
		dayLimit = daily[0]
	}
	if err := e.Configure(ctx, Policy{AccountID: "a", Organization: "org", Workspace: "ws", Enabled: false, OperatingCeiling: 190 * USD, DailyCeiling: dayLimit, TaskCeiling: 2 * USD, MaxTasks: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Confirm(ctx, Confirmation{AccountID: "a", GrantID: "g1", Amount: 200 * USD, Available: 200 * USD, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(24 * time.Hour), Operator: "operator", Evidence: "console"}); err != nil {
		t.Fatal(err)
	}
	if err := e.SetEnabled(ctx, "a", true, "operator", "reviewed-test-activation"); err != nil {
		t.Fatal(err)
	}
	return e, now
}
func admit(t *testing.T, e *Engine, now time.Time, id string) {
	t.Helper()
	if err := e.AdmitTask(context.Background(), Task{AccountID: "a", ID: id, AttemptID: "attempt", JobIdentity: "job@example", Models: []string{"claude-haiku-5-5"}, Ceiling: 2 * USD, LeaseUntil: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
}
func reservation(now time.Time, id, task string, amount MicroUSD) Reservation {
	return Reservation{AccountID: "a", TaskID: task, AttemptID: "attempt", JobIdentity: "job@example", Model: "claude-haiku-5-5", RequestID: id, Amount: amount, PricingRevision: "verified", Deadline: now.Add(time.Minute)}
}
func TestConcurrentReservationCannotExceedTaskAndSharedLimits(t *testing.T) {
	e, now := fixture(t)
	admit(t, e, now, "t")
	var wg sync.WaitGroup
	success := make(chan struct{}, 10)
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.Reserve(context.Background(), reservation(now, string(rune('a'+i)), "t", USD)); err == nil {
				success <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(success)
	if len(success) != 2 {
		t.Fatalf("successful reservations=%d, want 2", len(success))
	}
	s, err := e.Status(context.Background(), "a")
	if err != nil || s.Reserved != 2*USD {
		t.Fatalf("status=%+v err=%v", s, err)
	}
}
func TestForwardingCannotRefundAndDuplicateSettlementIsHarmless(t *testing.T) {
	e, now := fixture(t)
	admit(t, e, now, "t")
	ctx := context.Background()
	if _, err := e.Reserve(ctx, reservation(now, "r", "t", USD)); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkForwarded(ctx, "a", "r"); err != nil {
		t.Fatal(err)
	}
	if err := e.ReleaseUnsent(ctx, "a", "r"); err == nil {
		t.Fatal("forwarded reservation refunded")
	}
	if err := e.MarkUnresolved(ctx, "a", "r"); err != nil {
		t.Fatal(err)
	}
	if err := e.ReleaseTask(ctx, "a", "t"); err != nil {
		t.Fatal(err)
	}
	s, _ := e.Status(ctx, "a")
	if s.Reserved != USD || s.Unresolved != USD {
		t.Fatalf("lost uncertain exposure: %+v", s)
	}
	for range 2 {
		if err := e.Settle(ctx, "a", "r", USD/4, "upstream"); err != nil {
			t.Fatal(err)
		}
	}
	s, _ = e.Status(ctx, "a")
	if s.Settled != USD/4 || s.Reserved != 0 || s.Unresolved != 0 {
		t.Fatalf("non-idempotent settlement: %+v", s)
	}
	if err := e.Settle(ctx, "a", "r", USD/2, "upstream"); err == nil {
		t.Fatal("conflicting settlement accepted")
	}
	s, _ = e.Status(ctx, "a")
	if !s.ReconciliationRequired {
		t.Fatal("conflicting billed usage did not freeze admission")
	}
}
func TestRenewalPreservesExposureAndReconfirmationCannotAddCredits(t *testing.T) {
	e, now := fixture(t)
	admit(t, e, now, "t")
	ctx := context.Background()
	_, _ = e.Reserve(ctx, reservation(now, "r", "t", USD))
	_ = e.MarkForwarded(ctx, "a", "r")
	c := Confirmation{AccountID: "a", GrantID: "g2", ExpectedGrantID: "g1", Amount: 200 * USD, Available: 200 * USD, StartsAt: now, ExpiresAt: now.Add(48 * time.Hour), Operator: "operator", Evidence: "fresh-console"}
	s, err := e.Confirm(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if s.Reserved != USD {
		t.Fatal("renewal discarded exposure")
	}
	_ = e.Settle(ctx, "a", "r", USD/2, "old-request")
	s, err = e.Confirm(ctx, c)
	if err != nil || s.Settled != USD/2 {
		t.Fatalf("repeat added allowance: %+v %v", s, err)
	}
	c.Available = 199 * USD
	if _, err = e.Confirm(ctx, c); err == nil {
		t.Fatal("conflicting grant repeat accepted")
	}
}
func TestExpiryDisabledAndConcurrencyFailClosed(t *testing.T) {
	e, now := fixture(t)
	admit(t, e, now, "t1")
	admit(t, e, now, "t2")
	err := e.AdmitTask(context.Background(), Task{AccountID: "a", ID: "t3", AttemptID: "a", JobIdentity: "j", Models: []string{"m"}, Ceiling: USD, LeaseUntil: now.Add(time.Hour)})
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("third task: %v", err)
	}
	if _, err = e.Reserve(context.Background(), reservation(now.Add(24*time.Hour-6*time.Minute), "near-expiry", "t1", USD)); err == nil {
		t.Fatal("deadline without expiry headroom admitted")
	}
	if err = e.SetEnabled(context.Background(), "a", false, "operator", "disable"); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Reserve(context.Background(), reservation(now, "disabled", "t1", USD)); !errors.Is(err, ErrBlocked) {
		t.Fatalf("disabled: %v", err)
	}
}
func TestAboveReservationFreezesAndRetainsExposure(t *testing.T) {
	e, now := fixture(t)
	admit(t, e, now, "t")
	ctx := context.Background()
	_, _ = e.Reserve(ctx, reservation(now, "r", "t", USD))
	_ = e.MarkForwarded(ctx, "a", "r")
	if err := e.Settle(ctx, "a", "r", 2*USD, "up"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("above reservation: %v", err)
	}
	s, _ := e.Status(ctx, "a")
	if s.Eligible || s.Reserved != USD || !s.ReconciliationRequired {
		t.Fatalf("unsafe freeze: %+v", s)
	}
}

func TestUnsentReleaseOnlyAndIdentityMismatch(t *testing.T) {
	e, now := fixture(t)
	admit(t, e, now, "t")
	ctx := context.Background()
	wrong := reservation(now, "wrong", "t", USD)
	wrong.JobIdentity = "another-job"
	if _, err := e.Reserve(ctx, wrong); err == nil {
		t.Fatal("another job identity admitted")
	}
	wrong = reservation(now, "wrong", "t", USD)
	wrong.AttemptID = "another-attempt"
	if _, err := e.Reserve(ctx, wrong); err == nil {
		t.Fatal("another attempt admitted")
	}
	wrong = reservation(now, "wrong", "t", USD)
	wrong.Model = "unapproved"
	if _, err := e.Reserve(ctx, wrong); err == nil {
		t.Fatal("unapproved model admitted")
	}
	r := reservation(now, "r", "t", USD)
	if _, err := e.Reserve(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := e.ReleaseUnsent(ctx, "a", "r"); err != nil {
		t.Fatal(err)
	}
	if err := e.ReleaseUnsent(ctx, "a", "r"); err != nil {
		t.Fatal(err)
	}
	s, _ := e.Status(ctx, "a")
	if s.Reserved != 0 {
		t.Fatal("unsent reservation not released")
	}
	if _, err := e.Reserve(ctx, r); !errors.Is(err, ErrConflict) {
		t.Fatalf("request ID reused: %v", err)
	}
}
func TestDailyCeilingAndLeaseExpiryNeverRefund(t *testing.T) {
	e, now := fixture(t, 4*USD)
	ctx := context.Background()
	admit(t, e, now, "first")
	_, err := e.Reserve(ctx, reservation(now, "r", "first", 2*USD))
	if err != nil {
		t.Fatal(err)
	}
	_ = e.MarkForwarded(ctx, "a", "r")
	clock := now.Add(2 * time.Hour)
	e.now = func() time.Time { return clock }
	admit(t, e, clock, "second")
	_, err = e.Reserve(ctx, reservation(clock, "r2", "second", 2*USD))
	if err != nil {
		t.Fatal(err)
	}
	_ = e.MarkForwarded(ctx, "a", "r2")
	_ = e.ReleaseTask(ctx, "a", "second")
	admit(t, e, clock, "third")
	if _, err = e.Reserve(ctx, reservation(clock, "r3", "third", USD)); !errors.Is(err, ErrBlocked) {
		t.Fatalf("daily ceiling: %v", err)
	}
	s, _ := e.Status(ctx, "a")
	if s.Reserved != 4*USD || s.Day.Reserved != 4*USD {
		t.Fatalf("expired lease discarded reservations: %+v", s)
	}
}

func TestHistoricalGrantReplayAndStaleConfirmation(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	old := Confirmation{AccountID: "a", GrantID: "g1", Amount: 200 * USD, Available: 200 * USD, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(24 * time.Hour), Operator: "operator", Evidence: "console"}
	fresh := Confirmation{AccountID: "a", GrantID: "g2", ExpectedGrantID: "wrong", Amount: 200 * USD, Available: 200 * USD, StartsAt: now, ExpiresAt: now.Add(48 * time.Hour), Operator: "operator", Evidence: "fresh"}
	if _, err := e.Confirm(ctx, fresh); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale confirmation: %v", err)
	}
	fresh.ExpectedGrantID = "g1"
	if _, err := e.Confirm(ctx, fresh); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Confirm(ctx, old); !errors.Is(err, ErrConflict) {
		t.Fatalf("historical replay: %v", err)
	}
}
func TestMarkForwardedIsSingleSendPermission(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "t")
	_, _ = e.Reserve(ctx, reservation(now, "r", "t", USD))
	if err := e.MarkForwarded(ctx, "a", "r"); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkForwarded(ctx, "a", "r"); !errors.Is(err, ErrConflict) {
		t.Fatalf("second send authorized: %v", err)
	}
}

type unavailableStore struct{}

func (unavailableStore) Run(context.Context, string, func(Transaction) error) error {
	return errors.New("storage unavailable")
}
func TestStorageFailureCannotAuthorize(t *testing.T) {
	e := New(unavailableStore{}, nil)
	if _, err := e.Status(context.Background(), "a"); err == nil {
		t.Fatal("storage unavailable returned eligible status")
	}
	if _, err := e.Reserve(context.Background(), reservation(time.Now(), "r", "t", USD)); err == nil {
		t.Fatal("storage unavailable admitted request")
	}
}

func TestRestartCannotEnableOrResetCredits(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "t")
	_, _ = e.Reserve(ctx, reservation(now, "r", "t", USD))
	_ = e.MarkForwarded(ctx, "a", "r")
	_ = e.MarkUnresolved(ctx, "a", "r")
	if err := e.SetEnabled(ctx, "a", false, "operator", "kill-switch"); err != nil {
		t.Fatal(err)
	}
	p := Policy{AccountID: "a", Organization: "org", Workspace: "ws", Enabled: false, OperatingCeiling: 190 * USD, DailyCeiling: 6 * USD, TaskCeiling: 2 * USD, MaxTasks: 2}
	if err := e.Configure(ctx, p); err != nil {
		t.Fatal(err)
	}
	s, _ := e.Status(ctx, "a")
	if s.Enabled || s.Reserved != USD || s.Grant.GrantID != "g1" || !s.ReconciliationRequired {
		t.Fatalf("restart changed authority: %+v", s)
	}
	p.Enabled = true
	if err := e.Configure(ctx, p); err == nil {
		t.Fatal("configure can enable account")
	}
	p.Enabled = false
	p.DailyCeiling = 10 * USD
	if err := e.Configure(ctx, p); !errors.Is(err, ErrConflict) {
		t.Fatalf("config restart overwrote policy: %v", err)
	}
}
func TestKillSwitchAfterReservationPreventsForwarding(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "t")
	_, _ = e.Reserve(ctx, reservation(now, "r", "t", USD))
	if err := e.SetEnabled(ctx, "a", false, "operator", "stop-now"); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkForwarded(ctx, "a", "r"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("disabled lane can forward reserved request: %v", err)
	}
	if err := e.ReleaseUnsent(ctx, "a", "r"); err != nil {
		t.Fatal(err)
	}
}
func TestRenewalPreservesUnknownExposureAndDisabledState(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "t")
	_, _ = e.Reserve(ctx, reservation(now, "r", "t", USD))
	_ = e.MarkForwarded(ctx, "a", "r")
	_ = e.MarkUnresolved(ctx, "a", "r")
	_ = e.SetEnabled(ctx, "a", false, "operator", "kill")
	c := Confirmation{AccountID: "a", GrantID: "g2", ExpectedGrantID: "g1", Amount: 200 * USD, Available: 200 * USD, StartsAt: now, ExpiresAt: now.Add(48 * time.Hour), Operator: "operator", Evidence: "fresh"}
	s, err := e.Confirm(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if s.Enabled || s.Eligible || s.Reserved != USD || s.Unresolved != USD || !s.ReconciliationRequired {
		t.Fatalf("renewal discarded blocker: %+v", s)
	}
}

func TestCrashAfterSendIntentRemainsVisibleThroughRestartAndRenewal(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "t")
	_, _ = e.Reserve(ctx, reservation(now, "r", "t", USD))
	_ = e.MarkForwarded(ctx, "a", "r")
	// A new engine sees the durable send intent even if the original process
	// never called MarkUnresolved and no result arrived.
	resumed := New(e.store, func() time.Time { return now.Add(2 * time.Hour) })
	s, _ := resumed.Status(ctx, "a")
	if s.Forwarding != USD || s.Reserved != USD {
		t.Fatalf("lost crash exposure: %+v", s)
	}
	c := Confirmation{AccountID: "a", GrantID: "g2", ExpectedGrantID: "g1", Amount: 200 * USD, Available: 200 * USD, StartsAt: now, ExpiresAt: now.Add(48 * time.Hour), Operator: "operator", Evidence: "fresh"}
	s, err := resumed.Confirm(ctx, c)
	if err != nil || s.Forwarding != USD || s.Reserved != USD {
		t.Fatalf("renewal lost crashed request: %+v %v", s, err)
	}
	if err := resumed.ReleaseUnsent(ctx, "a", "r"); !errors.Is(err, ErrConflict) {
		t.Fatalf("crashed request refunded: %v", err)
	}
}
func TestExpiredSameCycleReconfirmationCannotReplenish(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	e.now = func() time.Time { return now.Add(48 * time.Hour) }
	c := Confirmation{AccountID: "a", GrantID: "g1", Amount: 200 * USD, Available: 200 * USD, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(24 * time.Hour), Operator: "operator", Evidence: "console"}
	s, err := e.Confirm(ctx, c)
	if err != nil || s.Eligible || s.Grant.GrantID != "g1" {
		t.Fatalf("expired cycle restored: %+v %v", s, err)
	}
}

func TestRetriedAttemptPreservesTaskExposureAndRejectsOldIdentity(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "t")
	_, _ = e.Reserve(ctx, reservation(now, "old", "t", USD))
	_ = e.MarkForwarded(ctx, "a", "old")
	retry := Task{AccountID: "a", ID: "t", AttemptID: "retry", JobIdentity: "next-job", Models: []string{"claude-haiku-5-5"}, Ceiling: 2 * USD, LeaseUntil: now.Add(2 * time.Hour)}
	if err := e.AdmitTask(ctx, retry); !errors.Is(err, ErrBlocked) {
		t.Fatalf("active attempt replaced: %v", err)
	}
	_ = e.ReleaseTask(ctx, "a", "t")
	if err := e.AdmitTask(ctx, retry); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Reserve(ctx, reservation(now, "old-job", "t", USD/2)); !errors.Is(err, ErrBlocked) {
		t.Fatalf("old identity admitted: %v", err)
	}
	next := reservation(now, "new-job", "t", USD)
	next.AttemptID = "retry"
	next.JobIdentity = "next-job"
	if _, err := e.Reserve(ctx, next); err != nil {
		t.Fatal(err)
	}
	next.RequestID = "too-much"
	next.Amount = 1
	if _, err := e.Reserve(ctx, next); !errors.Is(err, ErrBlocked) {
		t.Fatalf("retry discarded original task exposure: %v", err)
	}
}

func TestCanaryAggregateCeilingAndVerifiedPromotion(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "one")
	admit(t, e, now, "two")
	_, _ = e.Reserve(ctx, reservation(now, "r1", "one", 2*USD))
	_ = e.MarkForwarded(ctx, "a", "r1")
	_, _ = e.Reserve(ctx, reservation(now, "r2", "two", 2*USD))
	_ = e.MarkForwarded(ctx, "a", "r2")
	_ = e.ReleaseTask(ctx, "a", "one")
	admit(t, e, now, "three")
	_, err := e.Reserve(ctx, reservation(now, "r3", "three", USD))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Reserve(ctx, reservation(now, "over-smoke", "three", USD/2)); !errors.Is(err, ErrBlocked) {
		t.Fatalf("canary exceeded aggregate $5: %v", err)
	}
	if err = e.CompleteCanary(ctx, "a", "operator", "billing"); err == nil {
		t.Fatal("promoted with pending exposure")
	}
	_ = e.MarkForwarded(ctx, "a", "r3")
	_ = e.Settle(ctx, "a", "r1", USD/4, "up1")
	_ = e.Settle(ctx, "a", "r2", USD/4, "up2")
	_ = e.Settle(ctx, "a", "r3", USD/4, "up3")
	if err = e.CompleteCanary(ctx, "a", "operator", "provider-reconciled"); err != nil {
		t.Fatal(err)
	}
	if err = e.CompleteCanary(ctx, "a", "operator", "provider-reconciled"); err != nil {
		t.Fatal(err)
	}
	s, _ := e.Status(ctx, "a")
	if !s.CanaryComplete || s.CanarySettled != 3*USD/4 || s.CanaryReserved != 0 {
		t.Fatalf("promotion reset canary accounting: %+v", s)
	}
}
func TestCanaryCannotPromoteWithoutSpendAndRenewalCannotResetIt(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	if err := e.CompleteCanary(ctx, "a", "operator", "no-spend"); err == nil {
		t.Fatal("unexecuted canary promoted")
	}
	admit(t, e, now, "t")
	_, _ = e.Reserve(ctx, reservation(now, "r", "t", USD))
	_ = e.MarkForwarded(ctx, "a", "r")
	c := Confirmation{AccountID: "a", GrantID: "g2", ExpectedGrantID: "g1", Amount: 200 * USD, Available: 200 * USD, StartsAt: now, ExpiresAt: now.Add(48 * time.Hour), Operator: "operator", Evidence: "fresh"}
	s, err := e.Confirm(ctx, c)
	if err != nil || s.CanaryComplete || s.CanaryReserved != USD {
		t.Fatalf("renewal reset canary: %+v %v", s, err)
	}
}
func TestStaleAttemptCannotReleaseRetriedLease(t *testing.T) {
	e, now := fixture(t)
	ctx := context.Background()
	admit(t, e, now, "t")
	if err := e.ReleaseAttempt(ctx, "a", "t", "attempt", "job@example"); err != nil {
		t.Fatal(err)
	}
	retry := Task{AccountID: "a", ID: "t", AttemptID: "retry", JobIdentity: "next-job", Models: []string{"claude-haiku-5-5"}, Ceiling: 2 * USD, LeaseUntil: now.Add(time.Hour)}
	if err := e.AdmitTask(ctx, retry); err != nil {
		t.Fatal(err)
	}
	if err := e.ReleaseAttempt(ctx, "a", "t", "attempt", "job@example"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale capability released retry: %v", err)
	}
	s, _ := e.Status(ctx, "a")
	if s.ActiveTasks != 1 {
		t.Fatalf("retry lease removed: %+v", s)
	}
}
