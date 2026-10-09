package types

import (
	"github.com/sunholo-data/ailang/internal/typedast"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
)

// applicationEffectRow keeps a known callee's instantiated row at the App
// constraint. Minting another row loses callback/result sharing before solving.
// Unknown callees still need a fresh row; their equality determines it later.
func (ctx *InferenceContext) applicationEffectRow(callee Type) *Row {
	if fn, ok := callee.(*TFunc2); ok {
		if fn.ImplicitEffectContract || fn.ConcreteEffectContract || fn.EffectRow == nil || fn.EffectRow.Tail == nil {
			return ctx.freshEffectRow()
		}
		if fn.EffectRow != nil {
			ctx.defaultReturnOnlyRow(fn)
			return fn.EffectRow
		}
		return EmptyEffectRow()
	}
	return ctx.freshEffectRow()
}

// ApplicationEffects is the single publication authority for a typed App.
// CallRow excludes evaluation of arguments; the mask records pre-solve openness.
type ApplicationEffects struct {
	CallbackRows      []*Row
	ImplicitCallbacks []bool
	ImplicitOwnedTail bool
	LatentParamMask   []bool
	CallRow           *Row
	CallTailOwned     bool
	owned             map[string]bool
	implicitOwned     map[string]bool
	finalized         bool
	ownedTypes        []Type
	closingRows       Substitution
	callbackTypes     []Type
}

func (tc *CoreTypeChecker) publishApplication(id uint64, mask []bool, row *Row, owned map[string]bool) {
	tc.publishLatentMask(id, mask)
	record := tc.applicationEffects[id]
	record.CallRow = cloneApplicationRow(row)
	record.owned = owned
	tc.applicationEffects[id] = record
}

func (tc *CoreTypeChecker) ApplicationEffects(id uint64) (ApplicationEffects, bool) {
	record, ok := tc.applicationEffects[id]
	record.LatentParamMask = append([]bool(nil), record.LatentParamMask...)
	record.CallRow = cloneApplicationRow(record.CallRow)
	if record.CallRow != nil && record.CallRow.Tail != nil {
		record.CallTailOwned = record.owned[record.CallRow.Tail.Name]
	}
	record.CallbackRows = cloneCallbackRows(record.CallbackRows)
	record.ImplicitCallbacks = append([]bool(nil), record.ImplicitCallbacks...)
	return record, ok
}

func (tc *CoreTypeChecker) substituteApplicationRows(sub Substitution) {
	for id, record := range tc.applicationEffects {
		if record.finalized {
			continue
		}
		record.resolveOwnedTypes(sub)
		safe := publicationSubstitution(sub, record.owned)
		for name, closed := range record.closingRows {
			resolved := zonkApplicationRow(sub, &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: EffectRow}})
			if resolved != nil && resolved.Tail != nil {
				safe[resolved.Tail.Name] = closed
			}
			safe[name] = closed
		}
		record.CallRow = zonkApplicationRow(safe, record.CallRow)
		for i, row := range record.CallbackRows {
			if row == nil && i < len(record.callbackTypes) {
				if fn, ok := ApplySubstitution(sub, record.callbackTypes[i]).(*TFunc2); ok {
					row = fn.EffectRow
				}
			}
			record.CallbackRows[i] = zonkApplicationRow(safe, row)
			if r := record.CallbackRows[i]; r != nil && r.Tail != nil && record.implicitOwned[r.Tail.Name] {
				record.ImplicitCallbacks[i] = true
			}
		}
		if record.CallRow != nil && record.CallRow.Tail != nil && record.implicitOwned[record.CallRow.Tail.Name] {
			record.ImplicitOwnedTail = true
		}
		tc.applicationEffects[id] = record
	}
}

func (tc *CoreTypeChecker) applyApplicationSubstitution(sub Substitution) {
	tc.CoreTI.ApplySubstitution(sub)
	tc.substituteApplicationRows(sub)
	for id, record := range tc.applicationEffects {
		record.finalized = true
		tc.applicationEffects[id] = record
	}
}

