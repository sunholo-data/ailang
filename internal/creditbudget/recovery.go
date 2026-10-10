package creditbudget

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const RequestInspectionLimit = 100

func recoveryAllowed(a Account, grantID string, now time.Time) error {
	if a.Enabled {
		return blocked("recovery requires disabled account")
	}
	if a.Grant.GrantID != grantID {
		return fmt.Errorf("%w: recovery grant is not current", ErrConflict)
	}
	for _, until := range a.Leases {
		if until.After(now) {
			return blocked("recovery requires no active task leases")
		}
	}
	if a.Forwarding != 0 {
		return blocked("recovery requires no forwarding exposure")
	}
	return nil
}
func replayDebit(tx Transaction, id string, wanted DebitAudit) (bool, error) {
	var existing DebitAudit
	ok, err := tx.Get("debits", id, &existing)
	if err != nil || !ok {
		return false, err
	}
	existing.At = time.Time{}
	wanted.At = time.Time{}
	if !reflect.DeepEqual(existing, wanted) {
		return false, fmt.Errorf("%w: immutable debit audit differs", ErrConflict)
	}
	return true, nil
}

// ConservativeDebit replaces an unresolved reservation with the full original
// booked amount. It never claims a provider receipt or creates new capacity.
func (e *Engine) ConservativeDebit(ctx context.Context, d ConservativeDebit) error {
	if !validID(d.AccountID) || !validID(d.GrantID) || !validID(d.RequestID) || d.ExpectedAmount <= 0 || d.ExpectedAmount > MaximumGrant || strings.TrimSpace(d.Operator) == "" || strings.TrimSpace(d.Evidence) == "" {
		return fmt.Errorf("invalid conservative request debit")
	}
	now := e.now().UTC()
	auditID := "request-" + d.RequestID
	audit := DebitAudit{Action: "request-conservative-debit", AccountID: d.AccountID, GrantID: d.GrantID, RequestID: d.RequestID, Kind: "conservative", Amount: d.ExpectedAmount, Operator: d.Operator, Evidence: d.Evidence, At: now}
	return e.store.Run(ctx, d.AccountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		if repeated, err := replayDebit(tx, auditID, audit); err != nil || repeated {
			return err
		}
		r, err := getRequest(tx, d.RequestID)
		if err != nil {
			return err
		}
		if !validID(r.TaskID) || !validID(r.Day) || r.AccountID != d.AccountID || r.RequestID != d.RequestID {
			return fmt.Errorf("corrupt recovery request identity")
		}
		var task Task
		exists, err := tx.Get("tasks", r.TaskID, &task)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		var day Day
		exists, err = tx.Get("days", r.Day, &day)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		if err := recoveryAllowed(a, d.GrantID, now); err != nil {
			return err
		}
		if !a.ReconciliationRequired || a.ReconciliationReason != "upstream outcome unknown" || r.State != "unresolved" || r.Actual > r.Amount {
			return blocked("only ordinary unresolved upstream outcomes permit conservative recovery")
		}
		if r.Amount != d.ExpectedAmount {
			return fmt.Errorf("%w: expected amount differs from full reservation", ErrConflict)
		}
		if a.Reserved < r.Amount || a.Unresolved < r.Amount || task.Reserved < r.Amount || day.Reserved < r.Amount || a.Settled < 0 || task.Settled < 0 || day.Settled < 0 {
			return fmt.Errorf("corrupt recovery accounting")
		}
		if r.Canary && a.CanaryReserved < r.Amount {
			return fmt.Errorf("corrupt canary recovery accounting")
		}
		a.Reserved -= r.Amount
		a.Unresolved -= r.Amount
		a.Settled += r.Amount
		a.ConservativeDebited += r.Amount
		task.Reserved -= r.Amount
		task.Settled += r.Amount
		task.ConservativeDebited += r.Amount
		day.Reserved -= r.Amount
		day.Settled += r.Amount
		day.ConservativeDebited += r.Amount
		if r.Canary {
			a.CanaryReserved -= r.Amount
			a.CanarySettled += r.Amount
		}
		// Actual/UpstreamID and the original request/grant/pricing remain untouched.
		r.State = "conservative-debit"
		if a.Unresolved == 0 && a.Forwarding == 0 {
			a.ReconciliationRequired = false
			a.ReconciliationReason = ""
		}
		if err := tx.Put("requests", d.RequestID, r); err != nil {
			return err
		}
		if err := tx.Put("tasks", r.TaskID, task); err != nil {
			return err
		}
		if err := tx.Put("days", r.Day, day); err != nil {
			return err
		}
		if err := tx.Put("debits", auditID, audit); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}

// ExternalDebit books an attended diagnostic or retained external receipt. The
// provider-verified kind is an operator attestation, never gateway canary proof.
func (e *Engine) ExternalDebit(ctx context.Context, d ExternalDebit) error {
	if !validID(d.AccountID) || !validID(d.GrantID) || !validID(d.DebitID) || d.Amount <= 0 || d.Amount > MaximumGrant || (d.Kind != "conservative" && d.Kind != "provider-verified") || strings.TrimSpace(d.Operator) == "" || strings.TrimSpace(d.Evidence) == "" {
		return fmt.Errorf("invalid external debit")
	}
	now := e.now().UTC()
	dayID := now.Format("2006-01-02")
	auditID := "external-" + d.DebitID
	audit := DebitAudit{Action: "external-debit", AccountID: d.AccountID, GrantID: d.GrantID, DebitID: d.DebitID, Kind: d.Kind, Amount: d.Amount, Operator: d.Operator, Evidence: d.Evidence, At: now}
	return e.store.Run(ctx, d.AccountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		if repeated, err := replayDebit(tx, auditID, audit); err != nil || repeated {
			return err
		}
		var day Day
		if _, err := tx.Get("days", dayID, &day); err != nil {
			return err
		}
		if err := recoveryAllowed(a, d.GrantID, now); err != nil {
			return err
		}
		ceiling := a.OperatingCeiling
		if a.Grant.Available < ceiling {
			ceiling = a.Grant.Available
		}
		if !fits(a.Settled, a.Reserved, d.Amount, ceiling) || !fits(day.Settled, day.Reserved, d.Amount, a.DailyCeiling) || (!a.CanaryComplete && !fits(a.CanarySettled, a.CanaryReserved, d.Amount, CanaryCeiling)) {
			return blocked("external debit exceeds operating, daily or canary allowance")
		}
		a.Settled += d.Amount
		day.Settled += d.Amount
		if !a.CanaryComplete {
			a.CanarySettled += d.Amount
		}
		if d.Kind == "conservative" {
			a.ConservativeDebited += d.Amount
			day.ConservativeDebited += d.Amount
		}
		if err := tx.Put("days", dayID, day); err != nil {
			return err
		}
		if err := tx.Put("debits", auditID, audit); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}

func (e *Engine) Requests(ctx context.Context, accountID, taskID string) ([]Request, error) {
	if !validID(accountID) || !validID(taskID) {
		return nil, fmt.Errorf("valid account and task IDs required")
	}
	reader, ok := e.store.(RequestReader)
	if !ok {
		return nil, fmt.Errorf("credit store does not support bounded request inspection")
	}
	if err := e.store.Run(ctx, accountID, func(tx Transaction) error { _, err := getAccount(tx); return err }); err != nil {
		return nil, err
	}
	return reader.ListRequests(ctx, accountID, taskID, RequestInspectionLimit)
}
