package types

import "sort"

// Lambda open-on-use (#1958 re-land, F-B; Phase 2a of
// M-EFFECT-ROW-OPEN-CLOSED-SOUNDNESS).
//
// A closed row means "exactly these effects". An unannotated lambda whose body
// row is closed would therefore not fit a callback slot that permits more,
// e.g. `withDir(\d. 1)` against `f: string -> int ! {IO}`, because row
// unification is equality and the closed side cannot absorb the slot's IO.
// Koka's answer is to open such a row on use: the lambda gets `{L | ρ}` with a
// fresh ρ, which a wider slot binds and a closed pure slot still rejects
// (an IO lambda `{IO | ρ}` cannot meet `{}`).
//
// A tail that nothing constrains must not outlive its declaration: the
// ApplicationEffects publication record treats unowned tails as an internal
// invariant failure. closeOpenedLambdaTails binds every still-unbound terminal
// of an opened tail to closed {} before the final substitution is published.
// Explicitly annotated lambdas keep their declared row.

// openLambdaRow returns row opened with a fresh tail when it is closed.
func (ctx *InferenceContext) openLambdaRow(row *Row) *Row {
	if row != nil && row.Tail != nil {
		return row
	}
	open := ctx.freshEffectRow()
	if row != nil {
		for name, label := range row.Labels {
			open.Labels[name] = label
		}
		open.Budgets, open.MinBudgets = row.Budgets, row.MinBudgets
		open.Params, open.Provenance = row.Params, row.Provenance
	}
	if ctx.openedLambdaTails == nil {
		ctx.openedLambdaTails = make(map[string]bool)
	}
	ctx.openedLambdaTails[open.Tail.Name] = true
	return open
}

// closeOpenedLambdaTails extends sub so that every opened lambda tail still
// unbound after solving is closed to {}. It mutates and returns sub.
func (ctx *InferenceContext) closeOpenedLambdaTails(sub Substitution) Substitution {
	if len(ctx.openedLambdaTails) == 0 {
		return sub
	}
	if sub == nil {
		sub = make(Substitution)
	}
	for _, name := range sortedKeys(ctx.openedLambdaTails) {
		terminal := zonkApplicationRow(sub, &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: EffectRow}})
		if terminal == nil || terminal.Tail == nil {
			continue
		}
		if _, bound := sub[terminal.Tail.Name]; !bound {
			sub[terminal.Tail.Name] = &Row{Kind: EffectRow, Labels: map[string]Type{}}
		}
	}
	return sub
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