func cloneApplicationRow(row *Row) *Row {
	if row == nil {
		return nil
	}
	clone := *row
	if row.Tail != nil {
		tail := *row.Tail
		clone.Tail = &tail
	}
	clone.Provenance = make(map[string]ast.Span, len(row.Provenance))
	for effect, span := range row.Provenance {
		clone.Provenance[effect] = span
	}
	clone.Labels = make(map[string]Type, len(row.Labels))
	for k, v := range row.Labels {
		clone.Labels[k] = v
	}
	clone.Budgets = cloneApplicationBudgets(row.Budgets)
	clone.MinBudgets = cloneApplicationBudgets(row.MinBudgets)
	clone.Params = make(map[string]map[string]string, len(row.Params))
	for effect, params := range row.Params {
		clone.Params[effect] = make(map[string]string, len(params))
		for k, v := range params {
			clone.Params[effect][k] = v
		}
	}
	return &clone
}

func cloneApplicationBudgets(src map[string]*int) map[string]*int {
	if src == nil {
		return nil
	}
	dst := make(map[string]*int, len(src))
	for k, v := range src {
		if v != nil {
			value := *v
			dst[k] = &value
		}
	}
	return dst
}

// A quantified row used only in the result has a minimal empty instantiation.
// Rows owned by an enclosing binder or shared with a parameter are never defaulted.
func (ctx *InferenceContext) defaultReturnOnlyRow(fn *TFunc2) {
	row := fn.EffectRow
	if row == nil || row.Tail == nil {
		return
	}
	name := row.Tail.Name
	if ctx.env != nil && ctx.env.effectRowsAbove(ctx.baseEnv)[name] {
		return
	}
	if freeEffectRowVarsInType(fn.Return)[name] {
		return
	}
	for _, param := range fn.Params {
		if freeEffectRowVarsInType(param)[name] {
			return
		}
	}
	closed := cloneApplicationRow(row)
	closed.Tail = nil
	ctx.addConstraint(TypeEq{Left: row, Right: closed, Path: []string{"minimal return-only application effect row"}})
}

func (env *TypeEnv) effectRowsAbove(base *TypeEnv) map[string]bool {
	free := make(map[string]bool)
	if base == nil {
		return env.FreeEffectRowVars()
	}
	for e := env; e != base; e = e.parent {
		if e == nil {
			return make(map[string]bool)
		}
		(&TypeEnv{bindings: e.bindings}).collectFreeEffectRowVars(free)
	}
	return free
}

func (tc *CoreTypeChecker) publishApplicationWithCallee(id uint64, mask []bool, row *Row, ctx *InferenceContext, args []typedast.TypedNode) {
	owned := ctx.env.effectRowsAbove(ctx.baseEnv)
	tc.publishApplication(id, mask, row, owned)
	record := tc.applicationEffects[id]
	record.implicitOwned = ctx.env.implicitRowsAbove(ctx.baseEnv)
	for e := ctx.env; e != nil && e != ctx.baseEnv; e = e.parent {
		for _, binding := range e.bindings {
			if typ, ok := binding.(Type); ok {
				record.ownedTypes = append(record.ownedTypes, typ)
			}
		}
	}
	record.CallbackRows = make([]*Row, len(args))
	record.callbackTypes = make([]Type, len(args))
	record.ImplicitCallbacks = make([]bool, len(args))
	for i, arg := range args {
		record.callbackTypes[i] = getType(arg)
		if fn, ok := tc.effectAliasHead(getType(arg)).(*TFunc2); ok {
			record.CallbackRows[i] = cloneApplicationRow(fn.EffectRow)
			record.ImplicitCallbacks[i] = fn.EffectRow != nil && fn.EffectRow.Tail != nil && record.implicitOwned[fn.EffectRow.Tail.Name]
		}
	}
	tc.applicationEffects[id] = record

	if row != nil && row.Tail != nil && ctx.env.implicitRowsAbove(ctx.baseEnv)[row.Tail.Name] {
		record := tc.applicationEffects[id]
		record.ImplicitOwnedTail = true
		tc.applicationEffects[id] = record
	}
}

func (env *TypeEnv) implicitRowsAbove(base *TypeEnv) map[string]bool {
	rows := make(map[string]bool)
	for e := env; e != nil && e != base; e = e.parent {
		for _, binding := range e.bindings {
			if fn, ok := binding.(*TFunc2); ok && fn.ImplicitEffectContract && fn.EffectRow != nil && fn.EffectRow.Tail != nil {
				rows[fn.EffectRow.Tail.Name] = true
			}
		}
	}
	return rows
}

func cloneCallbackRows(rows []*Row) []*Row {
	result := make([]*Row, len(rows))
	for i, row := range rows {
		result[i] = cloneApplicationRow(row)
	}
	return result
}

