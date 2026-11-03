package types

// Effect-row union with operand resolution (#1958 re-land, F-A1 and F-A2).
//
// Effect rows of sibling sub-expressions are combined bottom-up while their
// operands can still be bare tails: inferApp leaves a callee's row unsubstituted,
// so `{ println(x); println(x) }` meets as `{...ε9} ∪ {...ε11}`. combineEffects
// cannot merge two distinct tails and closes the row, so the lambda published
// `{}` with no labels. That made an IO lambda look pure to a callback slot
// whenever its body made more than one call.
//
// unionEffects resolves such operands first, against the equality constraints
// added since the enclosing lambda began. It follows the same rules as #386's
// application-local solve in inferApp: bindings of variables owned by an
// enclosing scope are dropped, and an operand is replaced only when it resolves
// to a closed row. One exception is fixpoint absorption: an operand whose tail
// is the effect row of a let-rec binding being inferred is its own recursive
// call, and contributes only its labels (rho = L ∪ rho has least solution
// rho = L). A callback parameter's row is not a self row and is never absorbed.

// unionFrame is the resolution scope of one lambda body. Its local solve is
// cached and extended with only the constraints added since the last union,
// so a body with n unions costs one pass over its constraints, not n.
type unionFrame struct {
	checkpoint int             // first constraint added inside the lambda
	env        *TypeEnv        // the lambda's environment at entry (persistent)
	outer      map[string]bool // its type/row vars, computed on first use
	solvedTo   int             // constraints [checkpoint, solvedTo) are folded into sub
	sub        Substitution    // equalities solved so far (nil until first use)
	local      Substitution    // sub without the outer bindings
	failed     bool            // a local equality failed; SolveConstraints reports it
}

// solve extends the frame's cached solve to the current constraints.
func (f *unionFrame) solve(ctx *InferenceContext) bool {
	if f.failed {
		return false
	}
	if f.outer == nil {
		f.outer = envOwnedVars(f.env)
	}
	if f.sub == nil || f.solvedTo > len(ctx.constraints) {
		f.sub, f.solvedTo = make(Substitution), f.checkpoint
	}
	if f.solvedTo == len(ctx.constraints) && f.local != nil {
		return true
	}
	sub, err := ctx.solveEqualitiesInto(f.sub, ctx.constraints[f.solvedTo:])
	if err != nil {
		f.failed = true
		return false
	}
	f.sub, f.solvedTo = sub, len(ctx.constraints)
	f.local = make(Substitution, len(sub))
	for v, t := range sub {
		if !f.outer[v] {
			f.local[v] = t
		}
	}
	return true
}

// pushUnionFrame opens a lambda's resolution scope; call the result to close it.
// Walking the environment is deferred to the frame's first distinct-tail union:
// most lambdas never have one, and the walk dominated the type-check profile.
func (ctx *InferenceContext) pushUnionFrame() func() {
	ctx.unionFrames = append(ctx.unionFrames, &unionFrame{checkpoint: len(ctx.constraints), env: ctx.env})
	depth := len(ctx.unionFrames) - 1
	return func() { ctx.unionFrames = ctx.unionFrames[:depth] }
}

// pushSelfTypes records the binding types of a let-rec group being inferred.
func (ctx *InferenceContext) pushSelfTypes(types []Type) func() {
	depth := len(ctx.selfTypes)
	ctx.selfTypes = append(ctx.selfTypes, types...)
	return func() { ctx.selfTypes = ctx.selfTypes[:depth] }
}

// unionEffectList combines effect rows left to right with unionEffects.
func (ctx *InferenceContext) unionEffectList(effects []*Row) *Row {
	result := EmptyEffectRow()
	for _, e := range effects {
		result = ctx.unionEffects(result, e)
	}
	return result
}

// unionEffects is combineEffects with distinct-tail operands resolved first.
func (ctx *InferenceContext) unionEffects(e1, e2 *Row) *Row {
	if len(ctx.unionFrames) == 0 {
		return combineEffects(e1, e2)
	}
	if !hasDistinctEffectTails(e1, e2) {
		return ctx.absorbSingleSelfTail(e1, e2)
	}
	frame := ctx.unionFrames[len(ctx.unionFrames)-1]
	if !frame.solve(ctx) {
		// The enclosing SolveConstraints reports the error with its full path.
		return combineEffects(e1, e2)
	}
	return combineEffects(ctx.resolveUnionOperand(frame.sub, frame.local, e1), ctx.resolveUnionOperand(frame.sub, frame.local, e2))
}

