package effects

import (
	"fmt"
	"sync"
)

// M-EXECUTOR-POLICY-HARDENING M4 (D5) — the operator budget.
//
// Source budgets (@limit on a signature, BudgetFrames) are PER INVOCATION:
// every call of an annotated function gets a fresh frame, so a program can
// perform any number of operations by calling itself. The operator's
// `[budgets]` in the policy is a different quantity: an aggregate ceiling
// on the whole run. It is charged once per logical effect operation, at the
// canonical charge scope (the same place the frame charge dedups the wrapper
// and the impl), BEFORE the operation runs; it survives WithBudget scopes,
// imports, repeated calls and async paths because every derived context
// shares this one pointer; `--no-budgets` never touches it; source
// annotations can only add a tighter per-invocation limit on top.
//
// Semantics: a label present with 0 permits zero operations; a label absent
// from the map is unlimited. An absent capability still denies regardless.

// OperatorBudget is the shared, mutex-guarded per-effect ceiling.
type OperatorBudget struct {
	mu     sync.Mutex
	limits map[string]int
	used   map[string]int
}

// NewOperatorBudget copies limits (present = ceiling, absent = unlimited).
func NewOperatorBudget(limits map[string]int) *OperatorBudget {
	b := &OperatorBudget{limits: make(map[string]int, len(limits)), used: map[string]int{}}
	for k, v := range limits {
		b.limits[k] = v
	}
	return b
}

// OperatorBudgetError is the denial: the run's ceiling for one effect is spent.
type OperatorBudgetError struct {
	Effect   string
	Limit    int
	Position string
}

func (e *OperatorBudgetError) Error() string {
	msg := fmt.Sprintf("E_BUDGET_OPERATOR: effect '%s' operator budget exhausted: limit=%d for the whole run", e.Effect, e.Limit)
	if e.Position != "" {
		msg += " at " + e.Position
	}
	return msg + "\nHint: the ceiling comes from the policy's [budgets]; a source @limit cannot raise it"
}

// Charge consumes one unit for effect, or denies when the ceiling is spent.
// nil when the effect has no operator ceiling.
func (b *OperatorBudget) Charge(effect, position string) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	limit, ok := b.limits[effect]
	if !ok {
		return nil
	}
	if b.used[effect] >= limit {
		return &OperatorBudgetError{Effect: effect, Limit: limit, Position: position}
	}
	b.used[effect]++
	return nil
}

// Usage is a copy of the per-effect counters (for the banked report).
func (b *OperatorBudget) Usage() map[string]int {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make(map[string]int, len(b.used))
	for k, v := range b.used {
		out[k] = v
	}
	return out
}

// SetOperatorBudget installs the run's operator ceiling. The pointer is
// shared by every context derived afterwards (WithBudget, Clone).
func (ctx *EffContext) SetOperatorBudget(b *OperatorBudget) { ctx.operatorBudget = b }

// OperatorBudgetUsage reports the run's operator-budget counters (nil when
// no operator budget is installed).
func (ctx *EffContext) OperatorBudgetUsage() map[string]int { return ctx.operatorBudget.Usage() }
