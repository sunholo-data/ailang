package creditbudget

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
)

func validID(id string) bool {
	return id != "" && len(id) <= 256 && !strings.Contains(id, "/") && id != "." && id != ".."
}
func blocked(reason string) error { return fmt.Errorf("%w: %s", ErrBlocked, reason) }
func getAccount(tx Transaction) (Account, error) {
	var a Account
	ok, err := tx.Get("account", "current", &a)
	if err != nil {
		return a, err
	}
	if !ok {
		return a, ErrNotFound
	}
	return a, nil
}
func (e *Engine) Configure(ctx context.Context, p Policy) error {
	if !validID(p.AccountID) || p.Organization == "" || p.Workspace == "" || p.Enabled || p.OperatingCeiling <= 0 || p.OperatingCeiling > MaximumGrant || p.DailyCeiling <= 0 || p.DailyCeiling > p.OperatingCeiling || p.TaskCeiling <= 0 || p.TaskCeiling > p.DailyCeiling || p.MaxTasks < 1 || p.MaxTasks > 2 {
		return fmt.Errorf("invalid credit policy")
	}
	return e.store.Run(ctx, p.AccountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil && err != ErrNotFound {
			return err
		}
		if err == nil {
			existing := a.Policy
			existing.Enabled = false
			if !reflect.DeepEqual(existing, p) {
				return fmt.Errorf("%w: existing policy differs; startup cannot overwrite policy", ErrConflict)
			}
			return nil
		}
		a.Policy = p
		if a.Leases == nil {
			a.Leases = map[string]time.Time{}
		}
		return tx.Put("account", "current", a)
	})
}
func (e *Engine) SetEnabled(ctx context.Context, id string, enabled bool, operator, evidence string) error {
	if !validID(id) || operator == "" || evidence == "" {
		return fmt.Errorf("operator and evidence required")
	}
	now := e.now().UTC()
	auditID := "enable-" + uuid.NewString()
	return e.store.Run(ctx, id, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		a.Enabled = enabled
		if err := tx.Put("audit", auditID, Audit{Operator: operator, Evidence: evidence, Action: "set-enabled", At: now, Enabled: enabled}); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}
func sameGrant(g Grant, c Confirmation) bool {
	old := g.Confirmation
	old.Operator = ""
	old.Evidence = ""
	old.ExpectedGrantID = ""
	c.Operator = ""
	c.Evidence = ""
	c.ExpectedGrantID = ""
	return reflect.DeepEqual(old, c)
}
func (e *Engine) Confirm(ctx context.Context, c Confirmation) (Status, error) {
	now := e.now().UTC()
	if !validID(c.AccountID) || !validID(c.GrantID) || c.Amount <= 0 || c.Amount > MaximumGrant || c.Available <= 0 || c.Available > c.Amount || c.Operator == "" || c.Evidence == "" || c.StartsAt.IsZero() || !c.ExpiresAt.After(c.StartsAt) {
		return Status{}, fmt.Errorf("invalid credit confirmation")
	}
	var result Status
	err := e.store.Run(ctx, c.AccountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		var previous Grant
		exists, err := tx.Get("grants", c.GrantID, &previous)
		if err != nil {
			return err
		}
		var day Day
		if _, err := tx.Get("days", now.Format("2006-01-02"), &day); err != nil {
			return err
		}
		if exists {
			if a.Grant.GrantID != c.GrantID || !sameGrant(previous, c) {
				return ErrConflict
			}
			result = admissionStatus(a, day, now)
			return nil
		}
		if c.ExpectedGrantID != a.Grant.GrantID {
			return fmt.Errorf("%w: expected grant does not match current grant", ErrConflict)
		}
		if c.StartsAt.After(now) || !c.ExpiresAt.After(now.Add(ExpiryBuffer)) {
			return fmt.Errorf("new grant is not currently usable")
		}
		if a.Grant.GrantID != "" && (c.StartsAt.Before(a.Grant.StartsAt) || !c.ExpiresAt.After(a.Grant.ExpiresAt)) {
			return fmt.Errorf("%w: grant cycle does not advance", ErrConflict)
		}
		a.Grant = Grant{Confirmation: c, ConfirmedAt: now}
		a.Settled = 0
		a.ConservativeDebited = 0
		// Outstanding exposure, disabled state, and reconciliation survive renewal.
		if err := tx.Put("grants", c.GrantID, a.Grant); err != nil {
			return err
		}
		if err := tx.Put("account", "current", a); err != nil {
			return err
		}
		result = admissionStatus(a, day, now)
		return nil
	})
	return result, err
}
func status(a Account, now time.Time) Status {
	s := Status{Account: a}
	ceiling := a.OperatingCeiling
	if a.Grant.Available < ceiling {
		ceiling = a.Grant.Available
	}
	if a.Settled <= ceiling && a.Reserved <= ceiling-a.Settled {
		s.Available = ceiling - a.Settled - a.Reserved
	}
	if !a.Enabled {
		s.Blockers = append(s.Blockers, "account disabled")
	}
	if a.Grant.GrantID == "" {
		s.Blockers = append(s.Blockers, "fresh credits unconfirmed")
	} else if now.Before(a.Grant.StartsAt) || !now.Add(ExpiryBuffer).Before(a.Grant.ExpiresAt) {
		s.Blockers = append(s.Blockers, "grant expired or insufficient expiry headroom")
	}
	if a.ReconciliationRequired {
		s.Blockers = append(s.Blockers, "reconciliation required: "+a.ReconciliationReason)
	}
	if s.Available <= 0 {
		s.Blockers = append(s.Blockers, "allowance exhausted")
	}
	if a.CanarySettled >= 0 && a.CanaryReserved >= 0 && a.CanarySettled <= CanaryCeiling && a.CanaryReserved <= CanaryCeiling-a.CanarySettled {
		s.CanaryAvailable = CanaryCeiling - a.CanarySettled - a.CanaryReserved
	}
	if !a.CanaryComplete && s.CanaryAvailable <= 0 {
		s.Blockers = append(s.Blockers, "aggregate canary allowance exhausted")
	}
	s.Eligible = len(s.Blockers) == 0
	return s
}