// absorbSingleSelfTail handles a union with at most one open tail, which
// combineEffects keeps as is. When that tail is a let-rec binding's own row
// (its recursive call), it is absorbed like a distinct tail would be: otherwise
// `{ println(x); loop(n - 1) }` publishes {IO | ρ} for the row ρ it is the call
// of, and the binding's ρ ~ {IO | ρ} fails with "same row variable with
// different extensions" (#2074). Any other single tail is left alone.
func (ctx *InferenceContext) absorbSingleSelfTail(e1, e2 *Row) *Row {
	if len(ctx.selfTypes) == 0 || effectTailName(e1) == effectTailName(e2) {
		return combineEffects(e1, e2) // no let-rec in scope, or no tail / one shared tail
	}
	frame := ctx.unionFrames[len(ctx.unionFrames)-1]
	if !frame.solve(ctx) {
		return combineEffects(e1, e2)
	}
	return combineEffects(ctx.absorbSelfOperand(frame.sub, e1), ctx.absorbSelfOperand(frame.sub, e2))
}

// absorbSelfOperand drops row's tail when it resolves to a self row, keeping
// its labels; any other row is returned unchanged.
func (ctx *InferenceContext) absorbSelfOperand(full Substitution, row *Row) *Row {
	if row == nil || row.Tail == nil {
		return row
	}
	resolved := resolveEffectRow(full, row)
	if resolved == nil || resolved.Tail == nil || !ctx.isSelfEffectTail(full, resolved.Tail.Name) {
		return row
	}
	absorbed := *resolved
	absorbed.Tail = nil
	return &absorbed
}

// effectTailName is the name of row's effect-row tail, or "" when it has none.
func effectTailName(r *Row) string {
	if r == nil {
		return ""
	}
	if tv := effectTailOnly(r.Tail); tv != nil {
		return tv.Name
	}
	return ""
}

func hasDistinctEffectTails(e1, e2 *Row) bool {
	if e1 == nil || e2 == nil {
		return false
	}
	t1, t2 := effectTailOnly(e1.Tail), effectTailOnly(e2.Tail)
	return t1 != nil && t2 != nil && t1.Name != t2.Name
}

// resolveUnionOperand applies fixpoint absorption against the full local
// solution, then closeIfResolved against the solution without outer bindings.
func (ctx *InferenceContext) resolveUnionOperand(full, local Substitution, row *Row) *Row {
	if row == nil || row.Tail == nil {
		return row
	}
	resolved := resolveEffectRow(full, row)
	if resolved != nil && resolved.Tail != nil && ctx.isSelfEffectTail(full, resolved.Tail.Name) {
		absorbed := *resolved
		absorbed.Tail = nil
		return &absorbed
	}
	return closeIfResolved(row, resolveEffectRow(local, row))
}

// isSelfEffectTail reports whether tail is the (resolved) effect-row tail of a
// let-rec binding currently being inferred, or of any arrow it returns: a
// curried binding `\i. \acc. … loop(i + 1)(acc')` calls its own INNER arrow, so
// that arrow's row is the self row of the inner lambda (#2074).
//
// Cycle-safety: the walk follows Return through at most maxSelfArrowDepth
// arrows, so a cyclic type cannot loop it.
func (ctx *InferenceContext) isSelfEffectTail(sub Substitution, tail string) bool {
	for _, self := range ctx.selfTypes {
		t := ApplySubstitution(sub, self)
		for depth := 0; depth < maxSelfArrowDepth; depth++ {
			fn, ok := t.(*TFunc2)
			if !ok {
				break
			}
			if fn.EffectRow != nil {
				if row := resolveEffectRow(sub, fn.EffectRow); row != nil && row.Tail != nil && row.Tail.Name == tail {
					return true
				}
			}
			t = ApplySubstitution(sub, fn.Return)
		}
	}
	return false
}

// maxSelfArrowDepth bounds how many curried arrows isSelfEffectTail inspects.
const maxSelfArrowDepth = 16

// envOwnedVars returns the type, row and effect-row variables free in env.
// The result is shared and must not be modified.
func envOwnedVars(env *TypeEnv) map[string]bool {
	return env.ownedVars().all()
}

// resolveEffectRow resolves row's tail through every link of sub, not just one.
// The row unifier does not keep its substitution idempotent: once ρ1 ↦ {...ρ3}
// is recorded, a later ρ3 ↦ {...ρ6} leaves ρ1's entry pointing at ρ3. Each
// recursive call of a let-rec unifies its row with the previous representative,
// so the self row sits one link further away per call, and a one-step
// substitution stopped recognising it from the third call on ("same row
// variable with different extensions" on three nested branches, found in the
// v0.57.0 pre-release eval-code sweep). zonkApplicationRow follows the chain
// and is cycle-safe.
func resolveEffectRow(sub Substitution, row *Row) *Row {
	if row == nil || len(sub) == 0 {
		return row
	}
	return zonkApplicationRow(sub, row)
}