// Row unification can orient an owned row alias toward a fresh solver variable.
// Re-anchor that alias to its binder; never publish the local solver's temporary
// identity or let a later unrelated instantiation bind the generic declaration.
func publicationSubstitution(sub Substitution, owned map[string]bool) Substitution {
	safe := make(Substitution, len(sub))
	for name, value := range sub {
		if !owned[name] {
			safe[name] = value
		}
	}
	names := make([]string, 0, len(owned))
	for name := range owned {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		row := &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: EffectRow}}
		resolved := zonkApplicationRow(sub, row)
		if resolved != nil && len(resolved.Labels) == 0 && resolved.Tail != nil && resolved.Tail.Name != name {
			if owned[resolved.Tail.Name] && resolved.Tail.Name < name {
				safe[name] = &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: resolved.Tail}
			} else {
				safe[resolved.Tail.Name] = row
			}
		}
	}
	return safe
}

// The structural substitution API performs one row-tail replacement. Publication
// needs a zonked snapshot, including chains introduced by row equality replay.
func zonkApplicationRow(sub Substitution, row *Row) *Row {
	result := cloneApplicationRow(row)
	seen := make(map[string]bool)
	for result != nil && result.Tail != nil && !seen[result.Tail.Name] {
		name := result.Tail.Name
		replacement, ok := sub[name]
		if !ok {
			break
		}
		seen[name] = true
		previous := result
		result = result.Substitute(map[string]Type{name: replacement}).(*Row)
		result.MinBudgets = cloneApplicationBudgets(previous.MinBudgets)
		result.Provenance = previous.Provenance
	}
	return cloneApplicationRow(result)
}

func applicationCallRow(callee Type, inferred *Row) *Row {
	if fn, ok := callee.(*TFunc2); ok {
		if fn.ConcreteEffectContract {
			row := cloneApplicationRow(fn.EffectRow)
			if row != nil {
				row.Tail = nil
			}
			return row
		}
		if fn.EffectRow == nil {
			return EmptyEffectRow()
		}
		if fn.ImplicitEffectContract || fn.EffectRow.Tail == nil {
			return fn.EffectRow
		}
	}
	return inferred
}

func (record *ApplicationEffects) resolveOwnedTypes(sub Substitution) {
	for _, typ := range record.ownedTypes {
		resolved := ApplySubstitution(sub, typ)
		_, inferred := typ.(*TVar2)
		if fn, ok := resolved.(*TFunc2); ok && fn.ImplicitEffectContract {
			inferred = true
		}
		// Curried callback binders own the latent rows in their return functions too.
		for name := range freeEffectRowVarsInType(resolved) {
			row := zonkApplicationRow(sub, &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: EffectRow}})
			if row != nil && row.Tail != nil {
				record.owned[row.Tail.Name] = true
				if inferred {
					record.implicitOwned[row.Tail.Name] = true
				}
			}
		}
	}
}

func (tc *CoreTypeChecker) registerApplicationClosure(nodeID uint64, solved Substitution) {
	raw, ok := tc.CoreTI.Get(nodeID)
	if !ok {
		return
	}
	rawSub := closeDeclaredEffectRow(raw, tc.effectAnnotsFull[nodeID])
	for id, record := range tc.applicationEffects {
		if record.finalized {
			continue
		}
		if record.closingRows == nil {
			record.closingRows = make(Substitution)
		}
		for name, closed := range rawSub {
			// Keep concrete effects from the solved closure, including ghost effects.
			for _, value := range solved {
				closed = value
				break
			}
			record.closingRows[name] = closed
		}
		tc.applicationEffects[id] = record
	}
}

// Return annotations and explicit effect rows bind generic rows just as callback
// parameter annotations do. Put those binders in the lambda's type-only scope.
func (tc *CoreTypeChecker) lambdaRowOwnerEnv(env *TypeEnv, id uint64) (*TypeEnv, error) {
	owners := freeEffectRowVarsInType(tc.returnTypeAnnots[id])
	row, err := ElaborateEffectRowWithBudgets(tc.effectAnnotsFull[id])
	if err != nil {
		return nil, err
	}
	if row != nil && row.Tail != nil {
		owners[row.Tail.Name] = true
	}
	if len(owners) == 0 {
		return env, nil
	}
	labels := make([]string, 0, len(owners))
	for name := range owners {
		labels = append(labels, name)
	}
	sort.Strings(labels)
	for _, name := range labels {
		env = env.Extend("$effect-owner-"+name, &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: EffectRow}})
	}
	return env, nil
}