func admissionStatus(a Account, day Day, now time.Time) Status {
	s := status(a, now)
	s.Day = day
	for _, until := range a.Leases {
		if until.After(now) {
			s.ActiveTasks++
		}
	}
	if !fits(day.Settled, day.Reserved, 1, a.DailyCeiling) {
		s.Blockers = append(s.Blockers, "daily allowance exhausted")
	}
	if s.ActiveTasks >= a.MaxTasks {
		s.Blockers = append(s.Blockers, "concurrent task limit")
	}
	s.Eligible = len(s.Blockers) == 0
	return s
}
func (e *Engine) Status(ctx context.Context, id string) (Status, error) {
	if !validID(id) {
		return Status{}, fmt.Errorf("invalid account ID")
	}
	var s Status
	err := e.store.Run(ctx, id, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		now := e.now().UTC()
		var day Day
		if _, err := tx.Get("days", now.Format("2006-01-02"), &day); err != nil {
			return err
		}
		s = admissionStatus(a, day, now)
		return nil
	})
	return s, err
}
func fits(settled, reserved, amount, ceiling MicroUSD) bool {
	return settled >= 0 && reserved >= 0 && amount > 0 && settled <= ceiling && reserved <= ceiling-settled && amount <= ceiling-settled-reserved
}
func (e *Engine) AdmitTask(ctx context.Context, t Task) error {
	now := e.now().UTC()
	if !validID(t.AccountID) || !validID(t.ID) || !validID(t.AttemptID) || t.JobIdentity == "" || len(t.Models) == 0 || t.Ceiling <= 0 || t.Settled != 0 || t.Reserved != 0 || t.ConservativeDebited != 0 || t.Released || !t.LeaseUntil.After(now) {
		return fmt.Errorf("invalid task admission")
	}
	return e.store.Run(ctx, t.AccountID, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		var old Task
		exists, err := tx.Get("tasks", t.ID, &old)
		if err != nil {
			return err
		}
		if exists {
			binding := old
			binding.Settled = 0
			binding.Reserved = 0
			binding.ConservativeDebited = 0
			if reflect.DeepEqual(binding, t) {
				return nil
			}
			if old.AttemptID == t.AttemptID {
				return ErrConflict
			}
			if !old.Released && old.LeaseUntil.After(now) {
				return blocked("previous task attempt remains active")
			}
			if old.Ceiling != t.Ceiling || !reflect.DeepEqual(old.Models, t.Models) {
				return fmt.Errorf("%w: retry cannot change task ceiling or allowed models", ErrConflict)
			}
			t.Settled = old.Settled
			t.Reserved = old.Reserved
			t.ConservativeDebited = old.ConservativeDebited
		}
		s := status(a, now)
		if !s.Eligible {
			return blocked(strings.Join(s.Blockers, ", "))
		}
		if t.Ceiling > a.TaskCeiling {
			return blocked("task ceiling exceeds account policy")
		}
		for id, until := range a.Leases {
			if !until.After(now) {
				delete(a.Leases, id)
			}
		}
		if len(a.Leases) >= a.MaxTasks {
			return blocked("concurrent task limit")
		}
		a.Leases[t.ID] = t.LeaseUntil
		if err := tx.Put("tasks", t.ID, t); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}
func (e *Engine) ReleaseTask(ctx context.Context, id, taskID string) error {
	return e.releaseTask(ctx, id, taskID, "", "")
}

func (e *Engine) ReleaseAttempt(ctx context.Context, id, taskID, attemptID, jobIdentity string) error {
	if attemptID == "" || jobIdentity == "" {
		return fmt.Errorf("attempt identity required")
	}
	return e.releaseTask(ctx, id, taskID, attemptID, jobIdentity)
}

func (e *Engine) releaseTask(ctx context.Context, id, taskID, attemptID, jobIdentity string) error {
	if !validID(id) || !validID(taskID) {
		return fmt.Errorf("invalid task ID")
	}
	return e.store.Run(ctx, id, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		var t Task
		ok, err := tx.Get("tasks", taskID, &t)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if attemptID != "" && (t.AttemptID != attemptID || t.JobIdentity != jobIdentity) {
			return fmt.Errorf("%w: attempt release identity mismatch", ErrConflict)
		}
		t.Released = true
		delete(a.Leases, taskID)
		if err := tx.Put("tasks", taskID, t); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}

// CompleteCanary records the operator's verified provider reconciliation. It
// only removes the pilot sub-limit: enabling, grant confirmation, and accounting
// reconciliation remain independent controls. Promotion never resets spending.
func (e *Engine) CompleteCanary(ctx context.Context, id, operator, evidence string) error {
	if !validID(id) || operator == "" || evidence == "" {
		return fmt.Errorf("operator and provider reconciliation evidence required")
	}
	now := e.now().UTC()
	auditID := "canary-" + uuid.NewString()
	return e.store.Run(ctx, id, func(tx Transaction) error {
		a, err := getAccount(tx)
		if err != nil {
			return err
		}
		if a.CanaryComplete {
			return nil
		}
		if a.ReconciliationRequired || a.Reserved != 0 || a.Forwarding != 0 || a.Unresolved != 0 || a.CanaryReserved != 0 || a.CanaryGatewaySettled <= 0 {
			return blocked("canary requires settled spend and no uncertain or pending exposure")
		}
		a.CanaryComplete = true
		if err := tx.Put("audit", auditID, Audit{Operator: operator, Evidence: evidence, Action: "complete-canary", At: now, Enabled: a.Enabled}); err != nil {
			return err
		}
		return tx.Put("account", "current", a)
	})
}
