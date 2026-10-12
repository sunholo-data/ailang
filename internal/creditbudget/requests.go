package creditbudget

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

func (e *Engine) Reserve(ctx context.Context, r Reservation) (Request, error) {
	now := e.now().UTC()
	var result Request
	if !validID(r.AccountID) || !validID(r.TaskID) || !validID(r.RequestID) || r.AttemptID == "" || r.JobIdentity == "" || r.Model == "" || r.Amount <= 0 || r.Amount > MaximumGrant || r.PricingRevision == "" || !r.Deadline.After(now) {
		return result, fmt.Errorf("invalid credit reservation")
	}
	err := e.store.Run(ctx, r.AccountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		var t Task
		taskExists, err := tx.Get("tasks", r.TaskID, &t)
		if err != nil {
			return err
		}
		var previous Request
		exists, err := tx.Get("requests", r.RequestID, &previous)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%w: request ID already used", ErrConflict)
		}
		dayID := now.Format("2006-01-02")
		var day Day
		if _, err := tx.Get("days", dayID, &day); err != nil {
			return err
		}
		s := status(a, now)
		if !s.Eligible {
			return blocked(strings.Join(s.Blockers, ", "))
		}
		if !taskExists || t.Released || !t.LeaseUntil.After(now) || !a.Leases[r.TaskID].After(now) || t.AttemptID != r.AttemptID || t.JobIdentity != r.JobIdentity || !slices.Contains(t.Models, r.Model) {
			return blocked("task identity, model or lease invalid")
		}
		if !r.Deadline.Before(t.LeaseUntil) || !r.Deadline.Add(ExpiryBuffer).Before(a.Grant.ExpiresAt) {
			return blocked("request deadline lacks lease or credit expiry headroom")
		}
		if r.Amount > s.Available || !fits(t.Settled, t.Reserved, r.Amount, t.Ceiling) || !fits(day.Settled, day.Reserved, r.Amount, a.DailyCeiling) {
			return blocked("account, task or daily spending ceiling")
		}
		if !a.CanaryComplete && !fits(a.CanarySettled, a.CanaryReserved, r.Amount, CanaryCeiling) {
			return blocked("aggregate canary allowance exhausted")
		}
		a.Reserved += r.Amount
		t.Reserved += r.Amount
		day.Reserved += r.Amount
		result = Request{Reservation: r, GrantID: a.Grant.GrantID, Day: dayID, State: "reserved", CreatedAt: now}
		result.Canary = !a.CanaryComplete
		if result.Canary {
			a.CanaryReserved += r.Amount
		}
		if err := tx.Put("requests", r.RequestID, result); err != nil {
			return err
		}
		if err := tx.Put("tasks", r.TaskID, t); err != nil {
			return err
		}
		if err := tx.Put("days", dayID, day); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
	return result, err
}
func getRequest(tx Transaction, id string) (Request, error) {
	var r Request
	ok, err := tx.Get("requests", id, &r)
	if err != nil {
		return r, err
	}
	if !ok {
		return r, ErrNotFound
	}
	return r, nil
}
func (e *Engine) MarkForwarded(ctx context.Context, accountID, requestID string) error {
	if !validID(accountID) || !validID(requestID) {
		return fmt.Errorf("invalid request ID")
	}
	return e.store.Run(ctx, accountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		r, err := getRequest(tx, requestID)
		if err != nil {
			return err
		}
		if r.State != "reserved" {
			return fmt.Errorf("%w: send intent already recorded or request closed", ErrConflict)
		}
		var task Task
		if ok, err := tx.Get("tasks", r.TaskID, &task); err != nil {
			return err
		} else if !ok {
			return ErrNotFound
		}
		now := e.now().UTC()
		// A fully reserved final allowance has zero *new* admission capacity,
		// but still authorizes this already-reserved send. Recheck kill switch,
		// reconciliation and validity, rather than requiring free money twice.
		if !a.Enabled || a.ReconciliationRequired || a.Grant.GrantID == "" || now.Before(a.Grant.StartsAt) || !now.Add(ExpiryBuffer).Before(a.Grant.ExpiresAt) || !r.Deadline.After(now) || task.Released || !task.LeaseUntil.After(now) || !a.Leases[r.TaskID].After(now) {
			return blocked("account, deadline or task became unavailable before send")
		}
		// Intent is durable before the transport starts. A process crash immediately
		// after this write is ambiguous and must never be retried with this ID.
		r.State = "forwarding"
		a.Forwarding += r.Amount
		if err := tx.Put("requests", requestID, r); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}
func (e *Engine) MarkUnresolved(ctx context.Context, accountID, requestID string) error {
	if !validID(accountID) || !validID(requestID) {
		return fmt.Errorf("invalid request ID")
	}
	return e.store.Run(ctx, accountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		r, err := getRequest(tx, requestID)
		if err != nil {
			return err
		}
		if r.State == "unresolved" {
			return nil
		}
		if r.State != "forwarding" {
			return ErrConflict
		}
		r.State = "unresolved"
		if a.Forwarding < r.Amount {
			return fmt.Errorf("corrupt forwarding exposure")
		}
		a.Forwarding -= r.Amount
		a.Unresolved += r.Amount
		a.ReconciliationRequired = true
		if a.ReconciliationReason == "" {
			a.ReconciliationReason = "upstream outcome unknown"
		}
		if err := tx.Put("requests", requestID, r); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}
func (e *Engine) Settle(ctx context.Context, accountID, requestID string, actual MicroUSD, upstreamID string) error {
	if !validID(accountID) || !validID(requestID) || actual < 0 || upstreamID == "" {
		return fmt.Errorf("invalid settlement")
	}
	overage := false
	conflicting := false
	err := e.store.Run(ctx, accountID, func(tx Transaction) error {
		overage = false
		conflicting = false
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		r, err := getRequest(tx, requestID)
		if err != nil {
			return err
		}
		if r.State == "settled" {
			if r.Actual == actual && r.UpstreamID == upstreamID {
				return nil
			}
			a.ReconciliationRequired = true
			a.ReconciliationReason = fmt.Sprintf("conflicting settlement for %s: settled=%d reported=%d micro-USD", requestID, r.Actual, actual)
			conflicting = true
			return tx.Put("account", "current", a)
		}
		if r.State != "forwarding" && r.State != "unresolved" {
			return ErrConflict
		}
		var t Task
		ok, err := tx.Get("tasks", r.TaskID, &t)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		var day Day
		ok, err = tx.Get("days", r.Day, &day)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if a.Reserved < r.Amount || t.Reserved < r.Amount || day.Reserved < r.Amount {
			return fmt.Errorf("corrupt credit accounting")
		}
		if actual > r.Amount {
			a.ReconciliationRequired = true
			a.ReconciliationReason = "actual cost exceeds verified request reservation"
			if r.State != "unresolved" {
				if a.Forwarding < r.Amount {
					return fmt.Errorf("corrupt forwarding exposure")
				}
				a.Forwarding -= r.Amount
				a.Unresolved += r.Amount
			}
			r.State = "unresolved"
			r.Actual = actual
			r.UpstreamID = upstreamID
			overage = true
		} else {
			a.Reserved -= r.Amount
			a.Settled += actual
			t.Reserved -= r.Amount
			t.Settled += actual
			day.Reserved -= r.Amount
			day.Settled += actual
			if r.Canary {
				if a.CanaryReserved < r.Amount {
					return fmt.Errorf("corrupt canary exposure")
				}
				a.CanaryReserved -= r.Amount
				a.CanarySettled += actual
				a.CanaryGatewaySettled += actual
			}
			if r.State == "unresolved" {
				a.Unresolved -= r.Amount
			} else {
				if a.Forwarding < r.Amount {
					return fmt.Errorf("corrupt forwarding exposure")
				}
				a.Forwarding -= r.Amount
			}
			if a.Unresolved == 0 && a.ReconciliationReason == "upstream outcome unknown" {
				a.ReconciliationRequired = false
				a.ReconciliationReason = ""
			}
			r.State = "settled"
			r.Actual = actual
			r.UpstreamID = upstreamID
		}
		if err := tx.Put("requests", requestID, r); err != nil {
			return err
		}
		if err := tx.Put("tasks", r.TaskID, t); err != nil {
			return err
		}
		if err := tx.Put("days", r.Day, day); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
	if err != nil {
		return err
	}
	if overage {
		return blocked("settlement exceeds reservation; reconciliation required")
	}
	if conflicting {
		return fmt.Errorf("%w: conflicting settlement requires reconciliation", ErrConflict)
	}
	return nil
}
func (e *Engine) ReleaseUnsent(ctx context.Context, accountID, requestID string) error {
	if !validID(accountID) || !validID(requestID) {
		return fmt.Errorf("invalid request ID")
	}
	return e.store.Run(ctx, accountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		r, err := getRequest(tx, requestID)
		if err != nil {
			return err
		}
		if r.State == "released" {
			return nil
		}
		if r.State != "reserved" {
			return ErrConflict
		}
		var t Task
		ok, err := tx.Get("tasks", r.TaskID, &t)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		var day Day
		ok, err = tx.Get("days", r.Day, &day)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if a.Reserved < r.Amount || t.Reserved < r.Amount || day.Reserved < r.Amount {
			return fmt.Errorf("corrupt credit accounting")
		}
		a.Reserved -= r.Amount
		if r.Canary {
			if a.CanaryReserved < r.Amount {
				return fmt.Errorf("corrupt canary exposure")
			}
			a.CanaryReserved -= r.Amount
		}
		t.Reserved -= r.Amount
		day.Reserved -= r.Amount
		r.State = "released"
		if err := tx.Put("requests", requestID, r); err != nil {
			return err
		}
		if err := tx.Put("tasks", r.TaskID, t); err != nil {
			return err
		}
		if err := tx.Put("days", r.Day, day); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}
